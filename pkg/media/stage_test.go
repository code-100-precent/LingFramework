package media

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPipelineStage_String(t *testing.T) {
	session := NewDefaultSession()
	stage := &PipelineStage{
		index:   0,
		session: session,
	}

	str := stage.String()
	if str == "" {
		t.Error("expected non-empty string representation")
	}
}

func TestPipelineStage_GetContext(t *testing.T) {
	session := NewDefaultSession()
	stage := &PipelineStage{
		index:   0,
		session: session,
	}

	ctx := stage.GetContext()
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	// Test with custom context
	parentCtx := context.Background()
	session.Context(parentCtx)
	stage.ctx, stage.cancel = context.WithCancel(parentCtx)
	ctx2 := stage.GetContext()
	if ctx2 != stage.ctx {
		t.Error("expected custom context")
	}
}

func TestPipelineStage_GetSession(t *testing.T) {
	session := NewDefaultSession()
	stage := &PipelineStage{
		index:   0,
		session: session,
	}

	if stage.GetSession() != session {
		t.Error("expected session to match")
	}
}

func TestPipelineStage_InjectPacket(t *testing.T) {
	session := NewDefaultSession()
	stage := &PipelineStage{
		index:   0,
		session: session,
	}

	filter := func(packet MediaPacket) (bool, error) {
		return false, nil
	}

	stage.InjectPacket(filter)
	if stage.preFilter == nil {
		t.Error("expected pre-filter to be set")
	}
}

func TestPipelineStage_CauseError(t *testing.T) {
	session := NewDefaultSession()
	stage := &PipelineStage{
		index:   0,
		session: session,
	}

	errorCalled := false
	session.Error(func(sender any, err error) {
		errorCalled = true
	})

	testErr := errors.New("test error")
	stage.CauseError(stage, testErr)

	time.Sleep(50 * time.Millisecond)

	if !errorCalled {
		t.Error("expected error handlers to be called")
	}
}

func TestPipelineStage_EmitState(t *testing.T) {
	session := NewDefaultSession()
	stage := &PipelineStage{
		index:   0,
		session: session,
	}

	stateCalled := false
	session.On("test_state", func(event StateChange) {
		stateCalled = true
	})

	stage.EmitState(stage, "test_state", "param1")

	time.Sleep(50 * time.Millisecond)

	if !stateCalled {
		t.Error("expected state handlers to be called")
	}
}

func TestPipelineStage_EmitPacket_WithWorker(t *testing.T) {
	session := NewDefaultSession()
	stage := &PipelineStage{
		index:   0,
		session: session,
	}

	// Start worker
	stage.startWorker(10)
	defer stage.stopWorker()

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	stage.EmitPacket("sender", packet)

	// Wait for packet to be processed
	time.Sleep(100 * time.Millisecond)
}

func TestPipelineStage_EmitPacket_WithoutWorker(t *testing.T) {
	session := NewDefaultSession()
	stage := &PipelineStage{
		index:   0,
		session: session,
	}

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	stage.EmitPacket("sender", packet)

	// Should not panic
}

func TestPipelineStage_AddMetric(t *testing.T) {
	session := NewDefaultSession()
	stage := &PipelineStage{
		index:   0,
		session: session,
	}

	stage.AddMetric("test", 100*time.Millisecond)

	// Should not panic
}

func TestPipelineStage_SendToOutput(t *testing.T) {
	session := NewDefaultSession()
	transport := newMockTransport()
	session.AddOutputTransport(transport)

	// Start a goroutine to process outgoing packets
	go func() {
		for _, output := range session.outputs {
			if output.txqueue != nil {
				select {
				case packet := <-output.txqueue:
					if packet != nil && output.transport != nil {
						output.transport.Send(context.Background(), packet)
					}
				case <-time.After(100 * time.Millisecond):
					return
				}
			}
		}
	}()

	stage := &PipelineStage{
		index:   0,
		session: session,
	}

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	stage.SendToOutput("sender", packet)

	// Wait for async output processing
	time.Sleep(200 * time.Millisecond)

	sentPackets := transport.getSentPackets()
	if len(sentPackets) == 0 {
		t.Error("expected packet to be sent")
	}

	session.Close()
}

func TestPipelineStage_StartWorker(t *testing.T) {
	session := NewDefaultSession()
	stage := &PipelineStage{
		index:   0,
		session: session,
	}

	stage.startWorker(10)

	if !stage.workerRunning {
		t.Error("expected worker to be running")
	}
	if stage.eventQueue == nil {
		t.Error("expected event queue to be created")
	}

	stage.stopWorker()

	if stage.workerRunning {
		t.Error("expected worker to be stopped")
	}
}

func TestPipelineStage_StopWorker(t *testing.T) {
	session := NewDefaultSession()
	stage := &PipelineStage{
		index:   0,
		session: session,
	}

	// Stop without starting should not panic
	stage.stopWorker()

	// Start then stop
	stage.startWorker(10)
	stage.stopWorker()

	if stage.workerRunning {
		t.Error("expected worker to be stopped")
	}
}

