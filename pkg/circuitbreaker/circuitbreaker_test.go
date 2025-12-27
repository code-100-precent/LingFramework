package circuitbreaker

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testError = errors.New("test error")

func TestNew(t *testing.T) {
	cb := New(DefaultConfig("test"))
	assert.NotNil(t, cb)
	assert.Equal(t, StateClosed, cb.State())
	assert.Equal(t, "test", cb.Name())
}

func TestCircuitBreaker_State(t *testing.T) {
	cb := New(DefaultConfig("test"))
	assert.Equal(t, StateClosed, cb.State())
	assert.True(t, cb.IsClosed())
	assert.False(t, cb.IsOpen())
	assert.False(t, cb.IsHalfOpen())
}

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	cb := New(DefaultConfig("test"))
	err := cb.Execute(func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())

	counts := cb.Counts()
	assert.Equal(t, int64(1), counts.Requests)
	assert.Equal(t, int64(1), counts.TotalSuccesses)
	assert.Equal(t, int64(1), counts.ConsecutiveSuccesses)
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	cb := New(&Config{
		Name:        "test",
		MaxFailures: 3,
		Timeout:     100 * time.Millisecond,
	})

	// Fail 3 times to open the circuit
	for i := 0; i < 3; i++ {
		err := cb.Execute(func() error {
			return testError
		})
		assert.Error(t, err)
	}

	assert.Equal(t, StateOpen, cb.State())
	assert.True(t, cb.IsOpen())
}

func TestCircuitBreaker_Execute_CircuitOpen(t *testing.T) {
	cb := New(&Config{
		Name:        "test",
		MaxFailures: 1,
		Timeout:     100 * time.Millisecond,
	})

	// Fail once to open the circuit
	err := cb.Execute(func() error {
		return testError
	})
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.State())

	// Next request should fail immediately
	err = cb.Execute(func() error {
		return nil
	})
	assert.Equal(t, ErrCircuitOpen, err)
}

func TestCircuitBreaker_Execute_HalfOpen(t *testing.T) {
	cb := New(&Config{
		Name:             "test",
		MaxFailures:      1,
		Timeout:          50 * time.Millisecond,
		MaxRequests:      2,
		SuccessThreshold: 1,
	})

	// Open the circuit
	cb.Execute(func() error {
		return testError
	})
	assert.Equal(t, StateOpen, cb.State())

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// First request should be allowed (half-open)
	err := cb.Execute(func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_ExecuteWithFallback(t *testing.T) {
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

	// Execute with fallback
	err := cb.ExecuteWithFallback(func() error {
		return nil
	}, fallback)

	assert.Error(t, err)
	assert.Equal(t, "fallback error", err.Error())
	assert.True(t, fallbackCalled)
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := New(&Config{
		Name:        "test",
		MaxFailures: 1,
		Timeout:     100 * time.Millisecond,
	})

	// Open the circuit
	cb.Execute(func() error {
		return testError
	})
	assert.Equal(t, StateOpen, cb.State())

	// Reset
	cb.Reset()
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_Counts(t *testing.T) {
	cb := New(DefaultConfig("test"))

	// Success
	cb.Execute(func() error {
		return nil
	})

	// Failure
	cb.Execute(func() error {
		return testError
	})

	counts := cb.Counts()
	assert.Equal(t, int64(2), counts.Requests)
	assert.Equal(t, int64(1), counts.TotalSuccesses)
	assert.Equal(t, int64(1), counts.TotalFailures)
}

func TestCircuitBreaker_Concurrent(t *testing.T) {
	cb := New(&Config{
		Name:        "test",
		MaxFailures: 100, // High threshold to avoid opening during test
		Timeout:     100 * time.Millisecond,
	})

	var wg sync.WaitGroup
	numGoroutines := 100
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			cb.Execute(func() error {
				return nil
			})
		}()
	}

	wg.Wait()
	counts := cb.Counts()
	assert.Equal(t, int64(numGoroutines), counts.Requests)
}

func TestCircuitBreaker_OnStateChange(t *testing.T) {
	stateChanges := make([]StateChange, 0)
	onStateChange := func(name string, from, to State) {
		stateChanges = append(stateChanges, StateChange{
			Name: name,
			From: from,
			To:   to,
		})
	}

	cb := New(&Config{
		Name:          "test",
		MaxFailures:   1,
		Timeout:       50 * time.Millisecond,
		OnStateChange: onStateChange,
	})

	// Open the circuit
	cb.Execute(func() error {
		return testError
	})

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Close the circuit
	cb.Execute(func() error {
		return nil
	})

	require.GreaterOrEqual(t, len(stateChanges), 1)
	assert.Equal(t, StateClosed, stateChanges[0].From)
	assert.Equal(t, StateOpen, stateChanges[0].To)
}

type StateChange struct {
	Name string
	From State
	To   State
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig("test")
	assert.NotNil(t, config)
	assert.Equal(t, "test", config.Name)
	assert.Equal(t, int64(1), config.MaxRequests)
	assert.Equal(t, 60*time.Second, config.Interval)
	assert.Equal(t, 60*time.Second, config.Timeout)
	assert.Equal(t, int64(5), config.MaxFailures)
	assert.Equal(t, int64(1), config.SuccessThreshold)
}

func TestCircuitBreaker_GetStats(t *testing.T) {
	cb := New(DefaultConfig("test"))
	stats := cb.GetStats()

	assert.Equal(t, "test", stats.Name)
	assert.Equal(t, "CLOSED", stats.State)
	assert.Equal(t, uint64(0), stats.Generation)
}

func TestCircuitBreaker_GetState(t *testing.T) {
	cb := New(DefaultConfig("test"))
	assert.Equal(t, "CLOSED", cb.GetState())

	cb.Execute(func() error {
		return testError
	})
	cb.Execute(func() error {
		return testError
	})
	cb.Execute(func() error {
		return testError
	})
	cb.Execute(func() error {
		return testError
	})
	cb.Execute(func() error {
		return testError
	})

	assert.Equal(t, "OPEN", cb.GetState())
}

func TestCircuitBreaker_ExecuteWithTimeout(t *testing.T) {
	cb := New(DefaultConfig("test"))

	// Success case
	err := cb.ExecuteWithTimeout(func() error {
		return nil
	}, 100*time.Millisecond)
	assert.NoError(t, err)

	// Failure case
	err = cb.ExecuteWithTimeout(func() error {
		return testError
	}, 100*time.Millisecond)
	assert.Error(t, err)
}

func TestCircuitBreaker_CustomShouldTrip(t *testing.T) {
	customShouldTrip := func(counts Counts) bool {
		return counts.TotalFailures >= 2
	}

	cb := New(&Config{
		Name:       "test",
		ShouldTrip: customShouldTrip,
		Timeout:    100 * time.Millisecond,
	})

	// Fail twice
	cb.Execute(func() error {
		return testError
	})
	assert.Equal(t, StateClosed, cb.State())

	cb.Execute(func() error {
		return testError
	})
	assert.Equal(t, StateOpen, cb.State())
}
