package media

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/code-100-precent/LingFramework/pkg/logger"
	"go.uber.org/zap"
)

// TransportManager manages transport connections
type TransportManager struct {
	session             *MediaSession
	txqueue             chan MediaPacket
	transport           MediaTransport
	filters             []PacketFilter
	mtx                 sync.Mutex
	incomingClosedChan  chan struct{}
	outcomingClosedChan chan struct{}
}

func (tl *TransportManager) String() string {
	return fmt.Sprintf("TransportManager{Session: %s, Transport: %s}", tl.session, tl.transport)
}

func (tl *TransportManager) processIncoming() {
	logger.Info("input transport processing started", zap.String("sessionID", tl.session.GetSession().ID), zap.Any("transport", tl.transport))
	tl.incomingClosedChan = make(chan struct{}, 1)
	defer func() {
		if r := recover(); r != nil {
			logger.Error("input transport processing panic", zap.String("sessionID", tl.session.GetSession().ID), zap.Any("transport", tl.transport), zap.Any("error", r), zap.String("stacktrace", string(debug.Stack())))
		}
		tl.incomingClosedChan <- struct{}{}
	}()

	transport := tl.transport
inputLoop:
	for tl.session.ctx.Err() == nil {
		packet, err := transport.Next(tl.session.ctx)
		if err != nil {
			if err != io.EOF {
				tl.session.CauseError(tl, err)
			} else {
				tl.session.EmitState(tl, Hangup)
			}
			break inputLoop
		}
		if packet == nil {
			continue
		}

		// Decode packet if decoder is configured
		var decodedPackets []MediaPacket
		if tl.session.decoder != nil {
			decodedPackets, err = tl.session.decoder(packet)
			if err != nil {
				tl.session.CauseError(tl, err)
				break inputLoop
			}
		} else {
			decodedPackets = []MediaPacket{packet}
		}

		// Apply filters and route packets
		for _, packet := range decodedPackets {
			if packet == nil {
				continue
			}
			shouldSkip := false
			for _, filter := range tl.filters {
				shouldSkip, err = filter(packet)
				if err != nil {
					tl.session.CauseError(tl, err)
					break inputLoop
				}
				if shouldSkip {
					break
				}
			}
			if !shouldSkip {
				tl.session.EmitPacket(tl, packet)
			}
		}
	}
	logger.Warn("input transport processing ended", zap.String("sessionID", tl.session.ID), zap.Any("transport", transport))
}

func (tl *TransportManager) processOutgoing() {
	if tl.txqueue == nil {
		panic("output queue is nil, transport manager not properly initialized")
	}
	tl.outcomingClosedChan = make(chan struct{}, 1)
	defer func() {
		if r := recover(); r != nil {
			logger.Error("output transport processing panic", zap.String("sessionID", tl.session.ID), zap.Any("transport", tl.transport), zap.Any("error", r), zap.String("stacktrace", string(debug.Stack())))
		}
		tl.outcomingClosedChan <- struct{}{}
	}()

	logger.Info("output transport processing started", zap.String("sessionID", tl.session.ID), zap.Any("transport", tl.transport))
outputLoop:
	for {
		var packet MediaPacket
		var ok bool
		var err error
		var shouldSkip = false
		select {
		case <-tl.session.ctx.Done():
			logger.Info("output transport processing canceled", zap.String("sessionID", tl.session.ID), zap.Any("transport", tl.transport))
			break outputLoop
		case packet, ok = <-tl.txqueue:
			if !ok || packet == nil {
				logger.Info("output transport queue closed", zap.String("sessionID", tl.session.ID), zap.Any("transport", tl.transport))
				break outputLoop
			}
		}

		// Apply output filters
		for _, filter := range tl.filters {
			shouldSkip, err = filter(packet)
			if shouldSkip {
				break
			}
			if err != nil {
				tl.session.CauseError(tl, err)
				break outputLoop
			}
		}

		if shouldSkip {
			continue
		}

		// Encode packet if encoder is configured
		var encodedPackets []MediaPacket
		if tl.session.encoder != nil {
			encodedPackets, err = tl.session.encoder(packet)
			if err != nil {
				tl.session.CauseError(tl, err)
				break outputLoop
			}
		} else {
			encodedPackets = []MediaPacket{packet}
		}

		// Send all encoded packets
		for _, encodedPacket := range encodedPackets {
			tl.transport.Send(tl.session.ctx, encodedPacket)
		}
	}
	logger.Warn("output transport processing ended", zap.String("sessionID", tl.session.ID))
}

func (tl *TransportManager) waitForIncomingLoopStop() {
	select {
	case <-tl.incomingClosedChan:
		return
	case <-time.After(5 * time.Second):
		return
	}
}

