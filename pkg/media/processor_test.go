package media

import (
	"context"
	"testing"
	"time"
)

func TestProcessorPriority(t *testing.T) {
	if PriorityLow >= PriorityNormal || PriorityNormal >= PriorityHigh {
		t.Error("priority constants should be in ascending order")
	}
}

func TestNewProcessorRegistry(t *testing.T) {
	registry := NewProcessorRegistry()
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
	if len(registry.GetAllProcessors()) != 0 {
		t.Error("expected empty registry")
	}
}

func TestProcessorRegistry_Register(t *testing.T) {
	registry := NewProcessorRegistry()

	// Register processors with different priorities
	low := NewFuncProcessor("low", PriorityLow, func(ctx context.Context, session *MediaSession, event *MediaEvent) error {
		return nil
	})
	normal := NewFuncProcessor("normal", PriorityNormal, func(ctx context.Context, session *MediaSession, event *MediaEvent) error {
		return nil
	})
	high := NewFuncProcessor("high", PriorityHigh, func(ctx context.Context, session *MediaSession, event *MediaEvent) error {
		return nil
	})

	registry.Register(low)
	registry.Register(normal)
	registry.Register(high)

	processors := registry.GetAllProcessors()
	if len(processors) != 3 {
		t.Fatalf("expected 3 processors, got %d", len(processors))
	}

	// Check priority order (high first)
	if processors[0].Priority() != PriorityHigh {
		t.Errorf("expected first processor to have high priority, got %d", processors[0].Priority())
	}
	if processors[1].Priority() != PriorityNormal {
		t.Errorf("expected second processor to have normal priority, got %d", processors[1].Priority())
	}
	if processors[2].Priority() != PriorityLow {
		t.Errorf("expected third processor to have low priority, got %d", processors[2].Priority())
	}
}

func TestProcessorRegistry_Unregister(t *testing.T) {
	registry := NewProcessorRegistry()

	proc := NewFuncProcessor("test", PriorityNormal, func(ctx context.Context, session *MediaSession, event *MediaEvent) error {
		return nil
	})

	registry.Register(proc)
	if len(registry.GetAllProcessors()) != 1 {
		t.Error("expected 1 processor")
	}

	registry.Unregister("test")
	if len(registry.GetAllProcessors()) != 0 {
		t.Error("expected 0 processors after unregister")
	}

	// Unregister non-existent processor
	registry.Unregister("nonexistent")
}

func TestProcessorRegistry_GetProcessors(t *testing.T) {
	registry := NewProcessorRegistry()
	session := NewDefaultSession()

	// Create processor with condition
	called := false
	proc := NewBaseProcessor("conditional", PriorityNormal)
	proc.WithCondition(func(ctx context.Context, event *MediaEvent) bool {
		return event.Type == EventTypePacket
	})

	funcProc := &FuncProcessor{
		BaseProcessor: proc,
		processFunc: func(ctx context.Context, session *MediaSession, event *MediaEvent) error {
			called = true
			return nil
		},
	}

	registry.Register(funcProc)

	// Test with matching event
	event := &MediaEvent{
		Type:      EventTypePacket,
		Timestamp: time.Now(),
		SessionID: session.ID,
		Payload:   &AudioPacket{Payload: []byte{1, 2, 3}},
	}

	processors := registry.GetProcessors(context.Background(), event)
	if len(processors) != 1 {
		t.Fatalf("expected 1 processor, got %d", len(processors))
	}

	// Test with non-matching event
	stateEvent := &MediaEvent{
		Type:      EventTypeState,
		Timestamp: time.Now(),
		SessionID: session.ID,
		Payload:   StateChange{State: "begin"},
	}

	processors = registry.GetProcessors(context.Background(), stateEvent)
	if len(processors) != 0 {
		t.Errorf("expected 0 processors for non-matching event, got %d", len(processors))
	}

	// Process event to verify it works
	if len(processors) > 0 {
		if err := processors[0].Process(context.Background(), session, event); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected processor to be called")
		}
	}
}

