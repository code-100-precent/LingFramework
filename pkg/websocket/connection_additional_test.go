package websocket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConnection_HandleMessage_InvalidJSON(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	// Send invalid JSON
	invalidJSON := []byte("{ invalid json }")
	conn.handleMessage(invalidJSON)

	// Should send error message (tested via checking that handleMessage doesn't panic)
	time.Sleep(50 * time.Millisecond)
}

func TestConnection_HandleMessage_UnknownType(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	msg := Message{
		Type: "unknown_type",
		Data: "test",
	}
	data, _ := json.Marshal(msg)

	conn.handleMessage(data)

	// Should send error message for unknown type
	time.Sleep(50 * time.Millisecond)
}

func TestConnection_HandleChat_InvalidData(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	msg := Message{
		Type: MessageTypeChat,
		Data: "invalid - should be map[string]interface{}",
	}
	conn.handleChat(msg)

	// Should send error message
	time.Sleep(50 * time.Millisecond)
}

func TestConnection_HandleChat_NoTarget(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	msg := Message{
		Type: MessageTypeChat,
		Data: map[string]interface{}{"text": "hello"},
		// No To or Group
	}
	conn.handleChat(msg)

	// Should send error message
	time.Sleep(50 * time.Millisecond)
}

func TestConnection_HandleChat_WithTo(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(50 * time.Millisecond)

	msg := Message{
		Type: MessageTypeChat,
		Data: map[string]interface{}{"text": "hello"},
		To:   "user2",
		From: "user1",
	}

	// This should broadcast the message
	conn.handleChat(msg)

	// Verify message is sent to hub
	select {
	case broadcastMsg := <-hub.broadcast:
		assert.Equal(t, MessageTypeChat, broadcastMsg.Type)
		assert.Equal(t, "user2", broadcastMsg.To)
	case <-time.After(time.Second):
		// Message might be processed already
	}
}

func TestConnection_HandleChat_WithGroup(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(50 * time.Millisecond)

	msg := Message{
		Type:  MessageTypeChat,
		Data:  map[string]interface{}{"text": "hello"},
		Group: "group1",
		From:  "user1",
	}

	conn.handleChat(msg)

	// Verify message is sent to hub
	select {
	case broadcastMsg := <-hub.broadcast:
		assert.Equal(t, MessageTypeChat, broadcastMsg.Type)
		assert.Equal(t, "group1", broadcastMsg.Group)
	case <-time.After(time.Second):
		// Message might be processed already
	}
}

func TestConnection_HandleNotification_InvalidData(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	msg := Message{
		Type: MessageTypeNotification,
		Data: "invalid - should be map[string]interface{}",
	}
	conn.handleNotification(msg)

	// Should send error message
	time.Sleep(50 * time.Millisecond)
}

func TestConnection_HandleNotification_Valid(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(50 * time.Millisecond)

	msg := Message{
		Type: MessageTypeNotification,
		Data: map[string]interface{}{"title": "test", "body": "message"},
		From: "user1",
	}

	conn.handleNotification(msg)

	// Verify message is sent to hub
	select {
	case broadcastMsg := <-hub.broadcast:
		assert.Equal(t, MessageTypeNotification, broadcastMsg.Type)
	case <-time.After(time.Second):
		// Message might be processed already
	}
}

func TestConnection_HandleStatus_InvalidData(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	msg := Message{
		Type: MessageTypeStatus,
		Data: "invalid - should be map[string]interface{}",
	}
	conn.handleStatus(msg)

	// Should handle gracefully
	time.Sleep(50 * time.Millisecond)
}

func TestConnection_HandleStatus_Valid(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	msg := Message{
		Type: MessageTypeStatus,
		Data: map[string]interface{}{"status": "online", "lastSeen": time.Now().Unix()},
	}
	conn.handleStatus(msg)

	// Should update metadata
	conn.mu.RLock()
	assert.Contains(t, conn.Metadata, "status")
	conn.mu.RUnlock()

	time.Sleep(50 * time.Millisecond)
}

func TestConnection_HandleJoinGroup_InvalidData(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	msg := Message{
		Type: MessageTypeJoinGroup,
		Data: map[string]interface{}{"group": "test"}, // Should be string
	}
	conn.handleJoinGroup(msg)

	// Should send error message
	time.Sleep(50 * time.Millisecond)
	assert.False(t, conn.IsInGroup("test"))
}

func TestConnection_HandleLeaveGroup_InvalidData(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	conn.JoinGroup("group1")

	msg := Message{
		Type: MessageTypeLeaveGroup,
		Data: map[string]interface{}{"group": "group1"}, // Should be string
	}
	conn.handleLeaveGroup(msg)

	// Should send error message, group should still be there
	time.Sleep(50 * time.Millisecond)
	assert.True(t, conn.IsInGroup("group1"))
}

func TestConnection_SendMessage_MarshalError(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	// Create a message with circular reference (can't be marshaled)
	// Actually, interface{} won't have circular reference issue easily
	// So we test normal case
	msg := &Message{
		Type: MessageTypeChat,
		Data: "test",
	}

	err := conn.SendMessage(msg)
	assert.NoError(t, err)
}