func TestPipelineStage_ProcessPacketAsync_Terminal(t *testing.T) {
	session := NewDefaultSession()
	transport := newMockTransport()
	session.AddOutputTransport(transport)

	stage := &PipelineStage{
		index:      0,
		session:    session,
		middleware: nil, // Terminal stage
	}

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	stage.processPacketAsync("sender", packet)

	// putPacket puts packet into txqueue, but it needs to be processed
	// Since we're not running Serve(), we need to manually check the queue
	// or just verify that putPacket was called (which it was)
	// The actual sending happens in processOutgoing which requires Serve()

	// For this test, we just verify the method doesn't panic
	// The actual transport sending requires a running session
	time.Sleep(50 * time.Millisecond)

	session.Close()
}

func TestPipelineStage_ProcessPacketAsync_WithPreFilter(t *testing.T) {
	session := NewDefaultSession()
	stage := &PipelineStage{
		index:   0,
		session: session,
	}

	skipCalled := false
	stage.preFilter = func(packet MediaPacket) (bool, error) {
		skipCalled = true
		return true, nil // Skip packet
	}

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	stage.processPacketAsync("sender", packet)

	if !skipCalled {
		t.Error("expected pre-filter to be called")
	}
}

func TestPipelineStage_ProcessPacketAsync_WithPreFilterError(t *testing.T) {
	session := NewDefaultSession()
	stage := &PipelineStage{
		index:   0,
		session: session,
	}

	errorCalled := false
	session.Error(func(sender any, err error) {
		errorCalled = true
	})

	stage.preFilter = func(packet MediaPacket) (bool, error) {
		return false, errors.New("filter error")
	}

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	stage.processPacketAsync("sender", packet)

	time.Sleep(50 * time.Millisecond)

	if !errorCalled {
		t.Error("expected error handlers to be called")
	}
}

func TestPipelineStage_ProcessPacketAsync_WithMiddleware(t *testing.T) {
	session := NewDefaultSession()
	stage := &PipelineStage{
		index:   0,
		session: session,
	}

	middlewareCalled := false
	stage.middleware = func(h MediaHandler, data MediaData) {
		middlewareCalled = true
	}

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	stage.processPacketAsync("sender", packet)

	if !middlewareCalled {
		t.Error("expected middleware to be called")
	}
}

func TestPipelineStage_ProcessPacketAsync_WithPostFilter(t *testing.T) {
	session := NewDefaultSession()
	stage := &PipelineStage{
		index:   0,
		session: session,
	}

	postFilterCalled := false
	stage.postFilter = func(packet MediaPacket) (bool, error) {
		postFilterCalled = true
		return false, nil
	}

	stage.middleware = func(h MediaHandler, data MediaData) {
		// Do nothing
	}

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	stage.processPacketAsync("sender", packet)

	if !postFilterCalled {
		t.Error("expected post-filter to be called")
	}
}

func TestPipelineStage_ProcessPacketAsync_WithNextStage(t *testing.T) {
	session := NewDefaultSession()
	stage1 := &PipelineStage{
		index:   0,
		session: session,
	}
	stage2 := &PipelineStage{
		index:   1,
		session: session,
	}
	stage1.nextStage = stage2

	stage2.startWorker(10)
	defer stage2.stopWorker()

	stage1.middleware = func(h MediaHandler, data MediaData) {
		// Do nothing
	}

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	stage1.processPacketAsync("sender", packet)

	time.Sleep(100 * time.Millisecond)
}

func TestPipelineStage_EnqueueEvent(t *testing.T) {
	session := NewDefaultSession()
	stage := &PipelineStage{
		index:   0,
		session: session,
	}

	// Enqueue without worker should not panic
	data := &MediaData{
		Type:   MediaDataTypePacket,
		Packet: &AudioPacket{Payload: []byte{1, 2, 3}},
	}
	stage.enqueueEvent(data)

	// Start worker and enqueue
	stage.startWorker(10)
	defer stage.stopWorker()

	stage.enqueueEvent(data)
	time.Sleep(50 * time.Millisecond)
}

func TestPipelineStage_ProcessEvent(t *testing.T) {
	session := NewDefaultSession()
	stage := &PipelineStage{
		index:   0,
		session: session,
	}

	middlewareCalled := false
	stage.middleware = func(h MediaHandler, data MediaData) {
		middlewareCalled = true
	}

	// Test packet event
	packetData := &MediaData{
		Type:   MediaDataTypePacket,
		Packet: &AudioPacket{Payload: []byte{1, 2, 3}},
	}
	stage.processEvent(packetData)

	if !middlewareCalled {
		t.Error("expected middleware to be called for packet event")
	}

	// Test state event
	middlewareCalled = false
	stateData := &MediaData{
		Type:  MediaDataTypeState,
		State: StateChange{State: "test"},
	}
	stage.processEvent(stateData)

	if !middlewareCalled {
		t.Error("expected middleware to be called for state event")
	}
}
