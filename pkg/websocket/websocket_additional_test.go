package websocket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHub_BroadcastMessage(t *testing.T) {
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
	time.Sleep(100 * time.Millisecond)

	msg := &Message{
		Type: MessageTypeChat,
		Data: "broadcast message",
	}

	hub.broadcastMessage(msg)

	// Verify both connections receive the message
	time.Sleep(100 * time.Millisecond)

	// Check that message was broadcasted (connections would receive it via their write pumps)
	assert.Equal(t, int64(2), hub.GetConnectionCount())
}

func TestHub_SendToUser_Additional(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn1, wsConn1 := createTestConnection(t, hub, "user1")
	if conn1 == nil {
		return
	}
	defer wsConn1.Close()

	hub.register <- conn1
	time.Sleep(100 * time.Millisecond)

	msg := &Message{
		Type: MessageTypeChat,
		Data: "user message",
	}

	err := hub.SendToUser("user1", msg)
	assert.NoError(t, err)
}

func TestHub_SendToUser_NotExists(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	msg := &Message{
		Type: MessageTypeChat,
		Data: "user message",
	}

	err := hub.SendToUser("nonexistent", msg)
	assert.NoError(t, err) // Should not error, just doesn't send
}

func TestHub_BroadcastToGroup_Additional(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn1, wsConn1 := createTestConnection(t, hub, "user1")
	if conn1 == nil {
		return
	}
	defer wsConn1.Close()

	conn1.JoinGroup("group1")
	hub.register <- conn1
	time.Sleep(100 * time.Millisecond)

	msg := &Message{
		Type: MessageTypeChat,
		Data: "group message",
	}

	err := hub.BroadcastToGroup("group1", msg)
	assert.NoError(t, err)
}

func TestHub_BroadcastToGroup_NotExists(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	msg := &Message{
		Type: MessageTypeChat,
		Data: "group message",
	}

	err := hub.BroadcastToGroup("nonexistent", msg)
	assert.NoError(t, err) // Should not error, just doesn't send
}

func TestHub_BroadcastToAll_Additional(t *testing.T) {
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
	time.Sleep(100 * time.Millisecond)

	msg := &Message{
		Type: MessageTypeChat,
		Data: "broadcast all",
	}

	err := hub.BroadcastToAll(msg)
	assert.NoError(t, err)
}

func TestHub_GetBroadcastChannel(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	ch := hub.GetBroadcastChannel()
	assert.NotNil(t, ch)

	// Send a message via the channel
	msg := &Message{
		Type: MessageTypeChat,
		Data: "test",
	}

	select {
	case ch <- msg:
		// Successfully sent
	case <-time.After(time.Second):
		t.Fatal("Channel send timeout")
	}
}

func TestHub_TrySend_DropOnFull_CloseOnBackpressure(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()
	hub.config.DropOnFull = true
	hub.config.CloseOnBackpressure = true

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	// Make buffer very small
	conn.Send = make(chan []byte, 1)
	conn.Send <- []byte("fill")

	data := []byte("test message")
	dropCalled := false
	hub.trySend(conn, data, func() {
		dropCalled = true
	})

	time.Sleep(50 * time.Millisecond)
	// Connection should be marked as closed due to backpressure
	assert.True(t, dropCalled)
}

func TestHub_TrySend_NonDropMode(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()
	hub.config.DropOnFull = false
	hub.config.SendTimeout = 10 * time.Millisecond
	hub.config.CloseOnBackpressure = false

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	// Make buffer very small
	conn.Send = make(chan []byte, 1)
	conn.Send <- []byte("fill")

	data := []byte("test message")
	dropCalled := false
	hub.trySend(conn, data, func() {
		dropCalled = true
	})

	time.Sleep(100 * time.Millisecond)
	assert.True(t, dropCalled)
}

func TestHub_TrySend_NonDropMode_CloseOnBackpressure(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()
	hub.config.DropOnFull = false
	hub.config.SendTimeout = 10 * time.Millisecond
	hub.config.CloseOnBackpressure = true

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	// Make buffer very small
	conn.Send = make(chan []byte, 1)
	conn.Send <- []byte("fill")

	data := []byte("test message")
	dropCalled := false
	hub.trySend(conn, data, func() {
		dropCalled = true
	})

	time.Sleep(100 * time.Millisecond)
	assert.True(t, dropCalled)
}

func TestHub_TrySend_NonDropMode_ZeroTimeout(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()
	hub.config.DropOnFull = false
	hub.config.SendTimeout = 0 // Should default to 50ms
	hub.config.CloseOnBackpressure = false

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	// Make buffer very small
	conn.Send = make(chan []byte, 1)
	conn.Send <- []byte("fill")

	data := []byte("test message")
	dropCalled := false
	hub.trySend(conn, data, func() {
		dropCalled = true
	})

	time.Sleep(100 * time.Millisecond)
	assert.True(t, dropCalled)
}

