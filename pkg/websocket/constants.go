package websocket

// WebSocket message type constants
const (
	// System message types
	MessageTypePing          = "ping"
	MessageTypePong          = "pong"
	MessageTypeJoinGroup     = "join_group"
	MessageTypeLeaveGroup    = "leave_group"
	MessageTypeGroupJoined   = "group_joined"
	MessageTypeGroupLeft     = "group_left"
	MessageTypeStatus        = "status"
	MessageTypeStatusUpdated = "status_updated"

	// Business message types
	MessageTypeChat         = "chat"
	MessageTypeNotification = "notification"
	MessageTypeSystem       = "system"
	MessageTypeError        = "error"
	MessageTypeSuccess      = "success"

	// Connection status constants
	ConnectionStatusConnected    = "connected"
	ConnectionStatusDisconnected = "disconnected"
	ConnectionStatusReconnecting = "reconnecting"
	ConnectionStatusError        = "error"

	// Default configuration values
	DefaultMaxConnections    = 100000
	DefaultHeartbeatInterval = 30
	DefaultConnectionTimeout = 60
	DefaultMessageBufferSize = 256
	DefaultMessageQueueSize  = 1000
	DefaultReadBufferSize    = 1024
	DefaultWriteBufferSize   = 1024
	DefaultMaxMessageSize    = 512

	// Environment variable configuration keys
	EnvWebSocketMaxConnections      = "WEBSOCKET_MAX_CONNECTIONS"
	EnvWebSocketHeartbeatInterval   = "WEBSOCKET_HEARTBEAT_INTERVAL"
	EnvWebSocketConnectionTimeout   = "WEBSOCKET_CONNECTION_TIMEOUT"
	EnvWebSocketMessageBufferSize   = "WEBSOCKET_MESSAGE_BUFFER_SIZE"
	EnvWebSocketMessageQueueSize    = "WEBSOCKET_MESSAGE_QUEUE_SIZE"
	EnvWebSocketEnableCompression   = "WEBSOCKET_ENABLE_COMPRESSION"
	EnvWebSocketEnableMessageQueue  = "WEBSOCKET_ENABLE_MESSAGE_QUEUE"
	EnvWebSocketEnableCluster       = "WEBSOCKET_ENABLE_CLUSTER"
	EnvWebSocketClusterNodeID       = "WEBSOCKET_CLUSTER_NODE_ID"
	EnvWebSocketShardCount          = "WEBSOCKET_SHARD_COUNT"
	EnvWebSocketBroadcastWorkers    = "WEBSOCKET_BROADCAST_WORKERS"
	EnvWebSocketDropOnFull          = "WEBSOCKET_DROP_ON_FULL"
	EnvWebSocketCompressionLevel    = "WEBSOCKET_COMPRESSION_LEVEL"
	EnvWebSocketReadBufferSize      = "WEBSOCKET_READ_BUFFER_SIZE"
	EnvWebSocketWriteBufferSize     = "WEBSOCKET_WRITE_BUFFER_SIZE"
	EnvWebSocketMaxMessageSize      = "WEBSOCKET_MAX_MESSAGE_SIZE"
	EnvWebSocketCloseOnBackpressure = "WEBSOCKET_CLOSE_ON_BACKPRESSURE"
	EnvWebSocketSendTimeoutMs       = "WEBSOCKET_SEND_TIMEOUT_MS"
	EnvWebSocketEnableGlobalPing    = "WEBSOCKET_ENABLE_GLOBAL_PING"
	EnvWebSocketPingWorkers         = "WEBSOCKET_PING_WORKERS"

	// Error messages
	ErrConnectionLimitExceeded = "connection limit exceeded"
	ErrInvalidMessageType      = "invalid message type"
	ErrInvalidMessageData      = "invalid message data"
	ErrUserNotFound            = "user not found"
	ErrGroupNotFound           = "group not found"
	ErrConnectionClosed        = "connection closed"
	ErrSendBufferFull          = "send buffer full"
	ErrReadTimeout             = "read timeout"
	ErrWriteTimeout            = "write timeout"

	// Success messages
	MsgConnectionEstablished = "connection established"
	MsgMessageSent           = "message sent"
	MsgGroupJoined           = "group joined"
	MsgGroupLeft             = "group left"
	MsgStatusUpdated         = "status updated"

	// Route paths
	RouteWebSocket          = "/ws"
	RouteWebSocketStats     = "/ws/stats"
	RouteWebSocketHealth    = "/ws/health"
	RouteWebSocketMessage   = "/ws/message"
	RouteWebSocketBroadcast = "/ws/broadcast"
	RouteWebSocketUser      = "/ws/user/:user_id"
	RouteWebSocketGroup     = "/ws/group/:group"
)