func (tl *TransportManager) waitForOutcomingLoopStop() {
	select {
	case <-tl.outcomingClosedChan:
		return
	case <-time.After(5 * time.Second):
		return
	}
}

func (tl *TransportManager) cleanup() {
	tl.mtx.Lock()
	defer tl.mtx.Unlock()
	if tl.transport == nil {
		return
	}

	tl.transport.Close()
	tl.waitForIncomingLoopStop()
	tl.waitForOutcomingLoopStop()

	tl.transport = nil

	for _, f := range tl.filters {
		_, _ = f(&ClosePacket{Reason: "transport cleanup"})
	}

	logger.Info("transport layer cleaned up", zap.String("sessionID", tl.session.ID), zap.Any("transport", tl.transport))
}

func (tl *TransportManager) trySendPacket(packet MediaPacket) {
	tl.mtx.Lock()
	defer tl.mtx.Unlock()

	if tl.txqueue == nil || tl.transport == nil {
		return
	}
	select {
	case tl.txqueue <- packet:
	default:
		logger.Info("packet dropped", zap.String("sessionID", tl.session.ID), zap.Any("packet", packet))
	}
}

type MediaHandler interface {
	GetContext() context.Context
	GetSession() *MediaSession
	CauseError(sender any, err error)
	EmitState(sender any, state string, params ...any)
	EmitPacket(sender any, packet MediaPacket)
	SendToOutput(sender any, packet MediaPacket)
	AddMetric(key string, duration time.Duration)
	InjectPacket(f PacketFilter)
}

type PacketFilter func(packet MediaPacket) (bool, error)
type StateChangeHandler func(event StateChange)
type ErrorHandler func(sender any, err error)
type EncoderFunc func(packet MediaPacket) ([]MediaPacket, error)
type MediaHandlerFunc func(h MediaHandler, data MediaData)
type SessionHook func(session *MediaSession)

// SessionMetrics tracks comprehensive session statistics
type SessionMetrics struct {
	mu sync.RWMutex

	// Packet statistics
	packetCount      uint64 // Total packets processed
	totalBytes       uint64 // Total bytes processed
	audioPacketCount uint64 // Audio packets count
	textPacketCount  uint64 // Text packets count
	closePacketCount uint64 // Close packets count

	// Audio packet details
	audioBytes       uint64 // Total audio bytes
	synthesizedCount uint64 // Synthesized audio packets
	silenceCount     uint64 // Silence packets
	firstPacketCount uint64 // First packet count
	endPacketCount   uint64 // End packet count

	// Text packet details
	transcribedCount  uint64 // Transcribed text packets
	llmGeneratedCount uint64 // LLM generated text packets
	partialTextCount  uint64 // Partial text packets
	totalTextLength   uint64 // Total text length (characters)

	// Error and state statistics
	errorCount          uint64 // Total errors encountered
	stateChangeCount    uint64 // State changes count
	processorErrorCount uint64 // Processor errors count

	// Size statistics
	minPacketSize   uint64 // Minimum packet size
	maxPacketSize   uint64 // Maximum packet size
	totalPacketSize uint64 // Sum of all packet sizes (for average calculation)

	// Timing statistics
	firstPacketTime     time.Time     // Time of first packet
	lastPacketTime      time.Time     // Time of last packet
	totalProcessingTime time.Duration // Total processing time

	// Transport statistics
	inputTransportCount  int // Number of input transports
	outputTransportCount int // Number of output transports
	activeOutputCount    int // Number of active output transports
}

// GetPacketCount returns the total number of packets processed
func (sm *SessionMetrics) GetPacketCount() uint64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.packetCount
}

// GetTotalBytes returns the total bytes processed
func (sm *SessionMetrics) GetTotalBytes() uint64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.totalBytes
}

// GetMetrics returns a copy of current metrics
func (sm *SessionMetrics) GetMetrics() (packetCount, totalBytes uint64) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.packetCount, sm.totalBytes
}

