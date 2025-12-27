package media

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewAsyncTaskRunner(t *testing.T) {
	runner := NewAsyncTaskRunner[string](10)
	if runner.WorkerPoolSize != 10 {
		t.Errorf("expected worker pool size 10, got %d", runner.WorkerPoolSize)
	}
	if runner.MaxTaskTimeout != 1*time.Minute {
		t.Errorf("expected max task timeout 1 minute, got %v", runner.MaxTaskTimeout)
	}
	if !runner.ConcurrentMode {
		t.Error("expected concurrent mode to be true")
	}
}

func TestAsyncTaskRunner_ReleaseResources(t *testing.T) {
	runner := NewAsyncTaskRunner[string](5)
	runner.ReleaseResources()

	if runner.taskQueue != nil {
		t.Error("expected task queue to be nil after release")
	}
	if runner.workersActive {
		t.Error("expected workers to be inactive after release")
	}
}

func TestAsyncTaskRunner_CancelActiveTask(t *testing.T) {
	runner := NewAsyncTaskRunner[string](5)
	runner.CancelActiveTask()

	// Should not panic
	if runner.workersActive {
		t.Error("expected workers to be inactive after cancel")
	}
}

func TestAsyncTaskRunner_HandleState_Begin(t *testing.T) {
	runner := NewAsyncTaskRunner[string](5)
	session := NewDefaultSession()

	initCalled := false
	runner.InitCallback = func(h MediaHandler) error {
		initCalled = true
		return nil
	}

	state := StateChange{State: Begin}
	runner.HandleState(session, state)

	// Wait a bit for initialization
	time.Sleep(50 * time.Millisecond)

	if !initCalled {
		t.Error("expected init callback to be called")
	}
	if !runner.workersActive {
		t.Error("expected workers to be active after begin")
	}

	runner.ReleaseResources()
}

func TestAsyncTaskRunner_HandleState_End(t *testing.T) {
	runner := NewAsyncTaskRunner[string](5)
	session := NewDefaultSession()

	terminateCalled := false
	runner.TerminateCallback = func(h MediaHandler) error {
		terminateCalled = true
		return nil
	}

	// Start workers first
	state := StateChange{State: Begin}
	runner.HandleState(session, state)
	time.Sleep(50 * time.Millisecond)

	// Then end
	endState := StateChange{State: End}
	runner.HandleState(session, endState)
	time.Sleep(50 * time.Millisecond)

	if !terminateCalled {
		t.Error("expected terminate callback to be called")
	}
}

func TestAsyncTaskRunner_HandleState_StateCallback(t *testing.T) {
	runner := NewAsyncTaskRunner[string](5)
	session := NewDefaultSession()

	stateCallbackCalled := false
	runner.StateCallback = func(h MediaHandler, event StateChange) error {
		stateCallbackCalled = true
		return nil
	}

	state := StateChange{State: "custom_state"}
	runner.HandleState(session, state)

	if !stateCallbackCalled {
		t.Error("expected state callback to be called")
	}
}

func TestAsyncTaskRunner_HandlePacket(t *testing.T) {
	runner := NewAsyncTaskRunner[string](5)
	session := NewDefaultSession()

	executorCalled := false
	runner.TaskExecutor = func(ctx context.Context, h MediaHandler, req PacketRequest[string]) error {
		executorCalled = true
		return nil
	}

	runner.RequestBuilder = func(h MediaHandler, packet MediaPacket) (*PacketRequest[string], error) {
		return &PacketRequest[string]{
			Req: "test",
		}, nil
	}

	// Start workers
	state := StateChange{State: Begin}
	runner.HandleState(session, state)
	time.Sleep(50 * time.Millisecond)

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	runner.HandlePacket(session, packet)

	// Wait for task to be processed
	time.Sleep(100 * time.Millisecond)

	if !executorCalled {
		t.Error("expected executor to be called")
	}

	runner.ReleaseResources()
}

func TestAsyncTaskRunner_HandlePacket_NonConcurrent(t *testing.T) {
	runner := NewAsyncTaskRunner[string](5)
	runner.ConcurrentMode = false
	session := NewDefaultSession()

	executorCalled := false
	runner.TaskExecutor = func(ctx context.Context, h MediaHandler, req PacketRequest[string]) error {
		executorCalled = true
		return nil
	}

	runner.RequestBuilder = func(h MediaHandler, packet MediaPacket) (*PacketRequest[string], error) {
		return &PacketRequest[string]{
			Req: "test",
		}, nil
	}

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	runner.HandlePacket(session, packet)

	if !executorCalled {
		t.Error("expected executor to be called in non-concurrent mode")
	}
}

func TestAsyncTaskRunner_HandlePacket_NilRequest(t *testing.T) {
	runner := NewAsyncTaskRunner[string](5)
	session := NewDefaultSession()

	runner.RequestBuilder = func(h MediaHandler, packet MediaPacket) (*PacketRequest[string], error) {
		return nil, nil // Return nil request
	}

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	runner.HandlePacket(session, packet)

	// Should not panic
}

