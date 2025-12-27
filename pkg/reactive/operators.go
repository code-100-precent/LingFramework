package reactive

import (
	"context"
	"sync"
)

// Map applies a transformation function to each item in the stream
func Map[T any, R any](publisher Publisher, mapper func(T) R) Publisher {
	return &MapOperator[T, R]{
		source: publisher,
		mapper: mapper,
	}
}

// MapOperator implements the Map operator
type MapOperator[T any, R any] struct {
	source Publisher
	mapper func(T) R
}

func (m *MapOperator[T, R]) Subscribe(subscriber Subscriber) Subscription {
	downstream := &mapSubscriber[T, R]{
		downstream: subscriber,
		mapper:     m.mapper,
	}
	return m.source.Subscribe(downstream)
}

type mapSubscriber[T any, R any] struct {
	downstream   Subscriber
	mapper       func(T) R
	subscription Subscription
}

func (s *mapSubscriber[T, R]) OnSubscribe(subscription Subscription) {
	s.subscription = subscription
	s.downstream.OnSubscribe(subscription)
}

func (s *mapSubscriber[T, R]) OnNext(value interface{}) {
	if t, ok := value.(T); ok {
		mapped := s.mapper(t)
		s.downstream.OnNext(mapped)
	}
}

func (s *mapSubscriber[T, R]) OnError(err error) {
	s.downstream.OnError(err)
}

func (s *mapSubscriber[T, R]) OnComplete() {
	s.downstream.OnComplete()
}

// Filter filters items based on a predicate function
func Filter[T any](publisher Publisher, predicate func(T) bool) Publisher {
	return &FilterOperator[T]{
		source:    publisher,
		predicate: predicate,
	}
}

// FilterOperator implements the Filter operator
type FilterOperator[T any] struct {
	source    Publisher
	predicate func(T) bool
}

func (f *FilterOperator[T]) Subscribe(subscriber Subscriber) Subscription {
	downstream := &filterSubscriber[T]{
		downstream: subscriber,
		predicate:  f.predicate,
	}
	return f.source.Subscribe(downstream)
}

type filterSubscriber[T any] struct {
	downstream   Subscriber
	predicate    func(T) bool
	subscription Subscription
}

func (s *filterSubscriber[T]) OnSubscribe(subscription Subscription) {
	s.subscription = subscription
	s.downstream.OnSubscribe(subscription)
}

func (s *filterSubscriber[T]) OnNext(value interface{}) {
	if t, ok := value.(T); ok {
		if s.predicate(t) {
			s.downstream.OnNext(value)
		}
		// Don't request more items when filtering - the upstream handles the request flow
		// Requesting here causes infinite loops because it triggers OnNext again
	}
}

func (s *filterSubscriber[T]) OnError(err error) {
	s.downstream.OnError(err)
}

func (s *filterSubscriber[T]) OnComplete() {
	s.downstream.OnComplete()
}

// FlatMap transforms each item into a Publisher and flattens the results
// Note: This is a simplified implementation. For production use, consider a more robust approach.
func FlatMap[T any, R any](publisher Publisher, mapper func(T) Publisher) Publisher {
	return &FlatMapOperator[T, R]{
		source: publisher,
		mapper: mapper,
	}
}

// FlatMapOperator implements a simplified FlatMap operator
type FlatMapOperator[T any, R any] struct {
	source Publisher
	mapper func(T) Publisher
}

func (f *FlatMapOperator[T, R]) Subscribe(subscriber Subscriber) Subscription {
	// Simplified: collect all items, then map and flatten
	// For async handling, this would need a more complex implementation
	collector := &collectorSubscriber{
		done: make(chan struct{}),
	}
	upstream := f.source.Subscribe(collector)

	go func() {
		// Wait for source to complete
		<-collector.done

		// Process collected items
		for _, item := range collector.items {
			if t, ok := item.(T); ok {
				innerPub := f.mapper(t)
				innerSub := &flattenSubscriber{
					downstream: subscriber,
				}
				innerPub.Subscribe(innerSub)
			}
		}
		subscriber.OnComplete()
	}()

	return upstream
}

type collectorSubscriber struct {
	items []interface{}
	done  chan struct{}
}

func (c *collectorSubscriber) OnSubscribe(subscription Subscription) {
	c.done = make(chan struct{})
	subscription.Request(1000) // Request many items
}

func (c *collectorSubscriber) OnNext(value interface{}) {
	c.items = append(c.items, value)
}

func (c *collectorSubscriber) OnError(err error) {
	close(c.done)
}