// GetAllMetrics returns all metrics in a structured format
func (sm *SessionMetrics) GetAllMetrics() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	avgPacketSize := uint64(0)
	if sm.packetCount > 0 {
		avgPacketSize = sm.totalPacketSize / sm.packetCount
	}

	return map[string]interface{}{
		"packet_count":           sm.packetCount,
		"total_bytes":            sm.totalBytes,
		"audio_packet_count":     sm.audioPacketCount,
		"text_packet_count":      sm.textPacketCount,
		"close_packet_count":     sm.closePacketCount,
		"audio_bytes":            sm.audioBytes,
		"synthesized_count":      sm.synthesizedCount,
		"silence_count":          sm.silenceCount,
		"first_packet_count":     sm.firstPacketCount,
		"end_packet_count":       sm.endPacketCount,
		"transcribed_count":      sm.transcribedCount,
		"llm_generated_count":    sm.llmGeneratedCount,
		"partial_text_count":     sm.partialTextCount,
		"total_text_length":      sm.totalTextLength,
		"error_count":            sm.errorCount,
		"state_change_count":     sm.stateChangeCount,
		"processor_error_count":  sm.processorErrorCount,
		"min_packet_size":        sm.minPacketSize,
		"max_packet_size":        sm.maxPacketSize,
		"avg_packet_size":        avgPacketSize,
		"first_packet_time":      sm.firstPacketTime,
		"last_packet_time":       sm.lastPacketTime,
		"total_processing_time":  sm.totalProcessingTime.String(),
		"input_transport_count":  sm.inputTransportCount,
		"output_transport_count": sm.outputTransportCount,
		"active_output_count":    sm.activeOutputCount,
	}
}

// GetAudioPacketCount returns the number of audio packets
func (sm *SessionMetrics) GetAudioPacketCount() uint64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.audioPacketCount
}

// GetTextPacketCount returns the number of text packets
func (sm *SessionMetrics) GetTextPacketCount() uint64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.textPacketCount
}

// GetErrorCount returns the total error count
func (sm *SessionMetrics) GetErrorCount() uint64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.errorCount
}

// GetStateChangeCount returns the number of state changes
func (sm *SessionMetrics) GetStateChangeCount() uint64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.stateChangeCount
}

// GetAveragePacketSize returns the average packet size
func (sm *SessionMetrics) GetAveragePacketSize() uint64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.packetCount == 0 {
		return 0
	}
	return sm.totalPacketSize / sm.packetCount
}

// GetMinPacketSize returns the minimum packet size
func (sm *SessionMetrics) GetMinPacketSize() uint64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.minPacketSize
}

// GetMaxPacketSize returns the maximum packet size
func (sm *SessionMetrics) GetMaxPacketSize() uint64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.maxPacketSize
}

// GetSessionDuration returns the session duration based on packet timestamps
func (sm *SessionMetrics) GetSessionDuration() time.Duration {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.firstPacketTime.IsZero() {
		return 0
	}
	if sm.lastPacketTime.IsZero() {
		return 0
	}
	return sm.lastPacketTime.Sub(sm.firstPacketTime)
}

type MediaSession struct {
	ctx          context.Context
	cancel       context.CancelFunc
	encoder      EncoderFunc
	decoder      EncoderFunc
	values       sync.Map
	stateHandles map[string][]StateChangeHandler
	errors       []ErrorHandler
	inputs       []*TransportManager
	outputs      []*TransportManager
	trace        MediaHandlerFunc
	postHoooks   []SessionHook

	// New event-driven architecture
	eventBus          *EventBus
	processorRegistry *ProcessorRegistry
	router            *Router
	inputConnectors   []*TransportConnector
	outputConnectors  []*TransportConnector
	metrics           *SessionMetrics

	ID                 string             `json:"id"`
	Running            bool               `json:"running"`
	QueueSize          int                `json:"queueSize"`
	SampleRate         int                // sample rate of the session
	MaxSessionDuration int                `json:"maxSessionDuration"` // Set the maximum session duration in seconds.
	EffectAudios       map[string]*[]byte `json:"-"`
	StartAt            time.Time          `json:"startAt"`
}

func NewDefaultSession() *MediaSession {
	ctx, cancel := context.WithCancel(context.Background())
	session := &MediaSession{
		ID:           "session-" + time.Now().Format("20060102150405"),
		ctx:          ctx,
		cancel:       cancel,
		values:       sync.Map{},
		stateHandles: make(map[string][]StateChangeHandler),
		SampleRate:   16000,

		Running:            false,
		QueueSize:          128,
		MaxSessionDuration: 10 * 60,
		metrics:            &SessionMetrics{},
	}

	// Initialize new architecture components
	session.eventBus = NewEventBus(ctx, session.QueueSize, 4)
	session.processorRegistry = NewProcessorRegistry()
	session.router = NewRouter(StrategyBroadcast)

	// Subscribe to events
	session.setupEventHandlers()

	return session
}

