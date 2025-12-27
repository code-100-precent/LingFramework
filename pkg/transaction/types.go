package transaction

import (
	"errors"
	"gorm.io/gorm"
)

// Propagation defines transaction propagation behavior
type Propagation int

const (
	// PropagationRequired Default propagation. If a transaction exists, use it; otherwise, create a new one.
	PropagationRequired Propagation = iota
	// PropagationRequiresNew Always create a new transaction, suspending the current one if it exists.
	PropagationRequiresNew
	// PropagationSupports If a transaction exists, use it; otherwise, execute without a transaction.
	PropagationSupports
	// PropagationNotSupported Execute without a transaction, suspending the current one if it exists.
	PropagationNotSupported
	// PropagationMandatory A transaction must exist, otherwise throw an error.
	PropagationMandatory
	// PropagationNever Execute without a transaction, throw an error if a transaction exists.
	PropagationNever
	// PropagationNested Execute within a nested transaction (savepoint). If no transaction exists, create one.
	PropagationNested
)

// IsolationLevel defines transaction isolation level
type IsolationLevel int

const (
	// IsolationDefault Use the default isolation level of the database
	IsolationDefault IsolationLevel = iota
	// IsolationReadUncommitted Read uncommitted (lowest isolation level)
	IsolationReadUncommitted
	// IsolationReadCommitted Read committed
	IsolationReadCommitted
	// IsolationRepeatableRead Repeatable read
	IsolationRepeatableRead
	// IsolationSerializable Serializable (highest isolation level)
	IsolationSerializable
)

// Config represents transaction configuration
type Config struct {
	// Propagation transaction propagation behavior (default: PropagationRequired)
	Propagation Propagation
	// IsolationLevel transaction isolation level (default: IsolationDefault)
	IsolationLevel IsolationLevel
	// ReadOnly whether the transaction is read-only (default: false)
	ReadOnly bool
	// Timeout transaction timeout in seconds (0 means no timeout)
	Timeout int
	// RollbackOn list of error types that should trigger rollback
	// If empty, rollback on any error
	RollbackOn []error
	// NoRollbackOn list of error types that should NOT trigger rollback
	// Takes precedence over RollbackOn
	NoRollbackOn []error
}

// DefaultConfig returns a default transaction configuration
func DefaultConfig() *Config {
	return &Config{
		Propagation:    PropagationRequired,
		IsolationLevel: IsolationDefault,
		ReadOnly:       false,
		Timeout:        0,
		RollbackOn:     nil,
		NoRollbackOn:   nil,
	}
}

// WithPropagation sets the propagation behavior
func (c *Config) WithPropagation(p Propagation) *Config {
	c.Propagation = p
	return c
}

// WithIsolationLevel sets the isolation level
func (c *Config) WithIsolationLevel(level IsolationLevel) *Config {
	c.IsolationLevel = level
	return c
}

// WithReadOnly sets whether the transaction is read-only
func (c *Config) WithReadOnly(readOnly bool) *Config {
	c.ReadOnly = readOnly
	return c
}

// WithTimeout sets the transaction timeout
func (c *Config) WithTimeout(timeout int) *Config {
	c.Timeout = timeout
	return c
}

// WithRollbackOn sets error types that should trigger rollback
func (c *Config) WithRollbackOn(errors ...error) *Config {
	c.RollbackOn = errors
	return c
}

// WithNoRollbackOn sets error types that should NOT trigger rollback
func (c *Config) WithNoRollbackOn(errors ...error) *Config {
	c.NoRollbackOn = errors
	return c
}

// TransactionContext represents a transaction context stored in Gin context
type TransactionContext struct {
	// Transaction the GORM transaction instance
	Transaction *gorm.DB
	// Config the transaction configuration
	Config *Config
	// Parent the parent transaction context (for nested transactions)
	Parent *TransactionContext
	// Committed whether the transaction has been committed
	Committed bool
	// RolledBack whether the transaction has been rolled back
	RolledBack bool
}

// IsActive checks if the transaction is active
func (tc *TransactionContext) IsActive() bool {
	return tc != nil && !tc.Committed && !tc.RolledBack
}

// ShouldRollback checks if an error should trigger rollback
func (tc *TransactionContext) ShouldRollback(err error) bool {
	if err == nil {
		return false
	}

	// Check NoRollbackOn first (takes precedence)
	if len(tc.Config.NoRollbackOn) > 0 {
		for _, noRollbackErr := range tc.Config.NoRollbackOn {
			if errors.Is(err, noRollbackErr) {
				return false
			}
		}
	}

	// If RollbackOn is specified, only rollback on those errors
	if len(tc.Config.RollbackOn) > 0 {
		for _, rollbackErr := range tc.Config.RollbackOn {
			if errors.Is(err, rollbackErr) {
				return true
			}
		}
		return false
	}

	// Default: rollback on any error
	return true
}

var (
	// ErrNoTransaction indicates that no transaction exists in the context
	ErrNoTransaction = errors.New("no transaction in context")
	// ErrTransactionRequired indicates that a transaction is required but not found
	ErrTransactionRequired = errors.New("transaction required but not found")
	// ErrTransactionNotAllowed indicates that a transaction exists but is not allowed
	ErrTransactionNotAllowed = errors.New("transaction not allowed")
)
