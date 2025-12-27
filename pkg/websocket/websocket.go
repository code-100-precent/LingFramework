package websocket

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Message defines WebSocket message structure
type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
	From      string      `json:"from,omitempty"`
	To        string      `json:"to,omitempty"`
	Group     string      `json:"group,omitempty"`
}

// Connection represents a WebSocket connection
type Connection struct {
	ID       string
	UserID   string
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
	LastPing time.Time
	IsAlive  bool
	Status   string // Connection status (ConnectionStatusConnected, ConnectionStatusDisconnected, etc.)
	mu       sync.RWMutex
	Groups   map[string]bool
	Metadata map[string]interface{}
}

// Hub manages all WebSocket connections
type Hub struct {
	// Registered connections
	connections map[string]*Connection
	// User ID to connection ID mapping
	userConnections map[string]map[string]bool
	// Group to connection ID mapping
	groupConnections map[string]map[string]bool
	// Broadcast message channel
	broadcast chan *Message
	// Register connection channel
	register chan *Connection
	// Unregister connection channel
	unregister chan *Connection
	// Connection count
	connectionCount int64
	// Configuration
	config *Config
	// Mutex
	mu sync.RWMutex
	// Context
	ctx    context.Context
	cancel context.CancelFunc

	// Shards and locks to reduce contention when fanout
	shardCount int
	shardConns []map[string]*Connection
	shardLocks []sync.RWMutex

	// Broadcast worker pool
	broadcastJobs chan broadcastJob

	// Global ping
	pingJobs chan int
}

const (
	_broadcastAll = iota
)

type broadcastJob struct {
	kind  int
	shard int
	data  []byte
}

// Config is WebSocket configuration
type Config struct {
	// Maximum connections
	MaxConnections int64
	// Heartbeat interval
	HeartbeatInterval time.Duration
	// Connection timeout
	ConnectionTimeout time.Duration
	// Message buffer size
	MessageBufferSize int
	// Read buffer size
	ReadBufferSize int
	// Write buffer size
	WriteBufferSize int
	// Maximum message size
	MaxMessageSize int
	// Whether to enable compression
	EnableCompression bool
	// Whether to enable message queue
	EnableMessageQueue bool
	// Message queue size
	MessageQueueSize int
	// Whether to enable cluster mode
	EnableCluster bool
	// Cluster node ID
	ClusterNodeID string
	// Shard count
	ShardCount int
	// Broadcast worker count
	BroadcastWorkerCount int
	// Whether to drop when send buffer is full
	DropOnFull bool
	// Compression level (-2..9)
	CompressionLevel int
	// Slow consumer strategy: disconnect when backpressure is triggered
	CloseOnBackpressure bool
	// Send blocking timeout (for non-DropOnFull mode)
	SendTimeout time.Duration
	// Enable global ping
	EnableGlobalPing bool
	// Global ping workers
	PingWorkerCount int
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxConnections:       100000, // 100k connections
		HeartbeatInterval:    30 * time.Second,
		ConnectionTimeout:    60 * time.Second,
		MessageBufferSize:    256,
		ReadBufferSize:       1024,
		WriteBufferSize:      1024,
		MaxMessageSize:       512,
		EnableCompression:    true,
		EnableMessageQueue:   true,
		MessageQueueSize:     1000,
		EnableCluster:        false,
		ClusterNodeID:        "",
		ShardCount:           16,
		BroadcastWorkerCount: 32,
		DropOnFull:           true,
		CompressionLevel:     -2,
		CloseOnBackpressure:  false,
		SendTimeout:          50 * time.Millisecond,
		EnableGlobalPing:     false,
		PingWorkerCount:      8,
	}
}