func (c *collectorSubscriber) OnComplete() {
	close(c.done)
}

type flattenSubscriber struct {
	downstream   Subscriber
	subscription Subscription
}

func (f *flattenSubscriber) OnSubscribe(subscription Subscription) {
	f.subscription = subscription
	f.downstream.OnSubscribe(subscription)
	subscription.Request(1000)
}

func (f *flattenSubscriber) OnNext(value interface{}) {
	f.downstream.OnNext(value)
}

func (f *flattenSubscriber) OnError(err error) {
	f.downstream.OnError(err)
}

func (f *flattenSubscriber) OnComplete() {
	// Don't complete downstream here - wait for all inner publishers
}

// Reduce applies a reducer function to accumulate values
func Reduce[T any, R any](publisher Publisher, initial R, reducer func(R, T) R) Publisher {
	return &ReduceOperator[T, R]{
		source:  publisher,
		initial: initial,
		reducer: reducer,
	}
}

// ReduceOperator implements the Reduce operator
type ReduceOperator[T any, R any] struct {
	source  Publisher
	initial R
	reducer func(R, T) R
}

func (r *ReduceOperator[T, R]) Subscribe(subscriber Subscriber) Subscription {
	downstream := &reduceSubscriber[T, R]{
		downstream:  subscriber,
		accumulator: r.initial,
		reducer:     r.reducer,
	}
	return r.source.Subscribe(downstream)
}

type reduceSubscriber[T any, R any] struct {
	downstream   Subscriber
	accumulator  R
	reducer      func(R, T) R
	subscription Subscription
}

func (s *reduceSubscriber[T, R]) OnSubscribe(subscription Subscription) {
	s.subscription = subscription
	s.downstream.OnSubscribe(subscription)
	// Don't request here - let downstream control the flow
}

func (s *reduceSubscriber[T, R]) OnNext(value interface{}) {
	if t, ok := value.(T); ok {
		s.accumulator = s.reducer(s.accumulator, t)
		// Don't request more items here - this causes infinite loops
		// The downstream subscriber controls when to request more items via its subscription
	}
}

func (s *reduceSubscriber[T, R]) OnError(err error) {
	s.downstream.OnError(err)
}

func (s *reduceSubscriber[T, R]) OnComplete() {
	s.downstream.OnNext(s.accumulator)
	s.downstream.OnComplete()
}

// Take takes the first n items from the stream
func Take[T any](publisher Publisher, n int64) Publisher {
	return &TakeOperator[T]{
		source: publisher,
		count:  n,
	}
}

// TakeOperator implements the Take operator
type TakeOperator[T any] struct {
	source Publisher
	count  int64
}

func (t *TakeOperator[T]) Subscribe(subscriber Subscriber) Subscription {
	downstream := &takeSubscriber[T]{
		downstream: subscriber,
		remaining:  t.count,
	}
	return t.source.Subscribe(downstream)
}

type takeSubscriber[T any] struct {
	downstream   Subscriber
	remaining    int64
	subscription Subscription
}

func (s *takeSubscriber[T]) OnSubscribe(subscription Subscription) {
	s.subscription = subscription
	s.downstream.OnSubscribe(subscription)
	if s.remaining > 0 {
		subscription.Request(s.remaining)
	} else {
		subscription.Cancel()
		s.downstream.OnComplete()
	}
}

func (s *takeSubscriber[T]) OnNext(value interface{}) {
	if s.remaining > 0 {
		s.downstream.OnNext(value)
		s.remaining--
		if s.remaining == 0 {
			if s.subscription != nil {
				s.subscription.Cancel()
			}
			s.downstream.OnComplete()
		}
	}
}

func (s *takeSubscriber[T]) OnError(err error) {
	s.downstream.OnError(err)
}

func (s *takeSubscriber[T]) OnComplete() {
	s.downstream.OnComplete()
}

// Buffer buffers items and emits them as batches
func Buffer[T any](publisher Publisher, size int) Publisher {
	return &BufferOperator[T]{
		source: publisher,
		size:   size,
	}
}

// BufferOperator implements the Buffer operator
type BufferOperator[T any] struct {
	source Publisher
	size   int
}

func (b *BufferOperator[T]) Subscribe(subscriber Subscriber) Subscription {
	downstream := &bufferSubscriber[T]{
		downstream: subscriber,
		buffer:     make([]T, 0, b.size),
		size:       b.size,
	}
	return b.source.Subscribe(downstream)
}

