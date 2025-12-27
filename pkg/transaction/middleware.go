package transaction

import (
	"github.com/code-100-precent/LingFramework/pkg/constants"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	// TransactionContextKey is the key used to store transaction context in Gin context
	TransactionContextKey = "_ling_transaction_context"
)

// Middleware creates a transaction middleware with the given configuration
func Middleware(db *gorm.DB, config *Config) gin.HandlerFunc {
	manager := NewManager(db)
	if config == nil {
		config = DefaultConfig()
	}

	return func(c *gin.Context) {
		// Get parent transaction context if exists
		var parent *TransactionContext
		if parentVal, exists := c.Get(TransactionContextKey); exists {
			if parentCtx, ok := parentVal.(*TransactionContext); ok {
				parent = parentCtx
			}
		}

		// Begin transaction
		tc, err := manager.Begin(parent, config)
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			return
		}

		// Store transaction context in Gin context
		c.Set(TransactionContextKey, tc)
		// Also store the transaction DB instance in the standard DB field for backward compatibility
		c.Set(constants.DbField, tc.Transaction)

		// Process request
		c.Next()

		// Check if there was an error
		var commitErr error
		if len(c.Errors) > 0 {
			// Check if we should rollback
			lastErr := c.Errors.Last()
			if tc.ShouldRollback(lastErr) {
				commitErr = manager.Rollback(tc)
			} else {
				// Error should not trigger rollback
				commitErr = manager.Commit(tc)
			}
		} else {
			// No error, commit transaction
			commitErr = manager.Commit(tc)
		}

		// If commit/rollback failed, log the error (but don't change the response)
		if commitErr != nil {
			c.Error(commitErr)
		}
	}
}

// Transactional is a convenience function that creates a transaction middleware with default config
func Transactional(db *gorm.DB) gin.HandlerFunc {
	return Middleware(db, DefaultConfig())
}

// TransactionalWithConfig creates a transaction middleware with custom configuration
func TransactionalWithConfig(db *gorm.DB, config *Config) gin.HandlerFunc {
	return Middleware(db, config)
}

// GetTransactionContext retrieves the transaction context from Gin context
func GetTransactionContext(c *gin.Context) (*TransactionContext, error) {
	val, exists := c.Get(TransactionContextKey)
	if !exists {
		return nil, ErrNoTransaction
	}

	tc, ok := val.(*TransactionContext)
	if !ok {
		return nil, ErrNoTransaction
	}

	return tc, nil
}

// GetDB retrieves the database instance from transaction context
// Falls back to the standard DB field if no transaction context exists
func GetDB(c *gin.Context) (*gorm.DB, error) {
	tc, err := GetTransactionContext(c)
	if err == nil && tc != nil {
		return tc.Transaction, nil
	}

	// Fallback to standard DB field
	dbVal, exists := c.Get(constants.DbField)
	if !exists {
		return nil, ErrNoTransaction
	}

	db, ok := dbVal.(*gorm.DB)
	if !ok {
		return nil, ErrNoTransaction
	}

	return db, nil
}

// MustGetDB retrieves the database instance, panics if not found
func MustGetDB(c *gin.Context) *gorm.DB {
	db, err := GetDB(c)
	if err != nil {
		panic(err)
	}
	return db
}