// setupEventHandlers configures default event handlers
func (s *MediaSession) setupEventHandlers() {
	// Handle packet events through processor registry
	s.eventBus.Subscribe(EventTypePacket, func(ctx context.Context, event *MediaEvent) error {
		startTime := time.Now()
		processors := s.processorRegistry.GetProcessors(ctx, event)
		for _, processor := range processors {
			if err := processor.Process(ctx, s, event); err != nil {
				// Track processor errors
				s.metrics.mu.Lock()
				s.metrics.processorErrorCount++
				s.metrics.mu.Unlock()

				logger.Error("processor error",
					zap.String("processor", processor.Name()),
					zap.String("sessionID", s.ID),
					zap.Error(err))
				s.CauseError(processor, err)
			}
		}
		// Track processing time
		processingTime := time.Since(startTime)
		s.metrics.mu.Lock()
		s.metrics.totalProcessingTime += processingTime
		s.metrics.mu.Unlock()
		return nil
	})

	// Handle state events
	s.eventBus.Subscribe(EventTypeState, func(ctx context.Context, event *MediaEvent) error {
		if state, ok := event.Payload.(StateChange); ok {
			// Process state-specific handlers
			if handlers, found := s.stateHandles[state.State]; found {
				for _, handler := range handlers {
					callHandleWithState(s, handler, state)
				}
			}
			// Process wildcard handlers
			if handlers, found := s.stateHandles[AllStates]; found {
				for _, handler := range handlers {
					callHandleWithState(s, handler, state)
				}
			}
		}
		return nil
	})

	// Handle error events
	s.eventBus.Subscribe(EventTypeError, func(ctx context.Context, event *MediaEvent) error {
		if err, ok := event.Payload.(error); ok {
			sender := event.Metadata["sender"]
			for _, handler := range s.errors {
				handler(sender, err)
			}
		}
		return nil
	})
}
func (s *MediaSession) SetSessionID(id string) *MediaSession {
	s.ID = id
	return s
}

func (s *MediaSession) String() string {
	return fmt.Sprintf("MediaSession{ID: %s, Running: %t, SampleRate: %d}", s.ID, s.Running, s.SampleRate)
}

func (s *MediaSession) Get(key string) (val any, ok bool) {
	return s.values.Load(key)
}

func (s *MediaSession) GetString(key string) string {
	value, ok := s.values.Load(key)
	if !ok {
		return ""
	}
	val, ok := value.(string)
	if ok {
		return val
	}
	return ""
}

func (s *MediaSession) GetUint(key string) uint {
	value, ok := s.values.Load(key)
	if !ok {
		return 0
	}

	switch val := value.(type) {
	case uint:
		return val
	case int:
		if val >= 0 {
			return uint(val)
		}
		return 0
	default:
		return 0
	}
}

func (s *MediaSession) Set(key string, val any) {
	s.values.Store(key, val)
}

func (s *MediaSession) Delete(key string) {
	s.values.Delete(key)
}

func (s *MediaSession) GetContext() context.Context {
	if s.ctx == nil {
		s.Context(context.Background())
	}
	return s.ctx
}

func (s *MediaSession) GetSession() *MediaSession {
	return s
}

// Do nothing
func (s *MediaSession) InjectPacket(f PacketFilter) {
}

// chainable methods
func (s *MediaSession) Context(parent context.Context) *MediaSession {
	s.ctx, s.cancel = context.WithCancel(parent)
	return s
}

func (s *MediaSession) Trace(trace MediaHandlerFunc) *MediaSession {
	s.trace = trace
	return s
}

func (s *MediaSession) Encode(enc EncoderFunc) *MediaSession {
	s.encoder = enc
	return s
}

func (s *MediaSession) Decode(dec EncoderFunc) *MediaSession {
	s.decoder = dec
	return s
}

// AddInputTransport registers input transport with different method name
func (s *MediaSession) AddInputTransport(rx MediaTransport, filterFuncs ...PacketFilter) *MediaSession {
	tl := &TransportManager{
		session:   s,
		transport: rx,
		filters:   filterFuncs,
	}
	rx.Attach(s)
	s.inputs = append(s.inputs, tl)

	// Also add to connectors for router
	connectorID := fmt.Sprintf("input-%d", len(s.inputConnectors))
	connector := NewTransportConnector(connectorID, rx, DirectionInput)
	s.inputConnectors = append(s.inputConnectors, connector)

	// Update metrics
	if s.metrics != nil {
		s.metrics.mu.Lock()
		s.metrics.inputTransportCount = len(s.inputs)
		s.metrics.mu.Unlock()
	}

	return s
}

// Input is an alias for backward compatibility
func (s *MediaSession) Input(rx MediaTransport, filterFuncs ...PacketFilter) *MediaSession {
	return s.AddInputTransport(rx, filterFuncs...)
}