func TestBaseProcessor(t *testing.T) {
	bp := NewBaseProcessor("test", PriorityNormal)
	if bp.Name() != "test" {
		t.Errorf("expected name 'test', got '%s'", bp.Name())
	}
	if bp.Priority() != PriorityNormal {
		t.Errorf("expected priority %d, got %d", PriorityNormal, bp.Priority())
	}

	// Test CanHandle without condition
	event := &MediaEvent{Type: EventTypePacket}
	if !bp.CanHandle(context.Background(), event) {
		t.Error("expected CanHandle to return true without condition")
	}

	// Test with condition
	bp.WithCondition(func(ctx context.Context, event *MediaEvent) bool {
		return event.Type == EventTypePacket
	})
	if !bp.CanHandle(context.Background(), event) {
		t.Error("expected CanHandle to return true with matching condition")
	}

	stateEvent := &MediaEvent{Type: EventTypeState}
	if bp.CanHandle(context.Background(), stateEvent) {
		t.Error("expected CanHandle to return false with non-matching condition")
	}
}

func TestFuncProcessor(t *testing.T) {
	called := false
	session := NewDefaultSession()

	proc := NewFuncProcessor("test", PriorityNormal, func(ctx context.Context, s *MediaSession, event *MediaEvent) error {
		called = true
		return nil
	})

	if proc.Name() != "test" {
		t.Errorf("expected name 'test', got '%s'", proc.Name())
	}

	event := &MediaEvent{
		Type:      EventTypePacket,
		Timestamp: time.Now(),
		SessionID: session.ID,
	}

	if err := proc.Process(context.Background(), session, event); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected processor function to be called")
	}
}

func TestPacketProcessor(t *testing.T) {
	session := NewDefaultSession()
	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	called := false

	proc := NewPacketProcessor("test", PriorityNormal, func(ctx context.Context, s *MediaSession, p MediaPacket) error {
		called = true
		if p != packet {
			t.Error("expected packet to match")
		}
		return nil
	})

	// Test CanHandle with packet event
	event := &MediaEvent{
		Type:      EventTypePacket,
		Timestamp: time.Now(),
		SessionID: session.ID,
		Payload:   packet,
	}

	if !proc.CanHandle(context.Background(), event) {
		t.Error("expected CanHandle to return true for packet event")
	}

	// Test CanHandle with non-packet event
	stateEvent := &MediaEvent{
		Type:      EventTypeState,
		Timestamp: time.Now(),
		SessionID: session.ID,
	}

	if proc.CanHandle(context.Background(), stateEvent) {
		t.Error("expected CanHandle to return false for non-packet event")
	}

	// Test Process
	if err := proc.Process(context.Background(), session, event); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected processor to be called")
	}

	// Test Process with non-packet payload
	badEvent := &MediaEvent{
		Type:      EventTypePacket,
		Timestamp: time.Now(),
		SessionID: session.ID,
		Payload:   "not a packet",
	}

	err := proc.Process(context.Background(), session, badEvent)
	if err == nil {
		t.Error("expected error for non-packet payload")
	}
}

func TestNewHighPriorityProcessor(t *testing.T) {
	proc := NewHighPriorityProcessor("test", func(ctx context.Context, session *MediaSession, event *MediaEvent) error {
		return nil
	})

	if proc.Priority() != PriorityHigh {
		t.Errorf("expected priority %d, got %d", PriorityHigh, proc.Priority())
	}
}

func TestNewHighPriorityPacketProcessor(t *testing.T) {
	proc := NewHighPriorityPacketProcessor("test", func(ctx context.Context, session *MediaSession, packet MediaPacket) error {
		return nil
	})

	if proc.Priority() != PriorityHigh {
		t.Errorf("expected priority %d, got %d", PriorityHigh, proc.Priority())
	}
}

func TestProcessorRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewProcessorRegistry()
	session := NewDefaultSession()

	// Register multiple processors concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			proc := NewFuncProcessor("proc"+string(rune(id)), PriorityNormal, func(ctx context.Context, s *MediaSession, event *MediaEvent) error {
				return nil
			})
			registry.Register(proc)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	processors := registry.GetAllProcessors()
	if len(processors) != 10 {
		t.Errorf("expected 10 processors, got %d", len(processors))
	}

	// Test concurrent GetProcessors
	event := &MediaEvent{
		Type:      EventTypePacket,
		Timestamp: time.Now(),
		SessionID: session.ID,
		Payload:   &AudioPacket{Payload: []byte{1, 2, 3}},
	}

	for i := 0; i < 10; i++ {
		go func() {
			processors := registry.GetProcessors(context.Background(), event)
			if len(processors) == 0 {
				t.Error("expected at least one processor")
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
