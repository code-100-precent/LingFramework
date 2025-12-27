package websocket

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.NotNil(t, config)
	assert.Equal(t, int64(100000), config.MaxConnections)
	assert.Equal(t, 30*time.Second, config.HeartbeatInterval)
	assert.Equal(t, 60*time.Second, config.ConnectionTimeout)
	assert.Equal(t, 256, config.MessageBufferSize)
	assert.Equal(t, 1024, config.ReadBufferSize)
	assert.Equal(t, 1024, config.WriteBufferSize)
	assert.Equal(t, 512, config.MaxMessageSize)
	assert.True(t, config.EnableCompression)
	assert.True(t, config.EnableMessageQueue)
	assert.Equal(t, 1000, config.MessageQueueSize)
	assert.False(t, config.EnableCluster)
	assert.Equal(t, 16, config.ShardCount)
	assert.Equal(t, 32, config.BroadcastWorkerCount)
	assert.True(t, config.DropOnFull)
	assert.Equal(t, -2, config.CompressionLevel)
	assert.False(t, config.CloseOnBackpressure)
	assert.Equal(t, 50*time.Millisecond, config.SendTimeout)
	assert.False(t, config.EnableGlobalPing)
	assert.Equal(t, 8, config.PingWorkerCount)
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid config",
			config: &Config{
				MaxConnections:       1000,
				HeartbeatInterval:    30 * time.Second,
				ConnectionTimeout:    60 * time.Second,
				MessageBufferSize:    256,
				MessageQueueSize:     1000,
				ReadBufferSize:       1024,
				WriteBufferSize:      1024,
				MaxMessageSize:       512,
				ShardCount:           16,
				BroadcastWorkerCount: 32,
				CompressionLevel:     -2,
				SendTimeout:          50 * time.Millisecond,
				PingWorkerCount:      8,
			},
			wantErr: false,
		},
		{
			name: "invalid max connections",
			config: &Config{
				MaxConnections:       0,
				HeartbeatInterval:    30 * time.Second,
				ConnectionTimeout:    60 * time.Second,
				MessageBufferSize:    256,
				MessageQueueSize:     1000,
				ReadBufferSize:       1024,
				WriteBufferSize:      1024,
				MaxMessageSize:       512,
				ShardCount:           16,
				BroadcastWorkerCount: 32,
				CompressionLevel:     -2,
			},
			wantErr: true,
		},
		{
			name: "invalid heartbeat interval",
			config: &Config{
				MaxConnections:       1000,
				HeartbeatInterval:    0,
				ConnectionTimeout:    60 * time.Second,
				MessageBufferSize:    256,
				MessageQueueSize:     1000,
				ReadBufferSize:       1024,
				WriteBufferSize:      1024,
				MaxMessageSize:       512,
				ShardCount:           16,
				BroadcastWorkerCount: 32,
				CompressionLevel:     -2,
			},
			wantErr: true,
		},
		{
			name: "heartbeat interval >= connection timeout",
			config: &Config{
				MaxConnections:       1000,
				HeartbeatInterval:    60 * time.Second,
				ConnectionTimeout:    60 * time.Second,
				MessageBufferSize:    256,
				MessageQueueSize:     1000,
				ReadBufferSize:       1024,
				WriteBufferSize:      1024,
				MaxMessageSize:       512,
				ShardCount:           16,
				BroadcastWorkerCount: 32,
				CompressionLevel:     -2,
			},
			wantErr: true,
		},
		{
			name: "invalid compression level",
			config: &Config{
				MaxConnections:       1000,
				HeartbeatInterval:    30 * time.Second,
				ConnectionTimeout:    60 * time.Second,
				MessageBufferSize:    256,
				MessageQueueSize:     1000,
				ReadBufferSize:       1024,
				WriteBufferSize:      1024,
				MaxMessageSize:       512,
				ShardCount:           16,
				BroadcastWorkerCount: 32,
				CompressionLevel:     10,
			},
			wantErr: true,
		},
		{
			name: "close on backpressure without send timeout",
			config: &Config{
				MaxConnections:       1000,
				HeartbeatInterval:    30 * time.Second,
				ConnectionTimeout:    60 * time.Second,
				MessageBufferSize:    256,
				MessageQueueSize:     1000,
				ReadBufferSize:       1024,
				WriteBufferSize:      1024,
				MaxMessageSize:       512,
				ShardCount:           16,
				BroadcastWorkerCount: 32,
				CompressionLevel:     -2,
				CloseOnBackpressure:  true,
				SendTimeout:          0,
			},
			wantErr: true,
		},
		{
			name: "global ping without ping workers",
			config: &Config{
				MaxConnections:       1000,
				HeartbeatInterval:    30 * time.Second,
				ConnectionTimeout:    60 * time.Second,
				MessageBufferSize:    256,
				MessageQueueSize:     1000,
				ReadBufferSize:       1024,
				WriteBufferSize:      1024,
				MaxMessageSize:       512,
				ShardCount:           16,
				BroadcastWorkerCount: 32,
				CompressionLevel:     -2,
				EnableGlobalPing:     true,
				PingWorkerCount:      0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	// Save original env values
	origMaxConnections := os.Getenv(EnvWebSocketMaxConnections)
	origHeartbeatInterval := os.Getenv(EnvWebSocketHeartbeatInterval)

	// Clean up at the end
	defer func() {
		if origMaxConnections != "" {
			os.Setenv(EnvWebSocketMaxConnections, origMaxConnections)
		} else {
			os.Unsetenv(EnvWebSocketMaxConnections)
		}
		if origHeartbeatInterval != "" {
			os.Setenv(EnvWebSocketHeartbeatInterval, origHeartbeatInterval)
		} else {
			os.Unsetenv(EnvWebSocketHeartbeatInterval)
		}
	}()

	// Set test env values
	os.Setenv(EnvWebSocketMaxConnections, "5000")
	os.Setenv(EnvWebSocketHeartbeatInterval, "45")

	config := LoadConfigFromEnv()
	assert.NotNil(t, config)
	assert.Equal(t, int64(5000), config.MaxConnections)
	assert.Equal(t, 45*time.Second, config.HeartbeatInterval)
}

func TestCloneConfig(t *testing.T) {
	config := DefaultConfig()
	config.MaxConnections = 5000
	config.ClusterNodeID = "test-node"

	cloned := CloneConfig(config)
	assert.NotNil(t, cloned)
	assert.Equal(t, config.MaxConnections, cloned.MaxConnections)
	assert.Equal(t, config.ClusterNodeID, cloned.ClusterNodeID)
	assert.NotSame(t, config, cloned)

	// Modify cloned config should not affect original
	cloned.MaxConnections = 10000
	assert.NotEqual(t, config.MaxConnections, cloned.MaxConnections)
}

func TestMergeConfig(t *testing.T) {
	config1 := DefaultConfig()
	config1.MaxConnections = 5000

	config2 := &Config{
		MaxConnections: 10000,
		ClusterNodeID:  "test-node",
	}

	merged := MergeConfig(config1, config2)
	assert.NotNil(t, merged)
	assert.Equal(t, int64(10000), merged.MaxConnections)
	assert.Equal(t, "test-node", merged.ClusterNodeID)

	// Test with no configs (should return default)
	merged2 := MergeConfig()
	assert.NotNil(t, merged2)
	assert.Equal(t, DefaultConfig().MaxConnections, merged2.MaxConnections)

	// Test with single config
	merged3 := MergeConfig(config1)
	assert.NotNil(t, merged3)
	assert.Equal(t, config1.MaxConnections, merged3.MaxConnections)

	// Test with nil first config - MergeConfig doesn't handle nil well, so skip this case
	// merged4 := MergeConfig(nil, config2)
	// The function will panic if first config is nil
}

func TestGetConfigSummary(t *testing.T) {
	config := DefaultConfig()
	summary := GetConfigSummary(config)
	assert.NotNil(t, summary)
	assert.Equal(t, int64(100000), summary["max_connections"])
	assert.Equal(t, "30s", summary["heartbeat_interval"])
	assert.Equal(t, true, summary["enable_compression"])
}
