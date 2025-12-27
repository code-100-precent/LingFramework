package circuitbreaker

import (
	"sync/atomic"
	"time"
)

// onSuccess is called when a request succeeds
func (cb *CircuitBreaker) onSuccess() {
	cb.mu.Lock()
	cb.counts.TotalSuccesses++
	cb.counts.ConsecutiveSuccesses++
	cb.counts.ConsecutiveFailures = 0
	cb.counts.Requests++

	state := cb.State()
	consecutiveSuccesses := cb.counts.ConsecutiveSuccesses
	shouldClose := false
	if state == StateHalfOpen {
		// If we're in half-open state and have enough successes, close the circuit
		if consecutiveSuccesses >= cb.config.SuccessThreshold {
			shouldClose = true
		}
	} else if state == StateOpen {
		// Should not happen, but handle it anyway
		shouldClose = true
	}
	cb.mu.Unlock()

	if shouldClose {
		cb.setState(StateClosed)
	}
}

// onFailure is called when a request fails
func (cb *CircuitBreaker) onFailure() {
	cb.mu.Lock()
	cb.counts.TotalFailures++
	cb.counts.ConsecutiveFailures++
	cb.counts.ConsecutiveSuccesses = 0
	cb.counts.Requests++

	state := cb.State()
	consecutiveFailures := cb.counts.ConsecutiveFailures
	maxFailures := cb.config.MaxFailures
	var shouldTrip bool
	if cb.config.ShouldTrip != nil {
		shouldTrip = cb.config.ShouldTrip(cb.counts)
	} else {
		shouldTrip = consecutiveFailures >= maxFailures
	}

	shouldOpen := false
	if state == StateClosed && shouldTrip {
		shouldOpen = true
	} else if state == StateHalfOpen {
		// Any failure in half-open state opens the circuit
		shouldOpen = true
	}
	cb.mu.Unlock()

	if shouldOpen {
		cb.setState(StateOpen)
	}
}

// setState changes the state of the circuit breaker
func (cb *CircuitBreaker) setState(newState State) {
	oldState := cb.State()
	if oldState == newState {
		return
	}

	atomic.StoreInt32(&cb.state, int32(newState))
	cb.lastState = time.Now()
	cb.mu.Lock()
	cb.generation++
	cb.counts = Counts{} // Reset counts on state change
	cb.mu.Unlock()

	if cb.config.OnStateChange != nil {
		cb.config.OnStateChange(cb.name, oldState, newState)
	}
}

// Execute runs a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
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

// ExecuteWithFallback runs a function with circuit breaker protection and fallback
func (cb *CircuitBreaker) ExecuteWithFallback(fn func() error, fallback func(error) error) error {
	err := cb.Execute(fn)
	if err != nil && fallback != nil {
		// If circuit is open or function failed, use fallback
		if err == ErrCircuitOpen || err != nil {
			return fallback(err)
		}
	}
	return err
}

// beforeRequest checks if a request should be allowed
func (cb *CircuitBreaker) beforeRequest() bool {
	state := cb.State()

	switch state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if enough time has passed to try half-open
		cb.mu.RLock()
		lastStateTime := cb.lastState
		cb.mu.RUnlock()

		if cb.config.ReadyToTrip != nil {
			cb.mu.RLock()
			counts := cb.counts
			cb.mu.RUnlock()
			if cb.config.ReadyToTrip(counts) {
				cb.setState(StateHalfOpen)
				return true
			}
		} else {
			// Use timeout-based ready to trip
			if time.Since(lastStateTime) >= cb.config.Timeout {
				cb.setState(StateHalfOpen)
				return true
			}
		}
		return false
	case StateHalfOpen:
		// Allow limited number of requests
		cb.mu.RLock()
		requests := cb.counts.Requests
		cb.mu.RUnlock()
		if requests >= cb.config.MaxRequests {
			return false
		}
		return true
	default:
		return false
	}
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.setState(StateClosed)
}

// IsOpen returns true if the circuit breaker is open
func (cb *CircuitBreaker) IsOpen() bool {
	return cb.State() == StateOpen
}

// IsClosed returns true if the circuit breaker is closed
func (cb *CircuitBreaker) IsClosed() bool {
	return cb.State() == StateClosed
}

// IsHalfOpen returns true if the circuit breaker is half-open
func (cb *CircuitBreaker) IsHalfOpen() bool {
	return cb.State() == StateHalfOpen
}