// AddOutputTransport registers output transport with different method name
func (s *MediaSession) AddOutputTransport(tx MediaTransport, filterFuncs ...PacketFilter) *MediaSession {
	queueSize := s.QueueSize
	if queueSize == 0 {
		queueSize = 128
	}
	tl := &TransportManager{
		txqueue:   make(chan MediaPacket, queueSize),
		session:   s,
		transport: tx,
		filters:   filterFuncs,
	}
	logger.Info("output transport registered", zap.String("sessionID", s.ID), zap.Any("transport", tx), zap.Int("queueSize", queueSize))
	tx.Attach(s)
	s.outputs = append(s.outputs, tl)

	// Also add to connectors for router
	connectorID := fmt.Sprintf("output-%d", len(s.outputConnectors))
	connector := NewTransportConnector(connectorID, tx, DirectionOutput)
	s.outputConnectors = append(s.outputConnectors, connector)

	// Update metrics
	if s.metrics != nil {
		s.metrics.mu.Lock()
		s.metrics.outputTransportCount = len(s.outputs)
		// Count active outputs
		activeCount := 0
		for _, connector := range s.outputConnectors {
			if connector.IsActive() {
				activeCount++
			}
		}
		s.metrics.activeOutputCount = activeCount
		s.metrics.mu.Unlock()
	}

	return s
}

// Output is an alias for backward compatibility
func (s *MediaSession) Output(tx MediaTransport, filterFuncs ...PacketFilter) *MediaSession {
	return s.AddOutputTransport(tx, filterFuncs...)
}

func (s *MediaSession) PostHook(hooks ...SessionHook) *MediaSession {
	s.postHoooks = append(s.postHoooks, hooks...)
	return s
}

// Handle error caused
func (s *MediaSession) Error(handles ...ErrorHandler) *MediaSession {
	s.errors = append(s.errors, handles...)
	return s
}

func (s *MediaSession) On(state string, handles ...StateChangeHandler) *MediaSession {
	s.stateHandles[state] = append(s.stateHandles[state], handles...)
	return s
}

// setupOutputRouter configures output routing processor
func (s *MediaSession) setupOutputRouter() {
	// Register packet validation processor (highest priority, runs first)
	validationProcessor := NewHighPriorityPacketProcessor(
		"packet-validator",
		func(ctx context.Context, session *MediaSession, packet MediaPacket) error {
			// Validate packet is not nil
			if packet == nil {
				return fmt.Errorf("received nil packet")
			}

			// Validate audio packet payload
			if audioPacket, ok := packet.(*AudioPacket); ok {
				if audioPacket.Payload == nil {
					return fmt.Errorf("audio packet has nil payload")
				}
				// Validate payload size (prevent extremely large packets)
				if len(audioPacket.Payload) > 64*1024 { // 64KB max
					return fmt.Errorf("packet payload too large: %d bytes", len(audioPacket.Payload))
				}
			}

			// Validate text packet
			if textPacket, ok := packet.(*TextPacket); ok {
				if textPacket.Text == "" && !textPacket.IsEnd {
					return fmt.Errorf("text packet has empty text and is not end packet")
				}
			}

			return nil
		},
	)
	s.processorRegistry.Register(validationProcessor)

	// Register packet monitoring processor (high priority, runs early)
	monitoringProcessor := NewHighPriorityPacketProcessor(
		"packet-monitor",
		func(ctx context.Context, session *MediaSession, packet MediaPacket) error {
			now := time.Now()
			session.metrics.mu.Lock()
			defer session.metrics.mu.Unlock()

			// Update packet count
			session.metrics.packetCount++

			// Track first and last packet times
			if session.metrics.firstPacketTime.IsZero() {
				session.metrics.firstPacketTime = now
			}
			session.metrics.lastPacketTime = now

			// Handle different packet types
			switch p := packet.(type) {
			case *AudioPacket:
				session.metrics.audioPacketCount++
				packetSize := uint64(len(p.Payload))
				session.metrics.audioBytes += packetSize
				session.metrics.totalBytes += packetSize
				session.metrics.totalPacketSize += packetSize

				// Update size statistics
				if session.metrics.minPacketSize == 0 || packetSize < session.metrics.minPacketSize {
					session.metrics.minPacketSize = packetSize
				}
				if packetSize > session.metrics.maxPacketSize {
					session.metrics.maxPacketSize = packetSize
				}

				// Track audio packet flags
				if p.IsSynthesized {
					session.metrics.synthesizedCount++
				}
				if p.IsSilence {
					session.metrics.silenceCount++
				}
				if p.IsFirstPacket {
					session.metrics.firstPacketCount++
				}
				if p.IsEndPacket {
					session.metrics.endPacketCount++
				}

			case *TextPacket:
				session.metrics.textPacketCount++
				textLength := uint64(len(p.Text))
				session.metrics.totalTextLength += textLength
				session.metrics.totalBytes += textLength
				session.metrics.totalPacketSize += textLength

				// Update size statistics
				if session.metrics.minPacketSize == 0 || textLength < session.metrics.minPacketSize {
					session.metrics.minPacketSize = textLength
				}
				if textLength > session.metrics.maxPacketSize {
					session.metrics.maxPacketSize = textLength
				}

				// Track text packet flags
				if p.IsTranscribed {
					session.metrics.transcribedCount++
				}
				if p.IsLLMGenerated {
					session.metrics.llmGeneratedCount++
				}
				if p.IsPartial {
					session.metrics.partialTextCount++
				}

			case *ClosePacket:
				session.metrics.closePacketCount++
			}

			return nil
		},
	)
	s.processorRegistry.Register(monitoringProcessor)

	// Register state change monitoring processor
	stateMonitorProcessor := NewHighPriorityProcessor(
		"state-monitor",
		func(ctx context.Context, session *MediaSession, event *MediaEvent) error {
			if event.Type == EventTypeState {
				session.metrics.mu.Lock()
				session.metrics.stateChangeCount++
				session.metrics.mu.Unlock()
			}
			return nil
		},
	)
	s.processorRegistry.Register(stateMonitorProcessor)

	// Register error monitoring processor
	errorMonitorProcessor := NewHighPriorityProcessor(
		"error-monitor",
		func(ctx context.Context, session *MediaSession, event *MediaEvent) error {
			if event.Type == EventTypeError {
				session.metrics.mu.Lock()
				session.metrics.errorCount++
				session.metrics.mu.Unlock()
			}
			return nil
		},
	)
	s.processorRegistry.Register(errorMonitorProcessor)

	// Register output router processor (lowest priority, runs last)
	outputProcessor := NewPacketProcessor(
		"output-router",
		PriorityLow,
		func(ctx context.Context, session *MediaSession, packet MediaPacket) error {
			// Get active output connectors
			var activeOutputs []*TransportConnector
			for _, connector := range session.outputConnectors {
				if connector.IsActive() {
					activeOutputs = append(activeOutputs, connector)
				}
			}

			// Update active output count in metrics
			session.metrics.mu.Lock()
			session.metrics.activeOutputCount = len(activeOutputs)
			session.metrics.mu.Unlock()

			// Route packet to outputs
			targets := session.router.Route(packet, activeOutputs)
			for _, target := range targets {
				if target.Transport != nil {
					_, err := target.Transport.Send(ctx, packet)
					if err != nil {
						logger.Error("failed to send packet to transport",
							zap.String("sessionID", session.ID),
							zap.String("transportID", target.ID),
							zap.Error(err))
					}
				}
			}
			return nil
		},
	)
	s.processorRegistry.Register(outputProcessor)
}

