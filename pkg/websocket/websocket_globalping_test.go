package websocket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHub_GlobalPing_Enabled(t *testing.T) {
	config := DefaultConfig()
	config.EnableGlobalPing = true
	config.PingWorkerCount = 2
	config.ShardCount = 2
	hub := NewHub(config)
	defer hub.Close()

	// Register connections
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

	// Trigger ping by sending a ping job
	select {
	case hub.pingJobs <- 0:
		// Successfully sent
	case <-time.After(time.Second):
		t.Fatal("Failed to send ping job")
	}

	// Give time for ping worker to process
	time.Sleep(100 * time.Millisecond)

	// Verify connections are still alive
	assert.True(t, conn1.IsAlive)
	assert.True(t, conn2.IsAlive)
}

func TestHub_Run_GlobalPing(t *testing.T) {
	config := DefaultConfig()
	config.EnableGlobalPing = true
	config.PingWorkerCount = 1
	config.ShardCount = 2
	config.HeartbeatInterval = 100 * time.Millisecond
	hub := NewHub(config)
	defer hub.Close()

	// Register a connection
	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(200 * time.Millisecond)

	// Wait for at least one heartbeat interval to trigger global ping
	time.Sleep(200 * time.Millisecond)

	// Verify connection is still alive
	assert.True(t, conn.IsAlive)
}

func TestHub_GlobalPing_QueueFull(t *testing.T) {
	config := DefaultConfig()
	config.EnableGlobalPing = true
	config.PingWorkerCount = 1
	config.ShardCount = 2
	hub := NewHub(config)
	defer hub.Close()

	// Fill the ping jobs queue (small buffer)
	// Note: pingJobs channel size is same as shardCount, so we can fill it
	for i := 0; i < hub.shardCount; i++ {
		select {
		case hub.pingJobs <- i:
		default:
			// Queue full, this is expected behavior
		}
	}

	// Try to send one more - should fail gracefully
	select {
	case hub.pingJobs <- 0:
		t.Fatal("Expected queue to be full")
	default:
		// Expected - queue is full
	}

	time.Sleep(50 * time.Millisecond)
}
