package websocket

import (
	"fmt"
	"time"

	"github.com/code-100-precent/LingFramework/pkg/utils"
)

// LoadConfigFromEnv loads WebSocket configuration from environment variables
func LoadConfigFromEnv() *Config {
	config := DefaultConfig()

	// Load configuration from environment variables
	if maxConnections := utils.GetIntEnv(EnvWebSocketMaxConnections); maxConnections > 0 {
		config.MaxConnections = maxConnections
	}

	if heartbeatInterval := utils.GetIntEnv(EnvWebSocketHeartbeatInterval); heartbeatInterval > 0 {
		config.HeartbeatInterval = time.Duration(heartbeatInterval) * time.Second
	}

	if connectionTimeout := utils.GetIntEnv(EnvWebSocketConnectionTimeout); connectionTimeout > 0 {
		config.ConnectionTimeout = time.Duration(connectionTimeout) * time.Second
	}

	if messageBufferSize := utils.GetIntEnv(EnvWebSocketMessageBufferSize); messageBufferSize > 0 {
		config.MessageBufferSize = int(messageBufferSize)
	}

	if messageQueueSize := utils.GetIntEnv(EnvWebSocketMessageQueueSize); messageQueueSize > 0 {
		config.MessageQueueSize = int(messageQueueSize)
	}

	if shardCount := utils.GetIntEnv(EnvWebSocketShardCount); shardCount > 0 {
		config.ShardCount = int(shardCount)
	}

	if workerCount := utils.GetIntEnv(EnvWebSocketBroadcastWorkers); workerCount > 0 {
		config.BroadcastWorkerCount = int(workerCount)
	}

	if enableCompression := utils.GetEnv(EnvWebSocketEnableCompression); enableCompression != "" {
		config.EnableCompression = enableCompression == "true" || enableCompression == "1"
	}

	if enableMessageQueue := utils.GetEnv(EnvWebSocketEnableMessageQueue); enableMessageQueue != "" {
		config.EnableMessageQueue = enableMessageQueue == "true" || enableMessageQueue == "1"
	}

	if enableCluster := utils.GetEnv(EnvWebSocketEnableCluster); enableCluster != "" {
		config.EnableCluster = enableCluster == "true" || enableCluster == "1"
	}

	if clusterNodeID := utils.GetEnv(EnvWebSocketClusterNodeID); clusterNodeID != "" {
		config.ClusterNodeID = clusterNodeID
	}

	if dropOnFull := utils.GetEnv(EnvWebSocketDropOnFull); dropOnFull != "" {
		config.DropOnFull = dropOnFull == "true" || dropOnFull == "1"
	}

	if compressionLevel := utils.GetIntEnv(EnvWebSocketCompressionLevel); compressionLevel != 0 {
		config.CompressionLevel = int(compressionLevel)
	}

	if readBuf := utils.GetIntEnv(EnvWebSocketReadBufferSize); readBuf > 0 {
		config.ReadBufferSize = int(readBuf)
	}

	if writeBuf := utils.GetIntEnv(EnvWebSocketWriteBufferSize); writeBuf > 0 {
		config.WriteBufferSize = int(writeBuf)
	}

	if maxMsg := utils.GetIntEnv(EnvWebSocketMaxMessageSize); maxMsg > 0 {
		config.MaxMessageSize = int(maxMsg)
	}

	if closeOnBp := utils.GetEnv(EnvWebSocketCloseOnBackpressure); closeOnBp != "" {
		config.CloseOnBackpressure = closeOnBp == "true" || closeOnBp == "1"
	}

	if sendTimeoutMs := utils.GetIntEnv(EnvWebSocketSendTimeoutMs); sendTimeoutMs > 0 {
		config.SendTimeout = time.Duration(sendTimeoutMs) * time.Millisecond
	}

	if enableGlobalPing := utils.GetEnv(EnvWebSocketEnableGlobalPing); enableGlobalPing != "" {
		config.EnableGlobalPing = enableGlobalPing == "true" || enableGlobalPing == "1"
	}

	if pingWorkers := utils.GetIntEnv(EnvWebSocketPingWorkers); pingWorkers > 0 {
		config.PingWorkerCount = int(pingWorkers)
	}

	return config
}

// ValidateConfig validates WebSocket configuration
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.MaxConnections <= 0 {
		return fmt.Errorf("max connections must be greater than 0")
	}

	if config.HeartbeatInterval <= 0 {
		return fmt.Errorf("heartbeat interval must be greater than 0")
	}

	if config.ConnectionTimeout <= 0 {
		return fmt.Errorf("connection timeout must be greater than 0")
	}

	if config.MessageBufferSize <= 0 {
		return fmt.Errorf("message buffer size must be greater than 0")
	}

	if config.MessageQueueSize <= 0 {
		return fmt.Errorf("message queue size must be greater than 0")
	}

	if config.ShardCount <= 0 {
		return fmt.Errorf("shard count must be greater than 0")
	}

	if config.BroadcastWorkerCount <= 0 {
		return fmt.Errorf("broadcast worker count must be greater than 0")
	}

	if config.CompressionLevel < -2 || config.CompressionLevel > 9 {
		return fmt.Errorf("compression level must be between -2 and 9")
	}

	if config.ReadBufferSize <= 0 || config.WriteBufferSize <= 0 {
		return fmt.Errorf("read/write buffer size must be greater than 0")
	}

	if config.MaxMessageSize <= 0 {
		return fmt.Errorf("max message size must be greater than 0")
	}

	// Heartbeat interval should be less than connection timeout
	if config.HeartbeatInterval >= config.ConnectionTimeout {
		return fmt.Errorf("heartbeat interval must be less than connection timeout")
	}

	if config.CloseOnBackpressure && config.SendTimeout <= 0 {
		return fmt.Errorf("send timeout must be set when close on backpressure is enabled")
	}

	if config.EnableGlobalPing && config.PingWorkerCount <= 0 {
		return fmt.Errorf("ping worker count must be greater than 0 when global ping is enabled")
	}

	return nil
}

