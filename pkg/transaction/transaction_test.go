package transaction

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/code-100-precent/LingFramework/pkg/constants"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestUser is a test model
type TestUser struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"size:100"`
	Age  int
}

func setupTestDB(t *testing.T) *gorm.DB {
	// Disable SQL logging in tests - use silent logger
	silentLogger := logger.New(
		log.New(io.Discard, "", log.LstdFlags), // Discard output
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Silent, // Silent mode
			IgnoreRecordNotFoundError: true,          // Ignore not found errors
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: silentLogger,
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&TestUser{})
	require.NoError(t, err)

	return db
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, PropagationRequired, config.Propagation)
	assert.Equal(t, IsolationDefault, config.IsolationLevel)
	assert.False(t, config.ReadOnly)
	assert.Equal(t, 0, config.Timeout)
	assert.Nil(t, config.RollbackOn)
	assert.Nil(t, config.NoRollbackOn)
}

func TestConfigWithMethods(t *testing.T) {
	config := DefaultConfig().
		WithPropagation(PropagationRequiresNew).
		WithIsolationLevel(IsolationReadCommitted).
		WithReadOnly(true).
		WithTimeout(10)

	assert.Equal(t, PropagationRequiresNew, config.Propagation)
	assert.Equal(t, IsolationReadCommitted, config.IsolationLevel)
	assert.True(t, config.ReadOnly)
	assert.Equal(t, 10, config.Timeout)
}

func TestTransactionContext_IsActive(t *testing.T) {
	db := setupTestDB(t)
	tx := db.Begin()

	tc := &TransactionContext{
		Transaction: tx,
		Config:      DefaultConfig(),
	}
	assert.True(t, tc.IsActive())

	tc.Committed = true
	assert.False(t, tc.IsActive())

	tc.Committed = false
	tc.RolledBack = true
	assert.False(t, tc.IsActive())
}

func TestTransactionContext_ShouldRollback(t *testing.T) {
	testErr := errors.New("test error")
	noRollbackErr := errors.New("no rollback error")

	t.Run("rollback on any error by default", func(t *testing.T) {
		tc := &TransactionContext{
			Config: DefaultConfig(),
		}
		assert.True(t, tc.ShouldRollback(testErr))
	})

	t.Run("no rollback on NoRollbackOn errors", func(t *testing.T) {
		tc := &TransactionContext{
			Config: DefaultConfig().WithNoRollbackOn(noRollbackErr),
		}
		assert.False(t, tc.ShouldRollback(noRollbackErr))
		assert.True(t, tc.ShouldRollback(testErr))
	})

	t.Run("only rollback on RollbackOn errors", func(t *testing.T) {
		tc := &TransactionContext{
			Config: DefaultConfig().WithRollbackOn(testErr),
		}
		assert.True(t, tc.ShouldRollback(testErr))
		assert.False(t, tc.ShouldRollback(noRollbackErr))
	})

	t.Run("NoRollbackOn takes precedence", func(t *testing.T) {
		tc := &TransactionContext{
			Config: DefaultConfig().
				WithRollbackOn(testErr).
				WithNoRollbackOn(noRollbackErr),
		}
		// Even though testErr is in RollbackOn, if it matches NoRollbackOn, don't rollback
		assert.True(t, tc.ShouldRollback(testErr)) // Different error, so rollback
		assert.False(t, tc.ShouldRollback(noRollbackErr))
	})
}

func TestManager_Begin_PropagationRequired(t *testing.T) {
	db := setupTestDB(t)
	manager := NewManager(db)
	config := DefaultConfig().WithPropagation(PropagationRequired)

	t.Run("create new transaction when no parent", func(t *testing.T) {
		tc, err := manager.Begin(nil, config)
		require.NoError(t, err)
		require.NotNil(t, tc)
		assert.NotNil(t, tc.Transaction)
		assert.NotEqual(t, db, tc.Transaction)
	})

	t.Run("use existing transaction when parent exists", func(t *testing.T) {
		parentTx := db.Begin()
		parent := &TransactionContext{
			Transaction: parentTx,
			Config:      DefaultConfig(),
		}

		tc, err := manager.Begin(parent, config)
		require.NoError(t, err)
		require.NotNil(t, tc)
		assert.Equal(t, parent.Transaction, tc.Transaction)
		assert.Equal(t, parent, tc.Parent)
	})
}

func TestManager_Begin_PropagationRequiresNew(t *testing.T) {
	db := setupTestDB(t)
	manager := NewManager(db)
	config := DefaultConfig().WithPropagation(PropagationRequiresNew)

	t.Run("create new transaction even with parent", func(t *testing.T) {
		parentTx := db.Begin()
		parent := &TransactionContext{
			Transaction: parentTx,
			Config:      DefaultConfig(),
		}

		tc, err := manager.Begin(parent, config)
		require.NoError(t, err)
		require.NotNil(t, tc)
		assert.NotEqual(t, parent.Transaction, tc.Transaction)
		assert.NotEqual(t, db, tc.Transaction)
	})
}

func TestManager_Begin_PropagationMandatory(t *testing.T) {
	db := setupTestDB(t)
	manager := NewManager(db)
	config := DefaultConfig().WithPropagation(PropagationMandatory)

	t.Run("error when no parent transaction", func(t *testing.T) {
		tc, err := manager.Begin(nil, config)
		assert.Error(t, err)
		assert.Equal(t, ErrTransactionRequired, err)
		assert.Nil(t, tc)
	})

	t.Run("use parent transaction when exists", func(t *testing.T) {
		parentTx := db.Begin()
		parent := &TransactionContext{
			Transaction: parentTx,
			Config:      DefaultConfig(),
		}

		tc, err := manager.Begin(parent, config)
		require.NoError(t, err)
		assert.Equal(t, parent.Transaction, tc.Transaction)
	})
}

func TestManager_Begin_PropagationNever(t *testing.T) {
	db := setupTestDB(t)
	manager := NewManager(db)
	config := DefaultConfig().WithPropagation(PropagationNever)

	t.Run("error when parent transaction exists", func(t *testing.T) {
		parentTx := db.Begin()
		parent := &TransactionContext{
			Transaction: parentTx,
			Config:      DefaultConfig(),
		}

		tc, err := manager.Begin(parent, config)
		assert.Error(t, err)
		assert.Equal(t, ErrTransactionNotAllowed, err)
		assert.Nil(t, tc)
	})

	t.Run("use db connection when no parent", func(t *testing.T) {
		tc, err := manager.Begin(nil, config)
		require.NoError(t, err)
		assert.Equal(t, db, tc.Transaction)
	})
}

func TestManager_Commit(t *testing.T) {
	db := setupTestDB(t)
	manager := NewManager(db)

	t.Run("commit real transaction", func(t *testing.T) {
		tx := db.Begin()
		tc := &TransactionContext{
			Transaction: tx,
			Config:      DefaultConfig(),
		}

		// Create a user in transaction
		user := &TestUser{Name: "Test", Age: 20}
		err := tx.Create(user).Error
		require.NoError(t, err)

		err = manager.Commit(tc)
		require.NoError(t, err)
		assert.True(t, tc.Committed)

		// Verify the user is persisted
		var count int64
		db.Model(&TestUser{}).Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("no-op for db connection (non-transaction)", func(t *testing.T) {
		tc := &TransactionContext{
			Transaction: db, // Not a transaction, just the db connection
			Config:      DefaultConfig().WithPropagation(PropagationSupports),
		}

		err := manager.Commit(tc)
		require.NoError(t, err)
		// Should not be marked as committed since it's not a real transaction
		assert.False(t, tc.Committed)
	})
}

func TestManager_Rollback(t *testing.T) {
	db := setupTestDB(t)
	manager := NewManager(db)

	t.Run("rollback transaction", func(t *testing.T) {
		tx := db.Begin()
		tc := &TransactionContext{
			Transaction: tx,
			Config:      DefaultConfig(),
		}

		// Create a user in transaction
		user := &TestUser{Name: "Test", Age: 20}
		err := tx.Create(user).Error
		require.NoError(t, err)

		err = manager.Rollback(tc)
		require.NoError(t, err)
		assert.True(t, tc.RolledBack)

		// Verify the user is NOT persisted
		var count int64
		db.Model(&TestUser{}).Count(&count)
		assert.Equal(t, int64(0), count)
	})
}

func TestMiddleware_Basic(t *testing.T) {
	db := setupTestDB(t)
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Transactional(db))

	router.POST("/users", func(c *gin.Context) {
		db := MustGetDB(c)
		user := &TestUser{Name: "Test", Age: 20}
		err := db.Create(user).Error
		if err != nil {
			c.Error(err)
			return
		}
		c.JSON(200, gin.H{"id": user.ID})
	})

	// Test successful transaction
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/users", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// Verify user was created
	var count int64
	db.Model(&TestUser{}).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestMiddleware_RollbackOnError(t *testing.T) {
	db := setupTestDB(t)
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Transactional(db))

	testErr := errors.New("test error")
	router.POST("/users", func(c *gin.Context) {
		db := MustGetDB(c)
		user := &TestUser{Name: "Test", Age: 20}
		err := db.Create(user).Error
		require.NoError(t, err)

		// Add an error to trigger rollback
		c.Error(testErr)
		c.JSON(500, gin.H{"error": testErr.Error()})
	})

	// Test transaction rollback
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/users", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 500, w.Code)

	// Verify user was NOT created (rolled back)
	var count int64
	db.Model(&TestUser{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestMiddleware_NoRollbackOnSpecificError(t *testing.T) {
	db := setupTestDB(t)
	gin.SetMode(gin.TestMode)

	noRollbackErr := errors.New("no rollback error")
	config := DefaultConfig().WithNoRollbackOn(noRollbackErr)

	router := gin.New()
	router.Use(TransactionalWithConfig(db, config))

	router.POST("/users", func(c *gin.Context) {
		db := MustGetDB(c)
		user := &TestUser{Name: "Test", Age: 20}
		err := db.Create(user).Error
		require.NoError(t, err)

		// Add an error that should not trigger rollback
		c.Error(noRollbackErr)
		c.JSON(400, gin.H{"error": noRollbackErr.Error()})
	})

	// Test transaction commit despite error
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/users", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)

	// Verify user was created (not rolled back)
	var count int64
	db.Model(&TestUser{}).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestGetTransactionContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	// Test no transaction context
	tc, err := GetTransactionContext(c)
	assert.Error(t, err)
	assert.Equal(t, ErrNoTransaction, err)
	assert.Nil(t, tc)

	// Test with transaction context
	expectedTC := &TransactionContext{
		Config: DefaultConfig(),
	}
	c.Set(TransactionContextKey, expectedTC)

	tc, err = GetTransactionContext(c)
	require.NoError(t, err)
	assert.Equal(t, expectedTC, tc)
}

func TestGetDB(t *testing.T) {
	db := setupTestDB(t)
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	// Test no DB in context
	_, err := GetDB(c)
	assert.Error(t, err)

	// Test with transaction context
	tc := &TransactionContext{
		Transaction: db,
		Config:      DefaultConfig(),
	}
	c.Set(TransactionContextKey, tc)

	retrievedDB, err := GetDB(c)
	require.NoError(t, err)
	assert.Equal(t, db, retrievedDB)

	// Test fallback to standard DB field
	c, _ = gin.CreateTestContext(nil)
	c.Set(constants.DbField, db)

	retrievedDB, err = GetDB(c)
	require.NoError(t, err)
	assert.Equal(t, db, retrievedDB)
}

func TestManager_Begin_PropagationSupports(t *testing.T) {
	db := setupTestDB(t)
	manager := NewManager(db)
	config := DefaultConfig().WithPropagation(PropagationSupports)

	t.Run("use parent transaction when exists", func(t *testing.T) {
		parentTx := db.Begin()
		parent := &TransactionContext{
			Transaction: parentTx,
			Config:      DefaultConfig(),
		}

		tc, err := manager.Begin(parent, config)
		require.NoError(t, err)
		assert.Equal(t, parent.Transaction, tc.Transaction)
	})

	t.Run("use db connection when no parent", func(t *testing.T) {
		tc, err := manager.Begin(nil, config)
		require.NoError(t, err)
		assert.Equal(t, db, tc.Transaction)
	})
}

func TestManager_Begin_PropagationNotSupported(t *testing.T) {
	db := setupTestDB(t)
	manager := NewManager(db)
	config := DefaultConfig().WithPropagation(PropagationNotSupported)

	t.Run("use db connection even with parent", func(t *testing.T) {
		parentTx := db.Begin()
		parent := &TransactionContext{
			Transaction: parentTx,
			Config:      DefaultConfig(),
		}

		tc, err := manager.Begin(parent, config)
		require.NoError(t, err)
		assert.Equal(t, db, tc.Transaction)
	})
}

func TestManager_Begin_PropagationNested(t *testing.T) {
	db := setupTestDB(t)
	manager := NewManager(db)
	config := DefaultConfig().WithPropagation(PropagationNested)

	t.Run("create savepoint when parent exists", func(t *testing.T) {
		parentTx := db.Begin()
		parent := &TransactionContext{
			Transaction: parentTx,
			Config:      DefaultConfig(),
		}

		tc, err := manager.Begin(parent, config)
		require.NoError(t, err)
		require.NotNil(t, tc)
		// Nested transaction should have a transaction instance
		assert.NotNil(t, tc.Transaction)
	})

	t.Run("create new transaction when no parent", func(t *testing.T) {
		tc, err := manager.Begin(nil, config)
		require.NoError(t, err)
		require.NotNil(t, tc)
		assert.NotEqual(t, db, tc.Transaction)
	})
}

func TestManager_Begin_IsolationLevel(t *testing.T) {
	db := setupTestDB(t)
	manager := NewManager(db)

	testCases := []struct {
		name  string
		level IsolationLevel
	}{
		{"ReadUncommitted", IsolationReadUncommitted},
		{"ReadCommitted", IsolationReadCommitted},
		{"RepeatableRead", IsolationRepeatableRead},
		{"Serializable", IsolationSerializable},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := DefaultConfig().WithIsolationLevel(tc.level)
			txCtx, err := manager.Begin(nil, config)
			require.NoError(t, err)
			require.NotNil(t, txCtx)
		})
	}
}

func TestManager_Begin_DefaultPropagation(t *testing.T) {
	db := setupTestDB(t)
	manager := NewManager(db)

	// Test with invalid propagation (should default to PropagationRequired)
	config := &Config{
		Propagation: Propagation(999), // Invalid propagation
	}

	tc, err := manager.Begin(nil, config)
	require.NoError(t, err)
	require.NotNil(t, tc)
}

func TestMiddleware_PropagationRequiresNew(t *testing.T) {
	db := setupTestDB(t)
	gin.SetMode(gin.TestMode)

	config := DefaultConfig().WithPropagation(PropagationRequiresNew)
	router := gin.New()
	router.Use(TransactionalWithConfig(db, config))

	router.POST("/users", func(c *gin.Context) {
		db := MustGetDB(c)
		user := &TestUser{Name: "Test", Age: 20}
		err := db.Create(user).Error
		if err != nil {
			c.Error(err)
			return
		}
		c.JSON(200, gin.H{"id": user.ID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/users", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var count int64
	db.Model(&TestUser{}).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestMiddleware_PropagationSupports(t *testing.T) {
	db := setupTestDB(t)
	gin.SetMode(gin.TestMode)

	config := DefaultConfig().WithPropagation(PropagationSupports)
	router := gin.New()
	router.Use(TransactionalWithConfig(db, config))

	router.POST("/users", func(c *gin.Context) {
		db := MustGetDB(c)
		user := &TestUser{Name: "Test", Age: 20}
		err := db.Create(user).Error
		if err != nil {
			c.Error(err)
			return
		}
		c.JSON(200, gin.H{"id": user.ID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/users", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func TestMiddleware_ReadOnly(t *testing.T) {
	db := setupTestDB(t)
	gin.SetMode(gin.TestMode)

	config := DefaultConfig().WithReadOnly(true)
	router := gin.New()
	router.Use(TransactionalWithConfig(db, config))

	router.GET("/users", func(c *gin.Context) {
		db := MustGetDB(c)
		var users []TestUser
		db.Find(&users)
		c.JSON(200, users)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/users", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func TestMiddleware_MultipleErrors(t *testing.T) {
	db := setupTestDB(t)
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Transactional(db))

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	router.POST("/users", func(c *gin.Context) {
		db := MustGetDB(c)
		user := &TestUser{Name: "Test", Age: 20}
		err := db.Create(user).Error
		require.NoError(t, err)

		c.Error(err1)
		c.Error(err2)
		c.JSON(500, gin.H{"error": "multiple errors"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/users", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 500, w.Code)

	// Should rollback
	var count int64
	db.Model(&TestUser{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestMiddleware_NoError(t *testing.T) {
	db := setupTestDB(t)
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Transactional(db))

	router.POST("/users", func(c *gin.Context) {
		db := MustGetDB(c)
		user := &TestUser{Name: "Test", Age: 20}
		err := db.Create(user).Error
		require.NoError(t, err)
		c.JSON(200, gin.H{"id": user.ID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/users", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// Should commit
	var count int64
	db.Model(&TestUser{}).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestGetDB_Fallback(t *testing.T) {
	db := setupTestDB(t)
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	// Set DB in standard field (no transaction context)
	c.Set(constants.DbField, db)

	retrievedDB, err := GetDB(c)
	require.NoError(t, err)
	assert.Equal(t, db, retrievedDB)
}

func TestGetDB_NoDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	// No DB in context at all
	_, err := GetDB(c)
	assert.Error(t, err)
	assert.Equal(t, ErrNoTransaction, err)
}

func TestMustGetDB_Panic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	// Should panic when no DB
	assert.Panics(t, func() {
		MustGetDB(c)
	})
}

func TestTransactionContext_ShouldRollback_NilError(t *testing.T) {
	tc := &TransactionContext{
		Config: DefaultConfig(),
	}
	assert.False(t, tc.ShouldRollback(nil))
}

func TestManager_Begin_NilConfig(t *testing.T) {
	db := setupTestDB(t)
	manager := NewManager(db)

	tc, err := manager.Begin(nil, nil)
	require.NoError(t, err)
	require.NotNil(t, tc)
	assert.NotNil(t, tc.Transaction)
}

func TestManager_Commit_NestedTransaction(t *testing.T) {
	db := setupTestDB(t)
	manager := NewManager(db)

	parentTx := db.Begin()
	parent := &TransactionContext{
		Transaction: parentTx,
		Config:      DefaultConfig(),
	}

	nestedConfig := DefaultConfig().WithPropagation(PropagationNested)
	tc, err := manager.Begin(parent, nestedConfig)
	require.NoError(t, err)

	// Nested transactions don't commit independently
	err = manager.Commit(tc)
	require.NoError(t, err)
	// Should not be marked as committed (handled by parent)
}

func TestManager_Rollback_NestedTransaction(t *testing.T) {
	db := setupTestDB(t)
	manager := NewManager(db)

	parentTx := db.Begin()
	parent := &TransactionContext{
		Transaction: parentTx,
		Config:      DefaultConfig(),
	}

	nestedConfig := DefaultConfig().WithPropagation(PropagationNested)
	tc, err := manager.Begin(parent, nestedConfig)
	require.NoError(t, err)

	err = manager.Rollback(tc)
	require.NoError(t, err)
	assert.True(t, tc.RolledBack)
}

func TestConfig_ChainMethods(t *testing.T) {
	config := DefaultConfig().
		WithPropagation(PropagationRequiresNew).
		WithIsolationLevel(IsolationReadCommitted).
		WithReadOnly(true).
		WithTimeout(30).
		WithRollbackOn(errors.New("test")).
		WithNoRollbackOn(errors.New("no rollback"))

	assert.Equal(t, PropagationRequiresNew, config.Propagation)
	assert.Equal(t, IsolationReadCommitted, config.IsolationLevel)
	assert.True(t, config.ReadOnly)
	assert.Equal(t, 30, config.Timeout)
	assert.Len(t, config.RollbackOn, 1)
	assert.Len(t, config.NoRollbackOn, 1)
}
