package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func setupTestHub() *Hub {
	config := DefaultConfig()
	config.MaxConnections = 1000
	return NewHub(config)
}

// createTestConnection creates a test WebSocket connection
func createTestConnection(t *testing.T, hub *Hub, userID string) (*Connection, *websocket.Conn) {
	t.Helper()

	// Create test server
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = upgrader.Upgrade(w, r, nil)
	}))
	defer server.Close()

	// Create WebSocket client connection
	wsURL := "ws" + server.URL[4:]
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Skipf("Failed to create test connection: %v", err)
		return nil, nil
	}

	// Create connection instance
	connection := &Connection{
		ID:       generateConnectionID(),
		UserID:   userID,
		Conn:     wsConn,
		Send:     make(chan []byte, hub.config.MessageBufferSize),
		Hub:      hub,
		LastPing: time.Now(),
		IsAlive:  true,
		Status:   ConnectionStatusConnected,
		Groups:   make(map[string]bool),
		Metadata: make(map[string]interface{}),
	}

	return connection, wsConn
}

func TestGenerateConnectionID(t *testing.T) {
	id1 := generateConnectionID()
	id2 := generateConnectionID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "conn_")
}

func TestConnection_SendMessage(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	// Register connection
	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	msg := &Message{
		Type:      MessageTypeChat,
		Data:      "test message",
		Timestamp: time.Now().Unix(),
	}

	err := conn.SendMessage(msg)
	assert.NoError(t, err)

	// Unregister
	hub.unregister <- conn
	time.Sleep(50 * time.Millisecond)
}

func TestConnection_SendError(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	err := conn.SendError(ErrConnectionClosed)
	assert.NoError(t, err)

	hub.unregister <- conn
	time.Sleep(50 * time.Millisecond)
}

func TestConnection_SendSuccess(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	err := conn.SendSuccess(MsgMessageSent)
	assert.NoError(t, err)

	hub.unregister <- conn
	time.Sleep(50 * time.Millisecond)
}

func TestConnection_JoinGroup(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	conn.JoinGroup("test-group")
	assert.True(t, conn.IsInGroup("test-group"))

	groups := conn.GetGroups()
	assert.Contains(t, groups, "test-group")

	hub.unregister <- conn
	time.Sleep(50 * time.Millisecond)
}

func TestConnection_LeaveGroup(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	conn.JoinGroup("test-group")
	assert.True(t, conn.IsInGroup("test-group"))

	conn.LeaveGroup("test-group")
	assert.False(t, conn.IsInGroup("test-group"))

	hub.unregister <- conn
	time.Sleep(50 * time.Millisecond)
}

func TestConnection_GetStatus(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	assert.Equal(t, ConnectionStatusConnected, conn.GetStatus())

	conn.SetStatus(ConnectionStatusReconnecting)
	assert.Equal(t, ConnectionStatusReconnecting, conn.GetStatus())

	wsConn.Close()
}

func TestConnection_Close(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	err := conn.Close()
	assert.NoError(t, err)
	assert.Equal(t, ConnectionStatusDisconnected, conn.GetStatus())
	assert.False(t, conn.IsAlive)

	hub.unregister <- conn
	time.Sleep(50 * time.Millisecond)
}

func TestConnection_handlePing(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	// Send ping message
	pingMsg := Message{
		Type:      MessageTypePing,
		Timestamp: time.Now().Unix(),
	}
	data, _ := json.Marshal(pingMsg)
	conn.handleMessage(data)

	// Wait for pong response
	time.Sleep(100 * time.Millisecond)

	hub.unregister <- conn
	time.Sleep(50 * time.Millisecond)
}

func TestConnection_handleJoinGroup(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	joinMsg := Message{
		Type:      MessageTypeJoinGroup,
		Data:      "test-group",
		Timestamp: time.Now().Unix(),
	}
	data, _ := json.Marshal(joinMsg)
	conn.handleMessage(data)

	time.Sleep(100 * time.Millisecond)
	assert.True(t, conn.IsInGroup("test-group"))

	hub.unregister <- conn
	time.Sleep(50 * time.Millisecond)
}

func TestConnection_handleLeaveGroup(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	// Join first
	conn.JoinGroup("test-group")

	// Leave
	leaveMsg := Message{
		Type:      MessageTypeLeaveGroup,
		Data:      "test-group",
		Timestamp: time.Now().Unix(),
	}
	data, _ := json.Marshal(leaveMsg)
	conn.handleMessage(data)

	time.Sleep(100 * time.Millisecond)
	assert.False(t, conn.IsInGroup("test-group"))

	hub.unregister <- conn
	time.Sleep(50 * time.Millisecond)
}

func TestConnection_handleChat(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	chatMsg := Message{
		Type:      MessageTypeChat,
		Data:      map[string]interface{}{"text": "hello"},
		To:        "user2",
		Timestamp: time.Now().Unix(),
	}
	data, _ := json.Marshal(chatMsg)
	conn.handleMessage(data)

	time.Sleep(100 * time.Millisecond)

	hub.unregister <- conn
	time.Sleep(50 * time.Millisecond)
}

func TestConnection_handleStatus(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer func() {
		if wsConn != nil {
			wsConn.Close()
		}
	}()

	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	statusMsg := Message{
		Type:      MessageTypeStatus,
		Data:      map[string]interface{}{"status": "busy"},
		Timestamp: time.Now().Unix(),
	}
	data, _ := json.Marshal(statusMsg)
	conn.handleMessage(data)

	time.Sleep(100 * time.Millisecond)

	hub.unregister <- conn
	time.Sleep(50 * time.Millisecond)
}

func TestConnection_SendMessage_BufferFull(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	// Create a dummy connection with very small buffer
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = upgrader.Upgrade(w, r, nil)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Skipf("Failed to create test connection: %v", err)
		return
	}
	defer wsConn.Close()

	// Create connection with very small buffer (size 1)
	smallBuffer := make(chan []byte, 1)
	conn := &Connection{
		ID:       generateConnectionID(),
		UserID:   "user1",
		Conn:     wsConn,
		Send:     smallBuffer,
		Hub:      hub,
		IsAlive:  true,
		Status:   ConnectionStatusConnected,
		Groups:   make(map[string]bool),
		Metadata: make(map[string]interface{}),
	}

	// Fill buffer
	msg1 := &Message{Type: MessageTypeChat, Data: "test1"}
	data1, _ := json.Marshal(msg1)
	select {
	case smallBuffer <- data1:
		// Buffer filled successfully
	default:
		t.Fatal("Failed to fill buffer - this shouldn't happen")
	}

	// Try to send another message - should fail because buffer is full
	msg2 := &Message{Type: MessageTypeChat, Data: "test2"}
	err2 := conn.SendMessage(msg2)
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), ErrSendBufferFull)
}