// RegisterProcessor registers a processor in the registry
func (s *MediaSession) RegisterProcessor(processor Processor) *MediaSession {
	s.processorRegistry.Register(processor)
	return s
}

// UseMiddleware is deprecated, use RegisterProcessor instead
func (s *MediaSession) UseMiddleware(handles ...MediaHandlerFunc) *MediaSession {
	for i, handle := range handles {
		processor := NewFuncProcessor(
			fmt.Sprintf("middleware-%d", i),
			PriorityNormal,
			func(ctx context.Context, session *MediaSession, event *MediaEvent) error {
				if event.Type == EventTypePacket {
					if packet, ok := event.Payload.(MediaPacket); ok {
						handler := &sessionHandlerAdapter{session: session}
						data := MediaData{
							Type:      MediaDataTypePacket,
							Packet:    packet,
							CreatedAt: event.Timestamp,
							Sender:    event.Metadata["sender"],
						}
						handle(handler, data)
					}
				}
				return nil
			},
		)
		s.processorRegistry.Register(processor)
	}
	return s
}

// Pipeline is an alias for backward compatibility
func (s *MediaSession) Pipeline(handles ...MediaHandlerFunc) *MediaSession {
	return s.UseMiddleware(handles...)
}

// sessionHandlerAdapter adapts MediaSession to MediaHandler interface
type sessionHandlerAdapter struct {
	session *MediaSession
}

func (a *sessionHandlerAdapter) GetContext() context.Context {
	return a.session.GetContext()
}

func (a *sessionHandlerAdapter) GetSession() *MediaSession {
	return a.session
}

func (a *sessionHandlerAdapter) EmitState(sender any, state string, params ...any) {
	a.session.EmitState(sender, state, params...)
}

func (a *sessionHandlerAdapter) EmitPacket(sender any, packet MediaPacket) {
	a.session.EmitPacket(sender, packet)
}

func (a *sessionHandlerAdapter) SendToOutput(sender any, packet MediaPacket) {
	a.session.SendToOutput(sender, packet)
}

func (a *sessionHandlerAdapter) CauseError(sender any, err error) {
	a.session.CauseError(sender, err)
}

