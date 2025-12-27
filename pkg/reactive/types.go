package reactive

import (
	"context"
)

// Publisher is the producer of a stream of data
type Publisher interface {
	// Subscribe creates a subscription between the publisher and subscriber
	Subscribe(subscriber Subscriber) Subscription
}

// Subscriber receives notifications from a Publisher
type Subscriber interface {
	// OnSubscribe is called when the subscription is created
	OnSubscribe(subscription Subscription)
	// OnNext is called when a new value is available
	OnNext(value interface{})
	// OnError is called when an error occurs
	OnError(err error)
	// OnComplete is called when the stream completes
	OnComplete()
}

// Subscription represents the relationship between a Publisher and Subscriber
type Subscription interface {
	// Request requests n items from the publisher
	Request(n int64)
	// Cancel cancels the subscription
	Cancel()
}

// Flow represents a reactive stream with backpressure support
type Flow struct {
	subscribers []Subscriber
	mu          chan struct{} // Simple mutex using channel
	closed      int32         // Atomic flag for closed state
	cancel      context.CancelFunc
}

// Processor represents a component that acts as both a Subscriber and Publisher
type Processor interface {
	Subscriber
	Publisher
}

// Config represents configuration for reactive streams
type Config struct {
	// BufferSize is the buffer size for backpressure (default: 128)
	BufferSize int
	// Prefetch is the number of items to prefetch (default: 32)
	Prefetch int
	// OnError is called when an error occurs
	OnError func(error)
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		BufferSize: 128,
		Prefetch:   32,
	}
}
