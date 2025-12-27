package reactive

import (
	"context"
	"sync"
	"sync/atomic"
)

// NewFlow creates a new Flow
func NewFlow() *Flow {
	mu := make(chan struct{}, 1)
	mu <- struct{}{} // Initialize mutex
	_, cancel := context.WithCancel(context.Background())
	return &Flow{
		subscribers: make([]Subscriber, 0),
		mu:          mu,
		cancel:      cancel,
	}
}

// Subscribe adds a subscriber to the flow
func (f *Flow) Subscribe(subscriber Subscriber) Subscription {
	if atomic.LoadInt32(&f.closed) == 1 {
		subscriber.OnError(ErrFlowClosed)
		return nil
	}

	// Lock
	<-f.mu
	defer func() { f.mu <- struct{}{} }()

	// Create subscription
	sub := newSubscription(f, subscriber)
	f.subscribers = append(f.subscribers, subscriber)

	// Notify subscriber of subscription
	subscriber.OnSubscribe(sub)

	return sub
}

// Publish publishes a value to all subscribers
func (f *Flow) Publish(value interface{}) error {
	if atomic.LoadInt32(&f.closed) == 1 {
		return ErrFlowClosed
	}

	<-f.mu
	subscribers := make([]Subscriber, len(f.subscribers))
	copy(subscribers, f.subscribers)
	f.mu <- struct{}{}

	for _, sub := range subscribers {
		sub.OnNext(value)
	}
	return nil
}

// PublishError publishes an error to all subscribers
func (f *Flow) PublishError(err error) {
	<-f.mu
	subscribers := make([]Subscriber, len(f.subscribers))
	copy(subscribers, f.subscribers)
	f.mu <- struct{}{}

	for _, sub := range subscribers {
		sub.OnError(err)
	}
}

// Complete completes the flow
func (f *Flow) Complete() {
	if !atomic.CompareAndSwapInt32(&f.closed, 0, 1) {
		return
	}

	<-f.mu
	subscribers := make([]Subscriber, len(f.subscribers))
	copy(subscribers, f.subscribers)
	f.mu <- struct{}{}

	for _, sub := range subscribers {
		sub.OnComplete()
	}

	if f.cancel != nil {
		f.cancel()
	}
}

// Close closes the flow
func (f *Flow) Close() {
	f.Complete()
}

// IsClosed returns true if the flow is closed
func (f *Flow) IsClosed() bool {
	return atomic.LoadInt32(&f.closed) == 1
}

// subscription implements Subscription interface
type subscription struct {
	flow       *Flow
	subscriber Subscriber
	requested  int64 // Atomic counter for requested items
	cancelled  int32 // Atomic flag for cancelled state
	mu         sync.Mutex
}

func newSubscription(flow *Flow, subscriber Subscriber) *subscription {
	return &subscription{
		flow:       flow,
		subscriber: subscriber,
		requested:  0,
	}
}

// Request requests n items from the publisher
func (s *subscription) Request(n int64) {
	if n <= 0 {
		s.subscriber.OnError(ErrInvalidRequest)
		return
	}

	if atomic.LoadInt32(&s.cancelled) == 1 {
		return
	}

	atomic.AddInt64(&s.requested, n)
}

// Cancel cancels the subscription
func (s *subscription) Cancel() {
	if !atomic.CompareAndSwapInt32(&s.cancelled, 0, 1) {
		return
	}

	// Remove from flow's subscribers
	s.mu.Lock()
	<-s.flow.mu
	subscribers := s.flow.subscribers
	newSubscribers := make([]Subscriber, 0, len(subscribers))
	for _, sub := range subscribers {
		if sub != s.subscriber {
			newSubscribers = append(newSubscribers, sub)
		}
	}
	s.flow.subscribers = newSubscribers
	s.flow.mu <- struct{}{}
	s.mu.Unlock()
}

// GetRequested returns the number of requested items
func (s *subscription) GetRequested() int64 {
	return atomic.LoadInt64(&s.requested)
}

// IsCancelled returns true if the subscription is cancelled
func (s *subscription) IsCancelled() bool {
	return atomic.LoadInt32(&s.cancelled) == 1
}

// ConsumeRequested consumes n items from the requested count
func (s *subscription) ConsumeRequested(n int64) {
	atomic.AddInt64(&s.requested, -n)
}
