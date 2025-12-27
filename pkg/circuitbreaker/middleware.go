package circuitbreaker

import (
	"errors"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	// DefaultRegistry is the default circuit breaker registry
	DefaultRegistry = NewRegistry()
)

// Registry manages multiple circuit breakers
type Registry struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
}

// NewRegistry creates a new circuit breaker registry
func NewRegistry() *Registry {
	return &Registry{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetOrCreate returns an existing circuit breaker or creates a new one
func (r *Registry) GetOrCreate(name string, config *Config) *CircuitBreaker {
	r.mu.RLock()
	cb, exists := r.breakers[name]
	r.mu.RUnlock()

	if exists {
		return cb
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double check
	if cb, exists := r.breakers[name]; exists {
		return cb
	}

	if config == nil {
		config = DefaultConfig(name)
	} else {
		config.Name = name
	}

	cb = New(config)
	r.breakers[name] = cb
	return cb
}

// Get returns a circuit breaker by name
func (r *Registry) Get(name string) *CircuitBreaker {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.breakers[name]
}

// Remove removes a circuit breaker from the registry
func (r *Registry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.breakers, name)
}

// Clear removes all circuit breakers from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.breakers = make(map[string]*CircuitBreaker)
}

// GetAll returns all circuit breakers
func (r *Registry) GetAll() map[string]*CircuitBreaker {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]*CircuitBreaker)
	for k, v := range r.breakers {
		result[k] = v
	}
	return result
}

// Middleware creates a Gin middleware that applies circuit breaker protection
func Middleware(cb *CircuitBreaker, fallback func(*gin.Context, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if circuit is open before executing
		if cb.IsOpen() {
			err := ErrCircuitOpen
			if fallback != nil {
				fallback(c, err)
			} else {
				// Default fallback: return 503 Service Unavailable
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
					"error":   "service temporarily unavailable",
					"message": err.Error(),
				})
			}
			return
		}

		// Execute the handlers with circuit breaker protection
		err := cb.Execute(func() error {
			c.Next()
			// Check if there were any errors in the handlers chain
			if len(c.Errors) > 0 {
				return c.Errors.Last()
			}
			// Check if response indicates an error
			if c.Writer.Status() >= 400 {
				return errors.New("handlers returned error status")
			}
			return nil
		})

		if err != nil && err != ErrCircuitOpen {
			// If Execute failed (not circuit open), handle it
			if fallback != nil {
				fallback(c, err)
			} else {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
					"error":   "service temporarily unavailable",
					"message": err.Error(),
				})
			}
		}
	}
}

// MiddlewareWithName creates a middleware using a circuit breaker from the default registry
func MiddlewareWithName(name string, config *Config, fallback func(*gin.Context, error)) gin.HandlerFunc {
	cb := DefaultRegistry.GetOrCreate(name, config)
	return Middleware(cb, fallback)
}
