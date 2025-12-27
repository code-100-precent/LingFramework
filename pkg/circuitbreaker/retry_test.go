package circuitbreaker

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, config.InitialInterval)
	assert.Equal(t, 1*time.Second, config.MaxInterval)
	assert.Equal(t, 2.0, config.Multiplier)
}

func TestRetry_Success(t *testing.T) {
	attempts := 0
	err := Retry(func() error {
		attempts++
		return nil
	}, nil)

	assert.NoError(t, err)
	assert.Equal(t, 1, attempts)
}

func TestRetry_SuccessAfterRetries(t *testing.T) {
	attempts := 0
	err := Retry(func() error {
		attempts++
		if attempts < 3 {
			return testError
		}
		return nil
	}, &RetryConfig{
		MaxAttempts:     5,
		InitialInterval: 10 * time.Millisecond,
		Multiplier:      2.0,
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestRetry_MaxAttemptsExceeded(t *testing.T) {
	attempts := 0
	err := Retry(func() error {
		attempts++
		return testError
	}, &RetryConfig{
		MaxAttempts:     3,
		InitialInterval: 10 * time.Millisecond,
		Multiplier:      2.0,
	})

	assert.Error(t, err)
	assert.Equal(t, testError, err)
	assert.Equal(t, 3, attempts)
}

func TestRetry_RetryableErrors(t *testing.T) {
	retryableError := errors.New("retryable")
	nonRetryableError := errors.New("non-retryable")

	attempts := 0
	err := Retry(func() error {
		attempts++
		if attempts < 2 {
			return retryableError
		}
		return nonRetryableError
	}, &RetryConfig{
		MaxAttempts:     5,
		InitialInterval: 10 * time.Millisecond,
		Multiplier:      2.0,
		RetryableErrors: func(err error) bool {
			return err == retryableError
		},
	})

	assert.Error(t, err)
	assert.Equal(t, nonRetryableError, err)
	assert.Equal(t, 2, attempts) // Should stop after non-retryable error
}

func TestRetry_ExponentialBackoff(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:     5,
		InitialInterval: 50 * time.Millisecond,
		MaxInterval:     200 * time.Millisecond,
		Multiplier:      2.0,
	}

	start := time.Now()
	Retry(func() error {
		return testError
	}, config)
	duration := time.Since(start)

	// Should have waited for at least some time (allowing for timing variations)
	assert.Greater(t, duration, 100*time.Millisecond)
}

func TestRetryWithCircuitBreaker(t *testing.T) {
	cb := New(&Config{
		Name:        "test",
		MaxFailures: 10, // High threshold
		Timeout:     100 * time.Millisecond,
	})

	attempts := 0
	err := RetryWithCircuitBreaker(cb, func() error {
		attempts++
		if attempts < 3 {
			return testError
		}
		return nil
	}, &RetryConfig{
		MaxAttempts:     5,
		InitialInterval: 10 * time.Millisecond,
	}, nil)

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestRetryWithCircuitBreaker_Fallback(t *testing.T) {
	cb := New(&Config{
		Name:        "test",
		MaxFailures: 1,
		Timeout:     100 * time.Millisecond,
	})

	fallbackCalled := false
	fallback := func(err error) error {
		fallbackCalled = true
		return errors.New("fallback error")
	}

	// Open the circuit
	cb.Execute(func() error {
		return testError
	})

	// Verify circuit is open
	assert.True(t, cb.IsOpen())

	err := RetryWithCircuitBreaker(cb, func() error {
		return nil
	}, &RetryConfig{
		MaxAttempts:     3,
		InitialInterval: 10 * time.Millisecond,
		RetryableErrors: func(err error) bool {
			// Don't retry on circuit open
			return err != ErrCircuitOpen
		},
	}, fallback)

	assert.Error(t, err)
	assert.True(t, fallbackCalled)
}

func TestRetryWithCircuitBreaker_NoCircuitBreaker(t *testing.T) {
	attempts := 0
	err := RetryWithCircuitBreaker(nil, func() error {
		attempts++
		if attempts < 2 {
			return testError
		}
		return nil
	}, &RetryConfig{
		MaxAttempts:     5,
		InitialInterval: 10 * time.Millisecond,
	}, nil)

	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)
}
