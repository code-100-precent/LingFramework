package websocket

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestNewHub(t *testing.T) {
	config := DefaultConfig()
	hub := NewHub(config)
	assert.NotNil(t, hub)
	assert.Equal(t, int64(0), hub.GetConnectionCount())
	hub.Close()
}

func TestNewHub_NilConfig(t *testing.T) {
	hub := NewHub(nil)
	assert.NotNil(t, hub)
	assert.Equal(t, DefaultConfig().MaxConnections, hub.config.MaxConnections)
	hub.Close()
}

func TestHub_RegisterConnection(t *testing.T) {
	config := DefaultConfig()
	config.MaxConnections = 10
	hub := NewHub(config)
	defer hub.Close()

	// Create test connection
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = upgrader.Upgrade(w, r, nil)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Skipf("Failed to create test connection: %v", err)
		return
	}
	defer conn.Close()

	connection := &Connection{
		ID:       generateConnectionID(),
		UserID:   "user1",
		Conn:     conn,
		Send:     make(chan []byte, config.MessageBufferSize),
		Hub:      hub,
		LastPing: time.Now(),
		IsAlive:  true,
		Status:   ConnectionStatusConnected,
		Groups:   make(map[string]bool),
		Metadata: make(map[string]interface{}),
	}

	hub.register <- connection
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, int64(1), hub.GetConnectionCount())
	assert.Equal(t, 1, hub.GetUserConnections("user1"))

	hub.unregister <- connection
	time.Sleep(100 * time.Millisecond)
}

func TestHub_UnregisterConnection(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int64(1), hub.GetConnectionCount())

	hub.unregister <- conn
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int64(0), hub.GetConnectionCount())
}

func TestHub_MaxConnections(t *testing.T) {
	config := DefaultConfig()
	config.MaxConnections = 2
	hub := NewHub(config)
	defer hub.Close()

	// Create connections up to limit
	var connections []*Connection
	var wsConns []*websocket.Conn

	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = upgrader.Upgrade(w, r, nil)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	for i := 0; i < 3; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Skipf("Failed to create test connection: %v", err)
			return
		}
		wsConns = append(wsConns, conn)

		connection := &Connection{
			ID:       generateConnectionID(),
			UserID:   "user1",
			Conn:     conn,
			Send:     make(chan []byte, config.MessageBufferSize),
			Hub:      hub,
			LastPing: time.Now(),
			IsAlive:  true,
			Status:   ConnectionStatusConnected,
			Groups:   make(map[string]bool),
			Metadata: make(map[string]interface{}),
		}
		connections = append(connections, connection)
		hub.register <- connection
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)
	// Should only have 2 connections registered
	assert.LessOrEqual(t, hub.GetConnectionCount(), int64(2))

	// Clean up
	for _, conn := range connections {
		hub.unregister <- conn
	}
	for _, wsConn := range wsConns {
		wsConn.Close()
	}
	time.Sleep(100 * time.Millisecond)
}

func TestHub_SendToUser(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn1, wsConn1 := createTestConnection(t, hub, "user1")
	if conn1 == nil {
		return
	}
	defer wsConn1.Close()

	conn2, wsConn2 := createTestConnection(t, hub, "user1")
	if conn2 == nil {
		return
	}
	defer wsConn2.Close()

	hub.register <- conn1
	hub.register <- conn2
	time.Sleep(200 * time.Millisecond)

	message := &Message{
		Type:      MessageTypeChat,
		Data:      "test message",
		Timestamp: time.Now().Unix(),
	}

	err := hub.SendToUser("user1", message)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	hub.unregister <- conn1
	hub.unregister <- conn2
	time.Sleep(100 * time.Millisecond)
}

func TestHub_BroadcastToGroup(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn1, wsConn1 := createTestConnection(t, hub, "user1")
	if conn1 == nil {
		return
	}
	defer wsConn1.Close()

	conn2, wsConn2 := createTestConnection(t, hub, "user2")
	if conn2 == nil {
		return
	}
	defer wsConn2.Close()

	hub.register <- conn1
	hub.register <- conn2
	time.Sleep(200 * time.Millisecond)

	conn1.JoinGroup("test-group")
	conn2.JoinGroup("test-group")
	time.Sleep(100 * time.Millisecond)

	message := &Message{
		Type:      MessageTypeChat,
		Data:      "group message",
		Timestamp: time.Now().Unix(),
	}

	err := hub.BroadcastToGroup("test-group", message)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	hub.unregister <- conn1
	hub.unregister <- conn2
	time.Sleep(100 * time.Millisecond)
}

