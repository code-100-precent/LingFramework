package circuitbreaker

import (
	"errors"
	"time"
)

// RetryConfig represents the configuration for retry logic
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts (default: 3)
	MaxAttempts int
	// InitialInterval is the initial delay between retries (default: 100ms)
	InitialInterval time.Duration
	// MaxInterval is the maximum delay between retries (default: 1s)
	MaxInterval time.Duration
	// Multiplier is the multiplier for exponential backoff (default: 2.0)
	Multiplier float64
	// RetryableErrors is a function that determines if an error is retryable
	// If nil, all errors are retryable
	RetryableErrors func(error) bool
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:     3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		RetryableErrors: nil, // All errors are retryable by default
	}
}

// WithMaxAttempts sets the maximum number of retry attempts
func (rc *RetryConfig) WithMaxAttempts(maxAttempts int) *RetryConfig {
	rc.MaxAttempts = maxAttempts
	return rc
}

// WithInitialInterval sets the initial delay between retries
func (rc *RetryConfig) WithInitialInterval(interval time.Duration) *RetryConfig {
	rc.InitialInterval = interval
	return rc
}

// WithMaxInterval sets the maximum delay between retries
func (rc *RetryConfig) WithMaxInterval(interval time.Duration) *RetryConfig {
	rc.MaxInterval = interval
	return rc
}

// WithMultiplier sets the multiplier for exponential backoff
func (rc *RetryConfig) WithMultiplier(multiplier float64) *RetryConfig {
	rc.Multiplier = multiplier
	return rc
}

// WithRetryableErrors sets the function to determine retryable errors
func (rc *RetryConfig) WithRetryableErrors(fn func(error) bool) *RetryConfig {
	rc.RetryableErrors = fn
	return rc
}

// Retry executes a function with retry logic
func Retry(fn func() error, config *RetryConfig) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error
	interval := config.InitialInterval

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if config.RetryableErrors != nil && !config.RetryableErrors(err) {
			return err
		}

		// Don't sleep after the last attempt
		if attempt < config.MaxAttempts-1 {
			time.Sleep(interval)
			// Exponential backoff
			interval = time.Duration(float64(interval) * config.Multiplier)
			if interval > config.MaxInterval {
				interval = config.MaxInterval
			}
		}
	}

	return lastErr
}

// RetryWithCircuitBreaker executes a function with both retry and circuit breaker protection
func RetryWithCircuitBreaker(cb *CircuitBreaker, fn func() error, retryConfig *RetryConfig, fallback func(error) error) error {
	if cb == nil {
		// If no circuit breaker, just use retry
		return Retry(fn, retryConfig)
	}

	// Wrap the function to be retried with circuit breaker
	retryFn := func() error {
		return cb.Execute(fn)
	}

	err := Retry(retryFn, retryConfig)
	if err != nil {
		if fallback != nil {
			return fallback(err)
		}
		return err
	}
	return nil
}

// ErrMaxRetriesExceeded is returned when maximum retry attempts are exceeded
var ErrMaxRetriesExceeded = errors.New("max retries exceeded")
