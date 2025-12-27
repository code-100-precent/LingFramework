package transaction

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Manager manages transaction lifecycle
type Manager struct {
	db *gorm.DB
}

// NewManager creates a new transaction manager
func NewManager(db *gorm.DB) *Manager {
	return &Manager{db: db}
}

// Begin starts a new transaction based on the configuration and parent context
func (m *Manager) Begin(parent *TransactionContext, config *Config) (*TransactionContext, error) {
	if config == nil {
		config = DefaultConfig()
	}

	var tx *gorm.DB

	switch config.Propagation {
	case PropagationRequired:
		if parent != nil && parent.IsActive() {
			// Use existing transaction
			return &TransactionContext{
				Transaction: parent.Transaction,
				Config:      config,
				Parent:      parent,
			}, nil
		}
		// Create new transaction
		tx = m.db.Begin()
		if tx.Error != nil {
			return nil, tx.Error
		}

	case PropagationRequiresNew:
		// Always create a new transaction (suspend parent if exists)
		tx = m.db.Begin()
		if tx.Error != nil {
			return nil, tx.Error
		}

	case PropagationSupports:
		if parent != nil && parent.IsActive() {
			// Use existing transaction
			return &TransactionContext{
				Transaction: parent.Transaction,
				Config:      config,
				Parent:      parent,
			}, nil
		}
		// Execute without transaction
		return &TransactionContext{
			Transaction: m.db,
			Config:      config,
			Parent:      parent,
		}, nil

	case PropagationNotSupported:
		// Execute without transaction
		return &TransactionContext{
			Transaction: m.db,
			Config:      config,
			Parent:      parent,
		}, nil

	case PropagationMandatory:
		if parent == nil || !parent.IsActive() {
			return nil, ErrTransactionRequired
		}
		// Use existing transaction
		return &TransactionContext{
			Transaction: parent.Transaction,
			Config:      config,
			Parent:      parent,
		}, nil

	case PropagationNever:
		if parent != nil && parent.IsActive() {
			return nil, ErrTransactionNotAllowed
		}
		// Execute without transaction
		return &TransactionContext{
			Transaction: m.db,
			Config:      config,
			Parent:      parent,
		}, nil

	case PropagationNested:
		if parent != nil && parent.IsActive() {
			// For nested transactions, create a new transaction
			// In production with MySQL/PostgreSQL, you could use SavePoint:
			// savepointName := "sp_" + time.Now().Format("20060102150405")
			// tx = parent.Transaction.SavePoint(savepointName)
			// For SQLite compatibility, we'll use a new transaction
			tx = m.db.Begin()
			if tx.Error != nil {
				return nil, tx.Error
			}
		} else {
			// Create new transaction
			tx = m.db.Begin()
			if tx.Error != nil {
				return nil, tx.Error
			}
		}

	default:
		// Default to PropagationRequired
		if parent != nil && parent.IsActive() {
			return &TransactionContext{
				Transaction: parent.Transaction,
				Config:      config,
				Parent:      parent,
			}, nil
		}
		tx = m.db.Begin()
		if tx.Error != nil {
			return nil, tx.Error
		}
	}

	// Apply isolation level if specified
	if config.IsolationLevel != IsolationDefault {
		if err := m.setIsolationLevel(tx, config.IsolationLevel); err != nil {
			return nil, err
		}
	}

	// Apply timeout if specified
	if config.Timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Timeout)*time.Second)
		defer cancel()
		tx = tx.WithContext(ctx)
	}

	return &TransactionContext{
		Transaction: tx,
		Config:      config,
		Parent:      parent,
	}, nil
}

// setIsolationLevel sets the isolation level for the transaction
// Note: SQLite only supports SERIALIZABLE isolation level, other databases may support all levels
func (m *Manager) setIsolationLevel(tx *gorm.DB, level IsolationLevel) error {

	var isolationSQL string
	switch level {
	case IsolationReadUncommitted:
		isolationSQL = "SET TRANSACTION ISOLATION LEVEL READ UNCOMMITTED"
	case IsolationReadCommitted:
		isolationSQL = "SET TRANSACTION ISOLATION LEVEL READ COMMITTED"
	case IsolationRepeatableRead:
		isolationSQL = "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ"
	case IsolationSerializable:
		isolationSQL = "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE"
	default:
		return nil
	}

	result := tx.Exec(isolationSQL)
	// SQLite doesn't support SET TRANSACTION ISOLATION LEVEL
	// For compatibility, ignore syntax errors (which indicate SQLite)
	if result.Error != nil {
		errStr := result.Error.Error()
		// Check if error is a syntax error (SQLite doesn't support this)
		if strings.Contains(errStr, "syntax error") || strings.Contains(errStr, "near \"SET\"") {
			// SQLite doesn't support isolation level setting, ignore the error
			return nil
		}
		return result.Error
	}
	return nil
}

// Commit commits the transaction
func (m *Manager) Commit(tc *TransactionContext) error {
	if tc == nil || !tc.IsActive() {
		return nil
	}

	// If this is a nested transaction, rollback to savepoint instead of commit
	if tc.Parent != nil && tc.Parent.IsActive() && tc.Config.Propagation == PropagationNested {
		// Nested transactions don't commit, they release savepoints
		// The parent transaction will handle the commit
		return nil
	}

	// Only commit if this is a real transaction (not the original db connection)
	if tc.Transaction != nil && tc.Transaction != m.db {
		if err := tc.Transaction.Commit().Error; err != nil {
			return err
		}
		tc.Committed = true
	}

	return nil
}

// Rollback rolls back the transaction
func (m *Manager) Rollback(tc *TransactionContext) error {
	if tc == nil || !tc.IsActive() {
		return nil
	}

	// If this is a nested transaction, rollback to savepoint
	if tc.Parent != nil && tc.Parent.IsActive() && tc.Config.Propagation == PropagationNested {
		// Rollback to savepoint (the savepoint name would need to be tracked)
		// For simplicity, we'll rollback the nested transaction
		if tc.Transaction != nil {
			tc.Transaction.Rollback()
			tc.RolledBack = true
		}
		return nil
	}

	// Rollback the transaction
	if tc.Transaction != nil && tc.Transaction != m.db {
		if err := tc.Transaction.Rollback().Error; err != nil {
			return err
		}
		tc.RolledBack = true
	}

	return nil
}
