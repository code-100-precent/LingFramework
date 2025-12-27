package reactive

import (
	"github.com/gin-gonic/gin"
)

// Middleware provides reactive middleware for Gin
type Middleware struct {
	config *Config
}

// NewMiddleware creates a new reactive middleware
func NewMiddleware(config *Config) *Middleware {
	if config == nil {
		config = DefaultConfig()
	}
	return &Middleware{config: config}
}

// Handler wraps a handler to add reactive capabilities
func (m *Middleware) Handler(handler ReactiveHandler) gin.HandlerFunc {
	return Handler(handler)
}

// StreamHandler wraps a handler to add SSE streaming capabilities
func (m *Middleware) StreamHandler(handler ReactiveHandler) gin.HandlerFunc {
	return StreamHandler(handler)
}