func (a *sessionHandlerAdapter) AddMetric(key string, duration time.Duration) {
	a.session.AddMetric(key, duration)
}

func (a *sessionHandlerAdapter) InjectPacket(f PacketFilter) {
	a.session.InjectPacket(f)
}

func (s *MediaSession) IsValid() error {
	if len(s.inputs) == 0 {
		return ErrNotInputTransport
	}
	if len(s.outputs) == 0 {
		return ErrNotOutputTransport
	}
	return nil
}

// Serve Start the session, this will block the current goroutine
func (s *MediaSession) Serve() error {
	s.StartAt = time.Now()
	s.Running = true

	defer func() {
		if err := recover(); err != nil {
			logger.Error("session recover err", zap.Any("error", err), zap.String("stacktrace", string(debug.Stack())))
			return
		}
		s.Running = false
		logger.Info("session stopped", zap.String("sessionID", s.ID))
		s.cleanup()
		s.EmitState(s, End)
		for idx := range s.postHoooks {
			interceptor := s.postHoooks[idx]
			interceptor(s)
		}
	}()

	s.setupOutputRouter()

	if s.MaxSessionDuration > 0 {
		time.AfterFunc(time.Duration(s.MaxSessionDuration)*time.Second, func() {
			logger.Info("session stopped timeout", zap.String("sessionID", s.ID), zap.Int("timeout", s.MaxSessionDuration))
			s.EmitState(s, Hangup, []string{"timeout"})
			_ = s.Close()
		})
	}

	for idx := range s.inputs {
		tl := s.inputs[idx]
		go tl.processIncoming()
	}

	for idx := range s.outputs {
		tl := s.outputs[idx]
		go tl.processOutgoing()

	}
	s.EmitState(s, Begin)
	logger.Info("session started", zap.String("sessionID", s.ID))

	// Main event loop is now handled by event bus workers
	// Just wait for context cancellation
	<-s.ctx.Done()

	return nil
}

func (s *MediaSession) Close() error {
	s.cancel()
	return nil
}

func (s *MediaSession) Codec() CodecConfig {
	return CodecConfig{
		Codec:      "pcm",
		SampleRate: s.SampleRate,
		Channels:   1,
		BitDepth:   16,
	}
}

func (s *MediaSession) cleanup() {
	// Stop event bus
	if s.eventBus != nil {
		s.eventBus.Close()
	}

	for idx := range s.inputs {
		tl := s.inputs[idx]
		tl.cleanup()
	}

	for idx := range s.outputs {
		tl := s.outputs[idx]
		tl.cleanup()
	}
}

func (s *MediaSession) putPacket(direction string, packet MediaPacket) {
	tls := s.inputs
	if direction == DirectionOutput {
		tls = s.outputs
	}

	for idx := range tls {
		tl := tls[idx]
		tl.trySendPacket(packet)
	}
}

func senderAsString(sender any) string {
	if sender == nil {
		return ""
	}
	if s, ok := sender.(string); ok {
		return s
	}
	n := reflect.TypeOf(sender).String()
	if end := strings.LastIndex(n, "."); end != -1 {
		n = n[end+1:]
	}
	return n
}

func (s *MediaSession) CauseError(sender any, err error) {
	sender = senderAsString(sender)
	logger.Error("cause error", zap.String("sessionID", s.ID), zap.Any("sender", sender), zap.Error(err))

	// Publish error event
	if s.eventBus != nil {
		s.eventBus.PublishError(s.ID, err, sender)
	}

	// Also call direct error handlers for backward compatibility
	for _, handle := range s.errors {
		handle(sender, err)
	}
}

func (s *MediaSession) EmitState(sender any, state string, params ...any) {
	sender = senderAsString(sender)
	event := StateChange{
		State:  state,
		Params: params,
	}

	logger.Info("emitstate", zap.Any("sender", sender), zap.String("state", state), zap.Any("params", params), zap.String("sessionID", s.ID))

	if s.eventBus != nil {
		s.eventBus.PublishState(s.ID, event, sender)
	}
}

func (s *MediaSession) EmitPacket(sender any, packet MediaPacket) {
	if s.eventBus != nil {
		s.eventBus.PublishPacket(s.ID, packet, sender)
	}
}

func (s *MediaSession) SendToOutput(sender any, packet MediaPacket) {
	s.putPacket(DirectionOutput, packet)
}

func (s *MediaSession) AddMetric(key string, duration time.Duration) {
	// Metrics功能已移除

	// Metrics功能已移除
	if s.trace != nil {
		data := MediaData{
			CreatedAt: time.Now(),
			Type:      MediaDataTypeMetric,
			Sender:    key,
			Duration:  &duration,
		}
		s.trace(s, data)
	}
}

