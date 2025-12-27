package websocket

import (
	"fmt"
	"net/http"
	"time"

	"github.com/code-100-precent/LingFramework/internal/models"
	"github.com/code-100-precent/LingFramework/pkg/constants"
	"github.com/code-100-precent/LingFramework/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler is a WebSocket HTTP handler
type Handler struct {
	hub *Hub
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub) *Handler {
	return &Handler{
		hub: hub,
	}
}

// RegisterRoutes registers all routes
func RegisterRoutes(r *gin.Engine, handler *Handler) {
	r.GET(RouteWebSocket, handler.HandleWebSocket)
	r.GET(RouteWebSocketStats, handler.GetStats)
	r.GET(RouteWebSocketHealth, handler.HealthCheck)
	r.POST(RouteWebSocketMessage, handler.SendMessage)
	r.POST(RouteWebSocketBroadcast, handler.BroadcastMessage)
}

// HandleWebSocket handles WebSocket connection request
func (h *Handler) HandleWebSocket(c *gin.Context) {
	// Get user ID (from authentication middleware)
	userObj, exists := c.Get(constants.UserField)
	if !exists {
		if logger.Lg != nil {
			logger.Error("unauthenticated user")
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated user"})
		return
	}

	var userIDStr string
	// Handle different types of user ID: could be *User object or string
	switch v := userObj.(type) {
	case *models.User:
		// Extract ID from User object and convert to string
		userIDStr = fmt.Sprintf("%d", v.ID)
	case string:
		// If already a string, use it directly
		userIDStr = v
	default:
		if logger.Lg != nil {
			logger.Error("invalid user ID type", zap.String("type", fmt.Sprintf("%T", userObj)))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID"})
		return
	}

	if userIDStr == "" {
		if logger.Lg != nil {
			logger.Error("user ID is empty")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user ID is empty"})
		return
	}

	// Handle WebSocket upgrade
	HandleWebSocket(h.hub, c.Writer, c.Request, userIDStr)
}

// HandleAnonymousWebSocket handles anonymous WebSocket connection (optional)
func (h *Handler) HandleAnonymousWebSocket(c *gin.Context) {
	// Generate anonymous user ID
	anonymousID := "anonymous_" + c.Request.Header.Get("X-Request-ID")
	if anonymousID == "anonymous_" {
		anonymousID = "anonymous_" + c.Request.Header.Get("X-Real-IP")
	}

	// Handle WebSocket upgrade
	HandleWebSocket(h.hub, c.Writer, c.Request, anonymousID)
}

// GetStats gets WebSocket statistics
func (h *Handler) GetStats(c *gin.Context) {
	stats := gin.H{
		"total_connections":    h.hub.GetConnectionCount(),
		"max_connections":      h.hub.config.MaxConnections,
		"heartbeat_interval":   h.hub.config.HeartbeatInterval.String(),
		"connection_timeout":   h.hub.config.ConnectionTimeout.String(),
		"message_buffer_size":  h.hub.config.MessageBufferSize,
		"enable_compression":   h.hub.config.EnableCompression,
		"enable_message_queue": h.hub.config.EnableMessageQueue,
		"message_queue_size":   h.hub.config.MessageQueueSize,
		"enable_cluster":       h.hub.config.EnableCluster,
		"cluster_node_id":      h.hub.config.ClusterNodeID,
		"read_buffer_size":     h.hub.config.ReadBufferSize,
		"write_buffer_size":    h.hub.config.WriteBufferSize,
		"max_message_size":     h.hub.config.MaxMessageSize,
		"shard_count":          h.hub.config.ShardCount,
		"broadcast_workers":    h.hub.config.BroadcastWorkerCount,
		"drop_on_full":         h.hub.config.DropOnFull,
		"compression_level":    h.hub.config.CompressionLevel,
	}

	c.JSON(http.StatusOK, stats)
}

// GetUserStats gets connection statistics for a specific user
func (h *Handler) GetUserStats(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ID cannot be empty"})
		return
	}

	connectionCount := h.hub.GetUserConnections(userID)
	stats := gin.H{
		"user_id":          userID,
		"connection_count": connectionCount,
		"max_connections":  h.hub.config.MaxConnections,
	}

	c.JSON(http.StatusOK, stats)
}

// GetGroupStats gets connection statistics for a specific group
func (h *Handler) GetGroupStats(c *gin.Context) {
	groupName := c.Param("group")
	if groupName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group name cannot be empty"})
		return
	}

	connectionCount := h.hub.GetGroupConnections(groupName)
	stats := gin.H{
		"group":            groupName,
		"connection_count": connectionCount,
		"max_connections":  h.hub.config.MaxConnections,
	}

	c.JSON(http.StatusOK, stats)
}

// SendMessage sends a message to a specific user or group
func (h *Handler) SendMessage(c *gin.Context) {
	var request struct {
		Type  string      `json:"type" binding:"required"`
		Data  interface{} `json:"data"`
		To    string      `json:"to,omitempty"`
		Group string      `json:"group,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request data: " + err.Error()})
		return
	}

	// Validate request
	if request.To == "" && request.Group == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "must specify target user or group"})
		return
	}

	// Create message
	message := &Message{
		Type:  request.Type,
		Data:  request.Data,
		To:    request.To,
		Group: request.Group,
	}

	// Broadcast message
	h.hub.broadcast <- message

	c.JSON(http.StatusOK, gin.H{"message": MsgMessageSent})
}

// BroadcastMessage broadcasts a message to all connections
func (h *Handler) BroadcastMessage(c *gin.Context) {
	var request struct {
		Type string      `json:"type" binding:"required"`
		Data interface{} `json:"data"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request data: " + err.Error()})
		return
	}

	// Create message
	message := &Message{
		Type: request.Type,
		Data: request.Data,
	}

	// Broadcast message
	h.hub.broadcast <- message

	c.JSON(http.StatusOK, gin.H{"message": "broadcast message sent"})
}

// DisconnectUser disconnects all connections for a specific user
func (h *Handler) DisconnectUser(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ID cannot be empty"})
		return
	}

	// Get all connections for the user
	h.hub.mu.RLock()
	connections, exists := h.hub.userConnections[userID]
	h.hub.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": ErrUserNotFound})
		return
	}

	// Disconnect all connections
	disconnectedCount := 0
	for connID := range connections {
		if conn, ok := h.hub.connections[connID]; ok {
			conn.Conn.Close()
			disconnectedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "user connections disconnected",
		"user_id":            userID,
		"disconnected_count": disconnectedCount,
	})
}