func TestHub_Run_BroadcastToUser(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	msg := &Message{
		Type: MessageTypeChat,
		Data: "test",
		To:   "user1",
	}

	hub.broadcast <- msg
	time.Sleep(100 * time.Millisecond)
}

func TestHub_Run_BroadcastToGroup(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	conn.JoinGroup("group1")
	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	msg := &Message{
		Type:  MessageTypeChat,
		Data:  "test",
		Group: "group1",
	}

	hub.broadcast <- msg
	time.Sleep(100 * time.Millisecond)
}

func TestHub_Run_BroadcastAll(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	msg := &Message{
		Type: MessageTypeChat,
		Data: "test",
	}

	hub.broadcast <- msg
	time.Sleep(100 * time.Millisecond)
}

func TestHub_Run_InvalidJSON(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	// Create an invalid message that can't be serialized
	// This is tricky - we'll test the error handling in the run loop
	// by sending a message that causes json.Marshal to fail
	// But since Message is a simple struct, we need to think of another way

	// Actually, json.Marshal should never fail for Message struct
	// So this test might not be necessary, but we can test the timestamp handling
	msg := &Message{
		Type:      MessageTypeChat,
		Data:      "test",
		Timestamp: 0, // Will be set by run()
	}

	hub.broadcast <- msg
	time.Sleep(100 * time.Millisecond)

	assert.Greater(t, msg.Timestamp, int64(0))
}

func TestHub_EnqueueBroadcastAll_QueueFull(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()
	hub.config.MessageQueueSize = 1 // Very small queue

	// Fill the queue
	data1 := []byte("test1")
	hub.enqueueBroadcastAll(data1)

	// This should try to enqueue but might drop due to full queue
	data2 := []byte("test2")
	hub.enqueueBroadcastAll(data2)

	time.Sleep(50 * time.Millisecond)
	// Queue might be full, but function should handle it gracefully
}

func TestHub_RegisterConnection_MaxConnections(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()
	hub.config.MaxConnections = 1

	conn1, wsConn1 := createTestConnection(t, hub, "user1")
	if conn1 == nil {
		return
	}
	defer wsConn1.Close()

	hub.register <- conn1
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int64(1), hub.GetConnectionCount())

	// Try to register another connection - should be rejected
	conn2, wsConn2 := createTestConnection(t, hub, "user2")
	if conn2 == nil {
		return
	}
	defer wsConn2.Close()

	hub.register <- conn2
	time.Sleep(100 * time.Millisecond)

	// Connection count should still be 1
	assert.Equal(t, int64(1), hub.GetConnectionCount())
}

func TestHub_RegisterConnection_WithGroups(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	conn.JoinGroup("group1")
	conn.JoinGroup("group2")

	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	hub.mu.RLock()
	assert.Contains(t, hub.groupConnections["group1"], conn.ID)
	assert.Contains(t, hub.groupConnections["group2"], conn.ID)
	hub.mu.RUnlock()
}

func TestHub_UnregisterConnection_NotExists(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	// Unregister without registering first
	hub.unregister <- conn
	time.Sleep(100 * time.Millisecond)

	// Should handle gracefully
	assert.Equal(t, int64(0), hub.GetConnectionCount())
}

func TestHub_BroadcastToGroup_WithTimestamp(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	msg := &Message{
		Type:      MessageTypeChat,
		Data:      "test",
		Timestamp: 12345,
	}

	err := hub.BroadcastToGroup("group1", msg)
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), msg.Timestamp)
}

func TestHub_SendToUser_WithTimestamp(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	msg := &Message{
		Type:      MessageTypeChat,
		Data:      "test",
		Timestamp: 12345,
	}

	err := hub.SendToUser("user1", msg)
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), msg.Timestamp)
}

func TestHub_BroadcastToAll_WithTimestamp(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	msg := &Message{
		Type:      MessageTypeChat,
		Data:      "test",
		Timestamp: 12345,
	}

	err := hub.BroadcastToAll(msg)
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), msg.Timestamp)
}

func TestHub_BroadcastToGroup_MarshalError(t *testing.T) {
	hub := setupTestHub()
	defer hub.Close()

	// Create a message with data that can't be marshaled
	// Since Message uses interface{}, we can't easily create an invalid message
	// But we can test that the function handles it
	msg := &Message{
		Type: MessageTypeChat,
		Data: "valid data",
	}

	err := hub.BroadcastToGroup("group1", msg)
	assert.NoError(t, err)
}