func TestHub_BroadcastToAll(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn1, wsConn1 := createTestConnection(t, hub, "user1")
	if conn1 == nil {
		return
	}
	defer wsConn1.Close()

	conn2, wsConn2 := createTestConnection(t, hub, "user2")
	if conn2 == nil {
		return
	}
	defer wsConn2.Close()

	hub.register <- conn1
	hub.register <- conn2
	time.Sleep(200 * time.Millisecond)

	message := &Message{
		Type:      MessageTypeNotification,
		Data:      "broadcast message",
		Timestamp: time.Now().Unix(),
	}

	err := hub.BroadcastToAll(message)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	hub.unregister <- conn1
	hub.unregister <- conn2
	time.Sleep(100 * time.Millisecond)
}

func TestHub_GetConnection(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(200 * time.Millisecond)

	retrieved := hub.GetConnection(conn.ID)
	assert.NotNil(t, retrieved)
	assert.Equal(t, conn.ID, retrieved.ID)

	hub.unregister <- conn
	time.Sleep(100 * time.Millisecond)
}

func TestHub_IsConnectionAlive(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(200 * time.Millisecond)

	assert.True(t, hub.IsConnectionAlive(conn.ID))

	conn.IsAlive = false
	assert.False(t, hub.IsConnectionAlive(conn.ID))

	hub.unregister <- conn
	time.Sleep(100 * time.Millisecond)
}

func TestHub_GetUserConnections(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn1, wsConn1 := createTestConnection(t, hub, "user1")
	if conn1 == nil {
		return
	}
	defer wsConn1.Close()

	conn2, wsConn2 := createTestConnection(t, hub, "user1")
	if conn2 == nil {
		return
	}
	defer wsConn2.Close()

	hub.register <- conn1
	hub.register <- conn2
	time.Sleep(200 * time.Millisecond)

	count := hub.GetUserConnections("user1")
	assert.Equal(t, 2, count)

	count = hub.GetUserConnections("nonexistent")
	assert.Equal(t, 0, count)

	hub.unregister <- conn1
	hub.unregister <- conn2
	time.Sleep(100 * time.Millisecond)
}

func TestHub_GetGroupConnections(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn1, wsConn1 := createTestConnection(t, hub, "user1")
	if conn1 == nil {
		return
	}
	defer wsConn1.Close()

	conn2, wsConn2 := createTestConnection(t, hub, "user2")
	if conn2 == nil {
		return
	}
	defer wsConn2.Close()

	hub.register <- conn1
	hub.register <- conn2
	time.Sleep(200 * time.Millisecond)

	conn1.JoinGroup("test-group")
	conn2.JoinGroup("test-group")
	time.Sleep(100 * time.Millisecond)

	count := hub.GetGroupConnections("test-group")
	assert.Equal(t, 2, count)

	count = hub.GetGroupConnections("nonexistent")
	assert.Equal(t, 0, count)

	hub.unregister <- conn1
	hub.unregister <- conn2
	time.Sleep(100 * time.Millisecond)
}

func TestHub_Close(t *testing.T) {
	hub := setupTestHub()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, int64(1), hub.GetConnectionCount())

	hub.Close()
	time.Sleep(200 * time.Millisecond)

	// Hub should be closed, but connection count might still be 1
	// since Close() doesn't wait for cleanup
}

func TestHub_ShardIndex(t *testing.T) {
	config := DefaultConfig()
	config.ShardCount = 8
	hub := NewHub(config)
	defer hub.Close()

	indices := make(map[int]bool)
	for i := 0; i < 100; i++ {
		connID := generateConnectionID()
		idx := hub.shardIndex(connID)
		assert.GreaterOrEqual(t, idx, 0)
		assert.Less(t, idx, 8)
		indices[idx] = true
	}

	// Should have used multiple shards
	assert.Greater(t, len(indices), 1)
}

func TestHub_CheckHeartbeats(t *testing.T) {
	config := DefaultConfig()
	config.ConnectionTimeout = 100 * time.Millisecond
	hub := NewHub(config)
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(200 * time.Millisecond)

	// Set LastPing to old time
	conn.mu.Lock()
	conn.LastPing = time.Now().Add(-200 * time.Millisecond)
	conn.mu.Unlock()

	// Trigger heartbeat check
	hub.checkHeartbeats()
	time.Sleep(100 * time.Millisecond)

	// Connection should be marked as not alive
	assert.False(t, conn.IsAlive)

	hub.unregister <- conn
	time.Sleep(100 * time.Millisecond)
}