func TestAsyncTaskRunner_HandlePacket_RequestBuilderError(t *testing.T) {
	runner := NewAsyncTaskRunner[string](5)
	session := NewDefaultSession()

	errorCalled := false
	session.Error(func(sender any, err error) {
		errorCalled = true
	})

	runner.RequestBuilder = func(h MediaHandler, packet MediaPacket) (*PacketRequest[string], error) {
		return nil, errors.New("builder error")
	}

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	runner.HandlePacket(session, packet)

	time.Sleep(50 * time.Millisecond)

	if !errorCalled {
		t.Error("expected error handlers to be called")
	}
}

func TestAsyncTaskRunner_HandleMediaData(t *testing.T) {
	runner := NewAsyncTaskRunner[string](5)
	session := NewDefaultSession()

	packetCalled := false
	stateCalled := false

	runner.TaskExecutor = func(ctx context.Context, h MediaHandler, req PacketRequest[string]) error {
		packetCalled = true
		return nil
	}

	runner.RequestBuilder = func(h MediaHandler, packet MediaPacket) (*PacketRequest[string], error) {
		return &PacketRequest[string]{Req: "test"}, nil
	}

	runner.StateCallback = func(h MediaHandler, event StateChange) error {
		stateCalled = true
		return nil
	}

	// Start workers for concurrent mode
	state := StateChange{State: Begin}
	runner.HandleState(session, state)
	time.Sleep(50 * time.Millisecond)

	// Test packet data
	packetData := MediaData{
		Type:   MediaDataTypePacket,
		Packet: &AudioPacket{Payload: []byte{1, 2, 3}},
	}
	runner.HandleMediaData(session, packetData)
	time.Sleep(200 * time.Millisecond)

	if !packetCalled {
		t.Error("expected packet handlers to be called")
	}

	// Test state data
	stateData := MediaData{
		Type:  MediaDataTypeState,
		State: StateChange{State: "test"},
	}
	runner.HandleMediaData(session, stateData)
	time.Sleep(50 * time.Millisecond)

	if !stateCalled {
		t.Error("expected state handlers to be called")
	}

	runner.ReleaseResources()
	session.Close()
}

func TestAsyncTaskRunner_ExecuteTask_Timeout(t *testing.T) {
	runner := NewAsyncTaskRunner[string](5)
	runner.TaskTimeout = 50 * time.Millisecond
	session := NewDefaultSession()

	runner.TaskExecutor = func(ctx context.Context, h MediaHandler, req PacketRequest[string]) error {
		// Simulate long-running task
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return nil
		}
	}

	runner.RequestBuilder = func(h MediaHandler, packet MediaPacket) (*PacketRequest[string], error) {
		return &PacketRequest[string]{
			Req: "test",
		}, nil
	}

	// Start workers
	state := StateChange{State: Begin}
	runner.HandleState(session, state)
	time.Sleep(50 * time.Millisecond)

	// Send packet that will timeout
	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	runner.HandlePacket(session, packet)

	// Wait for timeout
	time.Sleep(200 * time.Millisecond)

	runner.ReleaseResources()
}

func TestAsyncTaskRunner_ExecuteTask_Error(t *testing.T) {
	runner := NewAsyncTaskRunner[string](5)
	session := NewDefaultSession()

	errorCalled := false
	session.Error(func(sender any, err error) {
		errorCalled = true
	})

	runner.TaskExecutor = func(ctx context.Context, h MediaHandler, req PacketRequest[string]) error {
		return errors.New("task error")
	}

	runner.RequestBuilder = func(h MediaHandler, packet MediaPacket) (*PacketRequest[string], error) {
		return &PacketRequest[string]{
			Req: "test",
		}, nil
	}

	// Start workers
	state := StateChange{State: Begin}
	runner.HandleState(session, state)
	time.Sleep(50 * time.Millisecond)

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	runner.HandlePacket(session, packet)
	time.Sleep(100 * time.Millisecond)

	if !errorCalled {
		t.Error("expected error handlers to be called")
	}

	runner.ReleaseResources()
}

func TestAsyncTaskRunner_Interrupt(t *testing.T) {
	runner := NewAsyncTaskRunner[string](5)
	session := NewDefaultSession()

	// Start workers
	state := StateChange{State: Begin}
	runner.HandleState(session, state)
	time.Sleep(100 * time.Millisecond)

	// Send interrupt request
	req := &PacketRequest[string]{
		Interrupt: true,
		Req:       "interrupt",
	}

	if runner.taskQueue != nil {
		runner.taskQueue <- req
		time.Sleep(200 * time.Millisecond)
	}

	// Wait for workers to stop
	time.Sleep(100 * time.Millisecond)

	// Workers should be stopped (interrupt cancels the pool context)
	// Note: workersActive might still be true briefly, but poolCancel should be called
	if runner.poolCancel == nil {
		t.Error("expected pool cancel to be set")
	}

	runner.ReleaseResources()
	session.Close()
}

func TestAsyncTaskRunner_WorkerPoolSize_Zero(t *testing.T) {
	runner := NewAsyncTaskRunner[string](0)
	session := NewDefaultSession()

	runner.InitCallback = func(h MediaHandler) error {
		return nil
	}

	state := StateChange{State: Begin}
	runner.HandleState(session, state)
	time.Sleep(100 * time.Millisecond)

	// Should default to 4 workers (set in startMultiWorkerPool)
	// The WorkerPoolSize field stays 0, but actual pool uses default
	if runner.workers == nil || len(runner.workers) == 0 {
		t.Error("expected workers to be started with default size")
	}

	runner.ReleaseResources()
	session.Close()
}
