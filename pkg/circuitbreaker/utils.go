package circuitbreaker

import (
	"context"
	"time"
)

// ExecuteWithContext executes a function with circuit breaker protection and context
func (cb *CircuitBreaker) ExecuteWithContext(ctx context.Context, fn func() error) error {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Check if we can execute the function
	if !cb.beforeRequest() {
		return ErrCircuitOpen
	}

	// Execute the function
	err := fn()

	// Record the result
	if err != nil {
		cb.onFailure()
		return err
	}

	cb.onSuccess()
	return nil
}

// ExecuteWithTimeout executes a function with circuit breaker protection and timeout
func (cb *CircuitBreaker) ExecuteWithTimeout(fn func() error, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return cb.ExecuteWithContext(ctx, fn)
}

// GetState returns the current state as a string
func (cb *CircuitBreaker) GetState() string {
	return cb.State().String()
}

// GetGeneration returns the current generation number (incremented on each state change)
func (cb *CircuitBreaker) GetGeneration() uint64 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.generation
}

// GetLastStateChange returns when the state last changed
func (cb *CircuitBreaker) GetLastStateChange() time.Time {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.lastState
}

// Stats returns statistics about the circuit breaker
type Stats struct {
	Name            string    `json:"name"`
	State           string    `json:"state"`
	Generation      uint64    `json:"generation"`
	LastStateChange time.Time `json:"last_state_change"`
	Counts          Counts    `json:"counts"`
	Config          Config    `json:"config"`
}

// GetStats returns statistics about the circuit breaker
func (cb *CircuitBreaker) GetStats() Stats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return Stats{
		Name:            cb.name,
		State:           cb.State().String(),
		Generation:      cb.generation,
		LastStateChange: cb.lastState,
		Counts:          cb.counts,
	}
}