// GetConfigSummary gets configuration summary
func GetConfigSummary(config *Config) map[string]interface{} {
	return map[string]interface{}{
		"max_connections":       config.MaxConnections,
		"heartbeat_interval":    config.HeartbeatInterval.String(),
		"connection_timeout":    config.ConnectionTimeout.String(),
		"message_buffer_size":   config.MessageBufferSize,
		"message_queue_size":    config.MessageQueueSize,
		"read_buffer_size":      config.ReadBufferSize,
		"write_buffer_size":     config.WriteBufferSize,
		"max_message_size":      config.MaxMessageSize,
		"enable_compression":    config.EnableCompression,
		"enable_message_queue":  config.EnableMessageQueue,
		"enable_cluster":        config.EnableCluster,
		"cluster_node_id":       config.ClusterNodeID,
		"shard_count":           config.ShardCount,
		"broadcast_workers":     config.BroadcastWorkerCount,
		"drop_on_full":          config.DropOnFull,
		"compression_level":     config.CompressionLevel,
		"close_on_backpressure": config.CloseOnBackpressure,
		"send_timeout":          config.SendTimeout.String(),
		"enable_global_ping":    config.EnableGlobalPing,
		"ping_workers":          config.PingWorkerCount,
	}
}

// CloneConfig clones configuration
func CloneConfig(config *Config) *Config {
	if config == nil {
		return nil
	}

	return &Config{
		MaxConnections:       config.MaxConnections,
		HeartbeatInterval:    config.HeartbeatInterval,
		ConnectionTimeout:    config.ConnectionTimeout,
		MessageBufferSize:    config.MessageBufferSize,
		ReadBufferSize:       config.ReadBufferSize,
		WriteBufferSize:      config.WriteBufferSize,
		MaxMessageSize:       config.MaxMessageSize,
		EnableCompression:    config.EnableCompression,
		EnableMessageQueue:   config.EnableMessageQueue,
		MessageQueueSize:     config.MessageQueueSize,
		EnableCluster:        config.EnableCluster,
		ClusterNodeID:        config.ClusterNodeID,
		ShardCount:           config.ShardCount,
		BroadcastWorkerCount: config.BroadcastWorkerCount,
		DropOnFull:           config.DropOnFull,
		CompressionLevel:     config.CompressionLevel,
		CloseOnBackpressure:  config.CloseOnBackpressure,
		SendTimeout:          config.SendTimeout,
		EnableGlobalPing:     config.EnableGlobalPing,
		PingWorkerCount:      config.PingWorkerCount,
	}
}

// MergeConfig merges configurations (later configs override earlier ones)
func MergeConfig(configs ...*Config) *Config {
	if len(configs) == 0 {
		return DefaultConfig()
	}

	if len(configs) == 1 {
		return configs[0]
	}

	// Handle nil first config
	if configs[0] == nil {
		return DefaultConfig()
	}

	result := CloneConfig(configs[0])

	for i := 1; i < len(configs); i++ {
		config := configs[i]
		if config == nil {
			continue
		}

		if config.MaxConnections > 0 {
			result.MaxConnections = config.MaxConnections
		}
		if config.HeartbeatInterval > 0 {
			result.HeartbeatInterval = config.HeartbeatInterval
		}
		if config.ConnectionTimeout > 0 {
			result.ConnectionTimeout = config.ConnectionTimeout
		}
		if config.MessageBufferSize > 0 {
			result.MessageBufferSize = config.MessageBufferSize
		}
		if config.MessageQueueSize > 0 {
			result.MessageQueueSize = config.MessageQueueSize
		}
		if config.ReadBufferSize > 0 {
			result.ReadBufferSize = config.ReadBufferSize
		}
		if config.WriteBufferSize > 0 {
			result.WriteBufferSize = config.WriteBufferSize
		}
		if config.MaxMessageSize > 0 {
			result.MaxMessageSize = config.MaxMessageSize
		}
		if config.ClusterNodeID != "" {
			result.ClusterNodeID = config.ClusterNodeID
		}

		// Boolean values are directly overwritten
		result.EnableCompression = config.EnableCompression
		result.EnableMessageQueue = config.EnableMessageQueue
		result.EnableCluster = config.EnableCluster
		result.DropOnFull = config.DropOnFull
		result.CloseOnBackpressure = config.CloseOnBackpressure
		result.EnableGlobalPing = config.EnableGlobalPing

		if config.ShardCount > 0 {
			result.ShardCount = config.ShardCount
		}
		if config.BroadcastWorkerCount > 0 {
			result.BroadcastWorkerCount = config.BroadcastWorkerCount
		}
		if config.CompressionLevel != 0 { // Allow -2..9, 0 means not explicitly set
			result.CompressionLevel = config.CompressionLevel
		}
		if config.SendTimeout > 0 {
			result.SendTimeout = config.SendTimeout
		}
		if config.PingWorkerCount > 0 {
			result.PingWorkerCount = config.PingWorkerCount
		}
	}

	return result
}