type bufferSubscriber[T any] struct {
	downstream   Subscriber
	buffer       []T
	size         int
	subscription Subscription
}

func (s *bufferSubscriber[T]) OnSubscribe(subscription Subscription) {
	s.subscription = subscription
	s.downstream.OnSubscribe(subscription)
	subscription.Request(int64(s.size))
}

func (s *bufferSubscriber[T]) OnNext(value interface{}) {
	if t, ok := value.(T); ok {
		s.buffer = append(s.buffer, t)
		if len(s.buffer) >= s.size {
			batch := make([]T, len(s.buffer))
			copy(batch, s.buffer)
			s.buffer = s.buffer[:0]
			s.downstream.OnNext(batch)
			if s.subscription != nil {
				s.subscription.Request(int64(s.size))
			}
		}
	}
}

func (s *bufferSubscriber[T]) OnError(err error) {
	if len(s.buffer) > 0 {
		s.downstream.OnNext(s.buffer)
		s.buffer = s.buffer[:0]
	}
	s.downstream.OnError(err)
}

func (s *bufferSubscriber[T]) OnComplete() {
	if len(s.buffer) > 0 {
		s.downstream.OnNext(s.buffer)
		s.buffer = s.buffer[:0]
	}
	s.downstream.OnComplete()
}

// FromSlice creates a Publisher from a slice
func FromSlice[T any](items []T) Publisher {
	return &SlicePublisher[T]{
		items: items,
	}
}

// SlicePublisher publishes items from a slice
type SlicePublisher[T any] struct {
	items []T
}

func (s *SlicePublisher[T]) Subscribe(subscriber Subscriber) Subscription {
	sub := &sliceSubscription[T]{
		items:      s.items,
		subscriber: subscriber,
		index:      0,
	}
	subscriber.OnSubscribe(sub)
	return sub
}

type sliceSubscription[T any] struct {
	items      []T
	subscriber Subscriber
	index      int
	requested  int64
	cancelled  bool
	mu         sync.Mutex
}

func (s *sliceSubscription[T]) Request(n int64) {
	s.mu.Lock()
	if s.cancelled {
		s.mu.Unlock()
		return
	}
	s.requested += n
	requested := s.requested
	index := s.index
	items := s.items
	subscriber := s.subscriber
	s.mu.Unlock()

	for requested > 0 && index < len(items) {
		subscriber.OnNext(items[index])
		index++
		requested--

		s.mu.Lock()
		s.index = index
		s.requested = requested
		if s.cancelled {
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()
	}

	s.mu.Lock()
	if index >= len(items) && !s.cancelled {
		s.cancelled = true
		subscriber.OnComplete()
	}
	s.mu.Unlock()
}

func (s *sliceSubscription[T]) Cancel() {
	s.mu.Lock()
	s.cancelled = true
	s.mu.Unlock()
}

// FromChannel creates a Publisher from a channel
func FromChannel[T any](ch <-chan T) Publisher {
	return &ChannelPublisher[T]{
		ch: ch,
	}
}

// ChannelPublisher publishes items from a channel
type ChannelPublisher[T any] struct {
	ch <-chan T
}

func (c *ChannelPublisher[T]) Subscribe(subscriber Subscriber) Subscription {
	ctx, cancel := context.WithCancel(context.Background())
	sub := &channelSubscription[T]{
		ch:         c.ch,
		subscriber: subscriber,
		cancel:     cancel,
		ctx:        ctx,
	}
	subscriber.OnSubscribe(sub)

	go sub.run()

	return sub
}

type channelSubscription[T any] struct {
	ch         <-chan T
	subscriber Subscriber
	cancel     context.CancelFunc
	ctx        context.Context
	requested  int64
	cancelled  bool
	mu         sync.Mutex
}

func (s *channelSubscription[T]) Request(n int64) {
	s.mu.Lock()
	s.requested += n
	s.mu.Unlock()
}

func (s *channelSubscription[T]) Cancel() {
	s.mu.Lock()
	s.cancelled = true
	s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *channelSubscription[T]) run() {
	defer func() {
		if r := recover(); r != nil {
			s.subscriber.OnError(r.(error))
		}
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		case item, ok := <-s.ch:
			if !ok {
				s.subscriber.OnComplete()
				return
			}

			s.mu.Lock()
			if s.cancelled {
				s.mu.Unlock()
				return
			}
			if s.requested <= 0 {
				s.mu.Unlock()
				continue
			}
			s.requested--
			s.mu.Unlock()

			s.subscriber.OnNext(item)
		}
	}
}