// NewHub creates a new Hub instance
func NewHub(config *Config) *Hub {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	hub := &Hub{
		connections:      make(map[string]*Connection),
		userConnections:  make(map[string]map[string]bool),
		groupConnections: make(map[string]map[string]bool),
		broadcast:        make(chan *Message, config.MessageQueueSize),
		register:         make(chan *Connection, 1000),
		unregister:       make(chan *Connection, 1000),
		config:           config,
		ctx:              ctx,
		cancel:           cancel,
	}

	// Initialize shards
	if hub.config.ShardCount <= 0 {
		hub.config.ShardCount = 1
	}
	hub.shardCount = hub.config.ShardCount
	hub.shardConns = make([]map[string]*Connection, hub.shardCount)
	hub.shardLocks = make([]sync.RWMutex, hub.shardCount)
	for i := 0; i < hub.shardCount; i++ {
		hub.shardConns[i] = make(map[string]*Connection)
	}

	// Initialize broadcast workers
	if hub.config.BroadcastWorkerCount <= 0 {
		hub.config.BroadcastWorkerCount = 1
	}
	hub.broadcastJobs = make(chan broadcastJob, hub.config.MessageQueueSize)
	for i := 0; i < hub.config.BroadcastWorkerCount; i++ {
		go hub.broadcastWorker()
	}

	// Initialize global ping workers
	if hub.config.EnableGlobalPing {
		if hub.config.PingWorkerCount <= 0 {
			hub.config.PingWorkerCount = 1
		}
		hub.pingJobs = make(chan int, hub.shardCount)
		for i := 0; i < hub.config.PingWorkerCount; i++ {
			go hub.pingWorker()
		}
	}

	go hub.run()
	return hub
}

// run is the main Hub loop
func (h *Hub) run() {
	ticker := time.NewTicker(h.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case conn := <-h.register:
			h.registerConnection(conn)
		case conn := <-h.unregister:
			h.unregisterConnection(conn)
		case message := <-h.broadcast:
			// Serialize once to reduce duplicate overhead
			if message.Timestamp == 0 {
				message.Timestamp = time.Now().Unix()
			}
			data, err := json.Marshal(message)
			if err != nil {
				logrus.Errorf("message serialization failed: %v", err)
				continue
			}
			switch {
			case message.To != "":
				h.sendToUser(message.To, data)
			case message.Group != "":
				h.sendToGroup(message.Group, data)
			default:
				h.enqueueBroadcastAll(data)
			}
		case <-ticker.C:
			if h.config.EnableGlobalPing {
				// Trigger ping at shard level
				for i := 0; i < h.shardCount; i++ {
					select {
					case h.pingJobs <- i:
					default:
					}
				}
			}
			h.checkHeartbeats()
		}
	}
}

// pingWorker is the global ping worker
func (h *Hub) pingWorker() {
	for shard := range h.pingJobs {
		h.shardLocks[shard].RLock()
		for _, conn := range h.shardConns[shard] {
			if conn.IsAlive {
				_ = conn.Conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second))
			}
		}
		h.shardLocks[shard].RUnlock()
	}
}

// registerConnection registers a connection
func (h *Hub) registerConnection(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check maximum connections
	if atomic.LoadInt64(&h.connectionCount) >= h.config.MaxConnections {
		conn.Conn.Close()
		logrus.Warnf(ErrConnectionLimitExceeded+": %d", h.config.MaxConnections)
		return
	}

	h.connections[conn.ID] = conn
	atomic.AddInt64(&h.connectionCount, 1)

	// Put into shard
	sh := h.shardIndex(conn.ID)
	h.shardLocks[sh].Lock()
	h.shardConns[sh][conn.ID] = conn
	h.shardLocks[sh].Unlock()

	// Add to user connection mapping
	if conn.UserID != "" {
		if h.userConnections[conn.UserID] == nil {
			h.userConnections[conn.UserID] = make(map[string]bool)
		}
		h.userConnections[conn.UserID][conn.ID] = true
	}

	// Add to group connection mapping
	for group := range conn.Groups {
		if h.groupConnections[group] == nil {
			h.groupConnections[group] = make(map[string]bool)
		}
		h.groupConnections[group][conn.ID] = true
	}

	logrus.Infof("WebSocket connection registered: %s, user: %s, current connections: %d",
		conn.ID, conn.UserID, atomic.LoadInt64(&h.connectionCount))
}