// DisconnectGroup disconnects all connections for a specific group
func (h *Handler) DisconnectGroup(c *gin.Context) {
	groupName := c.Param("group")
	if groupName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group name cannot be empty"})
		return
	}

	// Get all connections for the group
	h.hub.mu.RLock()
	connections, exists := h.hub.groupConnections[groupName]
	h.hub.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": ErrGroupNotFound})
		return
	}

	// Disconnect all connections
	disconnectedCount := 0
	for connID := range connections {
		if conn, ok := h.hub.connections[connID]; ok {
			conn.Conn.Close()
			disconnectedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "group connections disconnected",
		"group":              groupName,
		"disconnected_count": disconnectedCount,
	})
}

// HealthCheck performs WebSocket health check
func (h *Handler) HealthCheck(c *gin.Context) {
	// Check if Hub is running normally
	if h.hub.ctx.Err() != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"error":   "WebSocket Hub is closed",
			"details": h.hub.ctx.Err().Error(),
		})
		return
	}

	// Check if connection count is normal
	totalConnections := h.hub.GetConnectionCount()
	maxConnections := h.hub.config.MaxConnections

	status := "healthy"
	if totalConnections >= maxConnections*9/10 { // 90% or above is considered warning
		status = "warning"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":            status,
		"total_connections": totalConnections,
		"max_connections":   maxConnections,
		"connection_usage":  float64(totalConnections) / float64(maxConnections) * 100,
		"hub_running":       true,
		"timestamp":         time.Now().Unix(),
	})
}