// GetMetrics returns session metrics (backward compatible)
func (s *MediaSession) GetMetrics() (packetCount, totalBytes uint64) {
	if s.metrics == nil {
		return 0, 0
	}
	return s.metrics.GetMetrics()
}

// GetAllMetrics returns all metrics in a structured format
func (s *MediaSession) GetAllMetrics() map[string]interface{} {
	if s.metrics == nil {
		return make(map[string]interface{})
	}
	return s.metrics.GetAllMetrics()
}

// GetPacketCount returns the total number of packets processed
func (s *MediaSession) GetPacketCount() uint64 {
	if s.metrics == nil {
		return 0
	}
	return s.metrics.GetPacketCount()
}

// GetTotalBytes returns the total bytes processed
func (s *MediaSession) GetTotalBytes() uint64 {
	if s.metrics == nil {
		return 0
	}
	return s.metrics.GetTotalBytes()
}

// GetAudioPacketCount returns the number of audio packets
func (s *MediaSession) GetAudioPacketCount() uint64 {
	if s.metrics == nil {
		return 0
	}
	return s.metrics.GetAudioPacketCount()
}

// GetTextPacketCount returns the number of text packets
func (s *MediaSession) GetTextPacketCount() uint64 {
	if s.metrics == nil {
		return 0
	}
	return s.metrics.GetTextPacketCount()
}

// GetErrorCount returns the total error count
func (s *MediaSession) GetErrorCount() uint64 {
	if s.metrics == nil {
		return 0
	}
	return s.metrics.GetErrorCount()
}

// GetStateChangeCount returns the number of state changes
func (s *MediaSession) GetStateChangeCount() uint64 {
	if s.metrics == nil {
		return 0
	}
	return s.metrics.GetStateChangeCount()
}

// GetAveragePacketSize returns the average packet size
func (s *MediaSession) GetAveragePacketSize() uint64 {
	if s.metrics == nil {
		return 0
	}
	return s.metrics.GetAveragePacketSize()
}

// GetSessionDuration returns the session duration based on packet timestamps
func (s *MediaSession) GetSessionDuration() time.Duration {
	if s.metrics == nil {
		return 0
	}
	return s.metrics.GetSessionDuration()
}

// processData is deprecated, events are now handled by event bus
// This method is kept for backward compatibility but does nothing
func (s *MediaSession) processData(data *MediaData) {
	// Events are now processed through event bus
	// This method is kept for compatibility but should not be used
}

func callHandleWithState(s *MediaSession, handle StateChangeHandler, state StateChange) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("state panic", zap.String("sessionID", s.ID), zap.Any("state", state), zap.Any("error", r), zap.Any("handle", handle), zap.String("stacktrace", string(debug.Stack())))
		}
	}()
	handle(state)
}

func callHandleWithMediaData(s *MediaSession, h MediaHandler, handle MediaHandlerFunc, data MediaData) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("handle panic", zap.String("sessionID", s.ID), zap.Any("data", data), zap.Any("error", r), zap.Any("handle", handle), zap.String("stacktrace", string(debug.Stack())))
		}
	}()
	handle(h, data)
}

func CastOption[T any](options map[string]any) (val T) {
	setEnvOrDefaults(&val)
	if options == nil {
		return
	}
	data, err := json.Marshal(options)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &val)
	if err != nil {
		logger.Error("cast option error", zap.Any("options", options), zap.String("target", reflect.TypeOf(val).Name()), zap.Error(err))
	}
	return
}

func setEnvOrDefaults(opt any) {
	v := reflect.ValueOf(opt).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if !field.IsZero() {
			continue
		}

		if field.Kind() == reflect.Struct && field.CanAddr() {
			setEnvOrDefaults(field.Addr().Interface())
			continue
		}

		fieldVal := os.Getenv(fieldType.Tag.Get("env"))
		if fieldVal == "" {
			fieldVal = fieldType.Tag.Get("default")
		}

		if fieldVal != "" && field.IsZero() {
			switch field.Kind() {
			case reflect.String:
				field.SetString(fieldVal)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				value, err := strconv.ParseInt(fieldVal, 10, 64)
				if err == nil {
					field.SetInt(value)
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				value, err := strconv.ParseUint(fieldVal, 10, 64)
				if err == nil {
					field.SetUint(value)
				}
			case reflect.Float32, reflect.Float64:
				value, err := strconv.ParseFloat(fieldVal, 64)
				if err == nil {
					field.SetFloat(value)
				}
			case reflect.Bool:
				value, _ := strconv.ParseBool(fieldVal)
				field.SetBool(value)
			default:
			}
		}
	}
}