// unregisterConnection unregisters a connection
func (h *Hub) unregisterConnection(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.connections[conn.ID]; exists {
		delete(h.connections, conn.ID)
		atomic.AddInt64(&h.connectionCount, -1)

		// Remove from shard
		sh := h.shardIndex(conn.ID)
		h.shardLocks[sh].Lock()
		delete(h.shardConns[sh], conn.ID)
		h.shardLocks[sh].Unlock()

		// Remove from user connection mapping
		if conn.UserID != "" && h.userConnections[conn.UserID] != nil {
			delete(h.userConnections[conn.UserID], conn.ID)
			if len(h.userConnections[conn.UserID]) == 0 {
				delete(h.userConnections, conn.UserID)
			}
		}

		// Remove from group connection mapping
		for group := range conn.Groups {
			if h.groupConnections[group] != nil {
				delete(h.groupConnections[group], conn.ID)
				if len(h.groupConnections[group]) == 0 {
					delete(h.groupConnections, group)
				}
			}
		}

		close(conn.Send)
		logrus.Infof("WebSocket connection unregistered: %s, current connections: %d",
			conn.ID, atomic.LoadInt64(&h.connectionCount))
	}
}

// broadcastMessage broadcasts a message
func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Set timestamp
	if message.Timestamp == 0 {
		message.Timestamp = time.Now().Unix()
	}

	// Serialize message
	data, err := json.Marshal(message)
	if err != nil {
		logrus.Errorf("message serialization failed: %v", err)
		return
	}

	// Decide sending strategy based on message type
	switch {
	case message.To != "":
		// Send to specific user
		h.sendToUser(message.To, data)
	case message.Group != "":
		// Send to specific group
		h.sendToGroup(message.Group, data)
	default:
		// Broadcast to all connections
		h.sendToAll(data)
	}
}

// sendToUser sends a message to a specific user
func (h *Hub) sendToUser(userID string, data []byte) {
	if connections, exists := h.userConnections[userID]; exists {
		for connID := range connections {
			if conn, ok := h.connections[connID]; ok && conn.IsAlive {
				h.trySend(conn, data, func() { logrus.Warnf("user %s connection %s send buffer full", userID, connID) })
			}
		}
	}
}

// sendToGroup sends a message to a specific group
func (h *Hub) sendToGroup(group string, data []byte) {
	if connections, exists := h.groupConnections[group]; exists {
		for connID := range connections {
			if conn, ok := h.connections[connID]; ok && conn.IsAlive {
				h.trySend(conn, data, func() { logrus.Warnf("group %s connection %s send buffer full", group, connID) })
			}
		}
	}
}

// sendToAll sends a message to all connections
func (h *Hub) sendToAll(data []byte) {
	for i := 0; i < h.shardCount; i++ {
		select {
		case h.broadcastJobs <- broadcastJob{kind: _broadcastAll, shard: i, data: data}:
		default:
			logrus.Warnf("broadcast job queue full, message dropped")
		}
	}
}

// checkHeartbeats checks heartbeats
func (h *Hub) checkHeartbeats() {
	h.mu.RLock()
	defer h.mu.RUnlock()

	now := time.Now()
	for _, conn := range h.connections {
		if now.Sub(conn.LastPing) > h.config.ConnectionTimeout {
			logrus.Warnf("connection %s heartbeat timeout, closing", conn.ID)
			conn.mu.Lock()
			conn.IsAlive = false
			conn.Status = ConnectionStatusError
			conn.mu.Unlock()
			conn.Conn.Close()
		}
	}
}

// GetConnectionCount gets current connection count
func (h *Hub) GetConnectionCount() int64 {
	return atomic.LoadInt64(&h.connectionCount)
}

