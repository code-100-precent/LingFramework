package circuitbreaker

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// State represents the state of a circuit breaker
type State int32

const (
	// StateClosed Circuit breaker is closed, requests pass through normally
	StateClosed State = iota
	// StateOpen Circuit breaker is open, requests fail immediately without calling the function
	StateOpen
	// StateHalfOpen Circuit breaker is half-open, allows a limited number of requests to test if the service has recovered
	StateHalfOpen
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// Config represents the configuration for a circuit breaker
type Config struct {
	// Name is the name of the circuit breaker (for logging/monitoring)
	Name string
	// MaxRequests is the maximum number of requests allowed in half-open state (default: 1)
	MaxRequests int64
	// Interval is the time window for counting failures (default: 60s)
	Interval time.Duration
	// Timeout is the duration the circuit breaker stays open before transitioning to half-open (default: 60s)
	Timeout time.Duration
	// MaxFailures is the maximum number of failures before opening the circuit (default: 5)
	MaxFailures int64
	// SuccessThreshold is the number of successful requests needed in half-open state to close the circuit (default: 1)
	SuccessThreshold int64
	// OnStateChange is called when the circuit breaker state changes
	OnStateChange func(name string, from, to State)
	// ShouldTrip is a custom function to determine if the circuit should trip open
	// If nil, the default behavior (MaxFailures) is used
	ShouldTrip func(counts Counts) bool
	// ReadyToTrip is called when transitioning from open to half-open
	// If nil, uses Timeout duration
	ReadyToTrip func(counts Counts) bool
}

// DefaultConfig returns a default configuration
func DefaultConfig(name string) *Config {
	return &Config{
		Name:             name,
		MaxRequests:      1,
		Interval:         60 * time.Second,
		Timeout:          60 * time.Second,
		MaxFailures:      5,
		SuccessThreshold: 1,
	}
}

// Counts represents the statistics of the circuit breaker
type Counts struct {
	Requests             int64 // Total number of requests
	TotalSuccesses       int64 // Total number of successful requests
	TotalFailures        int64 // Total number of failed requests
	ConsecutiveSuccesses int64 // Number of consecutive successful requests
	ConsecutiveFailures  int64 // Number of consecutive failed requests
}

// CircuitBreaker is a circuit breaker implementation
type CircuitBreaker struct {
	name       string
	config     *Config
	state      int32 // atomic access, State enum
	counts     Counts
	lastState  time.Time
	mu         sync.RWMutex
	generation uint64 // incremented each time the circuit breaker opens
}

// New creates a new circuit breaker with the given configuration
func New(config *Config) *CircuitBreaker {
	if config == nil {
		config = DefaultConfig("default")
	}
	if config.MaxRequests <= 0 {
		config.MaxRequests = 1
	}
	if config.Interval <= 0 {
		config.Interval = 60 * time.Second
	}
	if config.Timeout <= 0 {
		config.Timeout = 60 * time.Second
	}
	if config.MaxFailures <= 0 {
		config.MaxFailures = 5
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = 1
	}

	cb := &CircuitBreaker{
		name:      config.Name,
		config:    config,
		state:     int32(StateClosed),
		lastState: time.Now(),
	}

	return cb
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() State {
	return State(atomic.LoadInt32(&cb.state))
}

// Counts returns the current counts
func (cb *CircuitBreaker) Counts() Counts {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.counts
}

// Name returns the name of the circuit breaker
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

var (
	// ErrCircuitOpen is returned when the circuit breaker is open
	ErrCircuitOpen = errors.New("circuit breaker is open")
	// ErrTooManyRequests is returned when too many requests are made in half-open state
	ErrTooManyRequests = errors.New("too many requests in half-open state")
)
