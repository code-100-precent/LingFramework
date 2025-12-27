package media

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewEventBus(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx, 10, 2)

	if bus == nil {
		t.Fatal("expected non-nil event bus")
	}
	if bus.eventQueue == nil {
		t.Error("expected non-nil event queue")
	}

	// Cleanup
	bus.Close()
}

func TestEventBus_Subscribe(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx, 10, 1)
	defer bus.Close()

	called := false
	handler := func(ctx context.Context, event *MediaEvent) error {
		called = true
		return nil
	}

	bus.Subscribe(EventTypePacket, handler)

	event := &MediaEvent{
		Type:      EventTypePacket,
		Timestamp: time.Now(),
		SessionID: "test",
	}

	bus.Publish(event)

	// Wait for event to be processed
	time.Sleep(100 * time.Millisecond)

	if !called {
		t.Error("expected handlers to be called")
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx, 10, 1)
	defer bus.Close()

	callCount := 0
	handler := func(ctx context.Context, event *MediaEvent) error {
		callCount++
		return nil
	}

	bus.Subscribe(EventTypePacket, handler)

	event := &MediaEvent{
		Type:      EventTypePacket,
		Timestamp: time.Now(),
		SessionID: "test",
	}

	bus.Publish(event)
	time.Sleep(50 * time.Millisecond)

	if callCount != 1 {
		t.Errorf("expected call count 1, got %d", callCount)
	}

	bus.Unsubscribe(EventTypePacket, handler)

	bus.Publish(event)
	time.Sleep(50 * time.Millisecond)

	if callCount != 1 {
		t.Errorf("expected call count to remain 1 after unsubscribe, got %d", callCount)
	}
}

func TestEventBus_Publish(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx, 10, 2)
	defer bus.Close()

	var mu sync.Mutex
	callCount := 0
	bus.Subscribe(EventTypePacket, func(ctx context.Context, event *MediaEvent) error {
		mu.Lock()
		callCount++
		mu.Unlock()
		return nil
	})

	// Publish multiple events
	for i := 0; i < 5; i++ {
		event := &MediaEvent{
			Type:      EventTypePacket,
			Timestamp: time.Now(),
			SessionID: "test",
		}
		bus.Publish(event)
	}

	// Wait for events to be processed
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	count := callCount
	mu.Unlock()

	if count != 5 {
		t.Errorf("expected call count 5, got %d", count)
	}
}

func TestEventBus_QueueFull(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx, 1, 1)
	defer bus.Close()

	// Fill the queue
	event := &MediaEvent{
		Type:      EventTypePacket,
		Timestamp: time.Now(),
		SessionID: "test",
	}
	bus.Publish(event)
	bus.Publish(event)

	// This should not block (queue is full, event should be dropped)
	bus.Publish(event)

	time.Sleep(50 * time.Millisecond)
}

func TestEventBus_PublishPacket(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx, 10, 1)
	defer bus.Close()

	var receivedPacket MediaPacket
	bus.Subscribe(EventTypePacket, func(ctx context.Context, event *MediaEvent) error {
		if packet, ok := event.Payload.(MediaPacket); ok {
			receivedPacket = packet
		}
		return nil
	})

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	bus.PublishPacket("session1", packet, "sender1")

	time.Sleep(100 * time.Millisecond)

	if receivedPacket == nil {
		t.Fatal("expected packet to be received")
	}
	if receivedPacket != packet {
		t.Error("expected received packet to match sent packet")
	}
}

func TestEventBus_PublishState(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx, 10, 1)
	defer bus.Close()

	var receivedState StateChange
	bus.Subscribe(EventTypeState, func(ctx context.Context, event *MediaEvent) error {
		if state, ok := event.Payload.(StateChange); ok {
			receivedState = state
		}
		return nil
	})

	state := StateChange{State: "begin", Params: []any{"param1"}}
	bus.PublishState("session1", state, "sender1")

	time.Sleep(100 * time.Millisecond)

	if receivedState.State != "begin" {
		t.Errorf("expected state 'begin', got '%s'", receivedState.State)
	}
}