// GetUserConnections gets connection count for a user
func (h *Hub) GetUserConnections(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if connections, exists := h.userConnections[userID]; exists {
		return len(connections)
	}
	return 0
}

// GetGroupConnections gets connection count for a group
func (h *Hub) GetGroupConnections(group string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if connections, exists := h.groupConnections[group]; exists {
		return len(connections)
	}
	return 0
}

// GetBroadcastChannel gets the broadcast channel (for external message sending)
func (h *Hub) GetBroadcastChannel() chan<- *Message {
	return h.broadcast
}

// BroadcastToGroup broadcasts a message to a specific group
func (h *Hub) BroadcastToGroup(group string, message *Message) error {
	if message.Timestamp == 0 {
		message.Timestamp = time.Now().Unix()
	}
	message.Group = group
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	h.sendToGroup(group, data)
	return nil
}

// SendToUser sends a message to a specific user
func (h *Hub) SendToUser(userID string, message *Message) error {
	if message.Timestamp == 0 {
		message.Timestamp = time.Now().Unix()
	}
	message.To = userID
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	h.sendToUser(userID, data)
	return nil
}

// BroadcastToAll broadcasts a message to all connections
func (h *Hub) BroadcastToAll(message *Message) error {
	if message.Timestamp == 0 {
		message.Timestamp = time.Now().Unix()
	}
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	h.enqueueBroadcastAll(data)
	return nil
}

// GetConnection gets a connection by ID
func (h *Hub) GetConnection(connID string) *Connection {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.connections[connID]
}

// IsConnectionAlive checks if a connection is alive
func (h *Hub) IsConnectionAlive(connID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if conn, ok := h.connections[connID]; ok {
		return conn.IsAlive
	}
	return false
}

// Close closes the Hub
func (h *Hub) Close() {
	h.cancel()

	// Close all connections
	h.mu.Lock()
	for _, conn := range h.connections {
		conn.Conn.Close()
	}
	h.mu.Unlock()

	logrus.Info("WebSocket Hub closed")
}

// shardIndex calculates shard index
func (h *Hub) shardIndex(id string) int {
	if h.shardCount <= 1 {
		return 0
	}
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(id))
	return int(hasher.Sum32() % uint32(h.shardCount))
}

// enqueueBroadcastAll enqueues broadcast tasks by shard
func (h *Hub) enqueueBroadcastAll(data []byte) {
	for i := 0; i < h.shardCount; i++ {
		select {
		case h.broadcastJobs <- broadcastJob{kind: _broadcastAll, shard: i, data: data}:
		default:
			logrus.Warnf("broadcast job queue full, message dropped")
		}
	}
}

// broadcastWorker is the broadcast worker
func (h *Hub) broadcastWorker() {
	for job := range h.broadcastJobs {
		switch job.kind {
		case _broadcastAll:
			h.shardLocks[job.shard].RLock()
			for _, conn := range h.shardConns[job.shard] {
				if conn.IsAlive {
					h.trySend(conn, job.data, func() { logrus.Debugf("connection %s send buffer full, handled by strategy", conn.ID) })
				}
			}
			h.shardLocks[job.shard].RUnlock()
		}
	}
}

// trySend implements backpressure strategy
func (h *Hub) trySend(conn *Connection, data []byte, onDrop func()) {
	if h.config.DropOnFull {
		select {
		case conn.Send <- data:
		default:
			onDrop()
			if h.config.CloseOnBackpressure {
				conn.Conn.Close()
			}
		}
		return
	}
	// Non-drop mode: limit wait duration
	timeout := h.config.SendTimeout
	if timeout <= 0 {
		timeout = 50 * time.Millisecond
	}
	select {
	case conn.Send <- data:
	case <-time.After(timeout):
		onDrop()
		if h.config.CloseOnBackpressure {
			conn.Conn.Close()
		}
	}
}
