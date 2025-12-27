package reactive

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewMiddleware(t *testing.T) {
	middleware := NewMiddleware(nil)
	assert.NotNil(t, middleware)
	assert.NotNil(t, middleware.config)
}

func TestNewMiddleware_WithConfig(t *testing.T) {
	config := &Config{
		BufferSize: 256,
		Prefetch:   64,
	}
	middleware := NewMiddleware(config)
	assert.NotNil(t, middleware)
	assert.Equal(t, config, middleware.config)
}

func TestMiddleware_Handler(t *testing.T) {
	middleware := NewMiddleware(nil)
	handler := middleware.Handler(func(c *gin.Context) Publisher {
		return FromSlice([]interface{}{map[string]interface{}{"test": "value"}})
	})
	assert.NotNil(t, handler)
}

func TestMiddleware_StreamHandler(t *testing.T) {
	middleware := NewMiddleware(nil)
	handler := middleware.StreamHandler(func(c *gin.Context) Publisher {
		return FromSlice([]interface{}{map[string]interface{}{"test": "value"}})
	})
	assert.NotNil(t, handler)
}
