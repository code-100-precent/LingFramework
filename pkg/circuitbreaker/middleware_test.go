package circuitbreaker

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRegistry_GetOrCreate(t *testing.T) {
	registry := NewRegistry()

	cb1 := registry.GetOrCreate("test", nil)
	assert.NotNil(t, cb1)

	cb2 := registry.GetOrCreate("test", nil)
	assert.Equal(t, cb1, cb2) // Should return the same instance
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	cb := registry.GetOrCreate("test", nil)
	retrieved := registry.Get("test")
	assert.Equal(t, cb, retrieved)

	retrieved = registry.Get("nonexistent")
	assert.Nil(t, retrieved)
}

func TestRegistry_Remove(t *testing.T) {
	registry := NewRegistry()

	registry.GetOrCreate("test", nil)
	registry.Remove("test")

	retrieved := registry.Get("test")
	assert.Nil(t, retrieved)
}

func TestRegistry_Clear(t *testing.T) {
	registry := NewRegistry()

	registry.GetOrCreate("test1", nil)
	registry.GetOrCreate("test2", nil)
	registry.Clear()

	assert.Nil(t, registry.Get("test1"))
	assert.Nil(t, registry.Get("test2"))
}

func TestMiddleware_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cb := New(DefaultConfig("test"))

	router := gin.New()
	router.Use(Middleware(cb, nil))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestMiddleware_CircuitOpen(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cb := New(&Config{
		Name:        "test",
		MaxFailures: 1,
		Timeout:     100,
	})

	// Open the circuit
	cb.Execute(func() error {
		return testError
	})

	customFallback := false
	fallback := func(c *gin.Context, err error) {
		customFallback = true
		c.JSON(503, gin.H{"error": "circuit open"})
	}

	router := gin.New()
	router.Use(Middleware(cb, fallback))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 503, w.Code)
	assert.True(t, customFallback)
}

func TestMiddleware_DefaultFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cb := New(&Config{
		Name:        "test",
		MaxFailures: 1,
		Timeout:     100,
	})

	// Open the circuit
	cb.Execute(func() error {
		return testError
	})

	router := gin.New()
	router.Use(Middleware(cb, nil)) // No custom fallback
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 503, w.Code)
}

func TestMiddlewareWithName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := NewRegistry()
	DefaultRegistry = registry // Replace for test

	router := gin.New()
	router.Use(MiddlewareWithName("test-middleware", nil, nil))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// Verify circuit breaker was created
	cb := registry.Get("test-middleware")
	assert.NotNil(t, cb)
}