func TestEventBus_PublishError(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx, 10, 1)
	defer bus.Close()

	var receivedError error
	bus.Subscribe(EventTypeError, func(ctx context.Context, event *MediaEvent) error {
		if err, ok := event.Payload.(error); ok {
			receivedError = err
		}
		return nil
	})

	testErr := &testError{msg: "test error"}
	bus.PublishError("session1", testErr, "sender1")

	time.Sleep(100 * time.Millisecond)

	if receivedError == nil {
		t.Fatal("expected error to be received")
	}
	if receivedError.Error() != "test error" {
		t.Errorf("expected error message 'test error', got '%s'", receivedError.Error())
	}
}

func TestEventBus_HandlerPanic(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx, 10, 1)
	defer bus.Close()

	bus.Subscribe(EventTypePacket, func(ctx context.Context, event *MediaEvent) error {
		panic("handlers panic")
	})

	event := &MediaEvent{
		Type:      EventTypePacket,
		Timestamp: time.Now(),
		SessionID: "test",
	}

	// Should not panic
	bus.Publish(event)
	time.Sleep(100 * time.Millisecond)
}

func TestEventBus_HandlerError(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx, 10, 1)
	defer bus.Close()

	bus.Subscribe(EventTypePacket, func(ctx context.Context, event *MediaEvent) error {
		return &testError{msg: "handlers error"}
	})

	event := &MediaEvent{
		Type:      EventTypePacket,
		Timestamp: time.Now(),
		SessionID: "test",
	}

	// Should not panic, error should be logged
	bus.Publish(event)
	time.Sleep(100 * time.Millisecond)
}

func TestEventBus_LifecycleSubscribers(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx, 10, 1)
	defer bus.Close()

	packetCallCount := 0
	lifecycleCallCount := 0

	bus.Subscribe(EventTypePacket, func(ctx context.Context, event *MediaEvent) error {
		packetCallCount++
		return nil
	})

	bus.Subscribe(EventTypeLifecycle, func(ctx context.Context, event *MediaEvent) error {
		lifecycleCallCount++
		return nil
	})

	// Publish packet event - should trigger both packet and lifecycle handlers
	event := &MediaEvent{
		Type:      EventTypePacket,
		Timestamp: time.Now(),
		SessionID: "test",
	}
	bus.Publish(event)

	time.Sleep(100 * time.Millisecond)

	if packetCallCount != 1 {
		t.Errorf("expected packet call count 1, got %d", packetCallCount)
	}
	if lifecycleCallCount != 1 {
		t.Errorf("expected lifecycle call count 1, got %d", lifecycleCallCount)
	}
}

func TestEventBus_Close(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx, 10, 2)

	var mu sync.Mutex
	callCount := 0
	bus.Subscribe(EventTypePacket, func(ctx context.Context, event *MediaEvent) error {
		mu.Lock()
		callCount++
		mu.Unlock()
		return nil
	})

	// Publish some events
	for i := 0; i < 5; i++ {
		event := &MediaEvent{
			Type:      EventTypePacket,
			Timestamp: time.Now(),
			SessionID: "test",
		}
		bus.Publish(event)
	}

	// Wait a bit before closing
	time.Sleep(100 * time.Millisecond)

	// Close and wait
	bus.Close()
	time.Sleep(200 * time.Millisecond)

	// Verify events were processed
	mu.Lock()
	count := callCount
	mu.Unlock()

	if count != 5 {
		t.Errorf("expected call count 5, got %d", count)
	}
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx, 100, 4)
	defer bus.Close()

	var mu sync.Mutex
	callCount := 0

	bus.Subscribe(EventTypePacket, func(ctx context.Context, event *MediaEvent) error {
		mu.Lock()
		callCount++
		mu.Unlock()
		return nil
	})

	// Publish concurrently
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			event := &MediaEvent{
				Type:      EventTypePacket,
				Timestamp: time.Now(),
				SessionID: "test",
			}
			bus.Publish(event)
		}()
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	count := callCount
	mu.Unlock()

	if count != 50 {
		t.Errorf("expected call count 50, got %d", count)
	}
}

// Helper type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
