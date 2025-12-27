package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// newUpgrader creates a WebSocket upgrader based on configuration
func newUpgrader(cfg *Config) websocket.Upgrader {
	up := websocket.Upgrader{
		ReadBufferSize:  cfg.ReadBufferSize,
		WriteBufferSize: cfg.WriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			// In production, should check Origin
			return true
		},
		EnableCompression: cfg.EnableCompression,
	}
	return up
}

// HandleWebSocket handles WebSocket connection
func HandleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request, userID string) {
	// Upgrade HTTP connection to WebSocket
	upgrader := newUpgrader(hub.config)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Errorf("WebSocket upgrade failed: %v", err)
		return
	}

	// Compression settings
	if hub.config.EnableCompression {
		conn.EnableWriteCompression(true)
		if hub.config.CompressionLevel != 0 {
			_ = conn.SetCompressionLevel(hub.config.CompressionLevel)
		}
	}

	// Create connection instance
	connection := &Connection{
		ID:       generateConnectionID(),
		UserID:   userID,
		Conn:     conn,
		Send:     make(chan []byte, hub.config.MessageBufferSize),
		Hub:      hub,
		LastPing: time.Now(),
		IsAlive:  true,
		Status:   ConnectionStatusConnected,
		Groups:   make(map[string]bool),
		Metadata: make(map[string]interface{}),
	}

	// Register connection to Hub
	hub.register <- connection

	// Send connection established message after a short delay to ensure registration is complete
	go func() {
		time.Sleep(50 * time.Millisecond) // Small delay to ensure registration
		establishedMsg := Message{
			Type:      MessageTypeSuccess,
			Data:      MsgConnectionEstablished,
			Timestamp: time.Now().Unix(),
		}
		if data, err := json.Marshal(establishedMsg); err == nil {
			select {
			case connection.Send <- data:
			default:
				// Buffer full, skip
			}
		}
	}()

	// Start read/write goroutines
	go connection.writePump()
	go connection.readPump()
}

// generateConnectionID generates a unique connection ID
func generateConnectionID() string {
	return fmt.Sprintf("conn_%d", time.Now().UnixNano())
}

// readPump reads messages from the connection
func (c *Connection) readPump() {
	defer func() {
		c.mu.Lock()
		c.Status = ConnectionStatusDisconnected
		c.IsAlive = false
		c.mu.Unlock()
		c.Hub.unregister <- c
		// Gracefully close connection
		c.Conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(int64(c.Hub.config.MaxMessageSize))
	c.Conn.SetReadDeadline(time.Now().Add(c.Hub.config.ConnectionTimeout))
	c.Conn.SetPongHandler(func(string) error {
		c.mu.Lock()
		c.LastPing = time.Now()
		c.mu.Unlock()
		c.Conn.SetReadDeadline(time.Now().Add(c.Hub.config.ConnectionTimeout))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			// Check if it's a normal close
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				logrus.Debugf("WebSocket connection closed normally: %s, user: %s", c.ID, c.UserID)
			} else {
				// Check if it's close 1005 (no status) - usually means client directly closed connection (e.g., closing browser tab)
				// This should not be treated as an error, but normal client behavior
				errStr := err.Error()
				if strings.Contains(errStr, "close 1005") || strings.Contains(errStr, "no status") {
					logrus.Debugf("WebSocket connection closed by client: %s, user: %s", c.ID, c.UserID)
				} else if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					// Unexpected close error, log detailed information
					logrus.Errorf("WebSocket read error: %v, connection ID: %s, user: %s", err, c.ID, c.UserID)
				} else {
					// Other errors (timeout, network errors, etc.)
					logrus.Warnf("WebSocket connection error: %v, connection ID: %s, user: %s", err, c.ID, c.UserID)
				}
			}
			break
		}

		// Update read deadline after successfully reading message
		c.Conn.SetReadDeadline(time.Now().Add(c.Hub.config.ConnectionTimeout))

		// Handle received message
		c.handleMessage(message)
	}
}

// writePump sends messages to the connection
func (c *Connection) writePump() {
	var ticker *time.Ticker
	if !c.Hub.config.EnableGlobalPing {
		interval := c.Hub.config.HeartbeatInterval
		if interval <= 0 {
			interval = 30 * time.Second
		}
		pingEvery := time.Duration(float64(interval) * 0.9)
		ticker = time.NewTicker(pingEvery)
	}
	defer func() {
		if ticker != nil {
			ticker.Stop()
		}
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel closed, send close message
				c.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}

			// Send each message separately to avoid multi-line JSON parsing issues
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				// Write error, log it
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logrus.Warnf("WebSocket write error: %v, connection ID: %s, user: %s", err, c.ID, c.UserID)
				}
				return
			}
		case <-func() <-chan time.Time {
			if ticker != nil {
				return ticker.C
			}
			return make(chan time.Time)
		}():
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				// Ping failed, log it
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logrus.Warnf("WebSocket ping failed: %v, connection ID: %s, user: %s", err, c.ID, c.UserID)
				}
				return
			}
		}
	}
}

// handleMessage handles received messages
func (c *Connection) handleMessage(message []byte) {
	var msg Message
	if err := json.Unmarshal(message, &msg); err != nil {
		logrus.Errorf("message parse failed: %v", err)
		// Send error message back to client
		errorMsg := Message{
			Type:      MessageTypeError,
			Data:      ErrInvalidMessageData,
			Timestamp: time.Now().Unix(),
		}
		data, _ := json.Marshal(errorMsg)
		select {
		case c.Send <- data:
		default:
		}
		return
	}

	// Set sender ID
	msg.From = c.UserID

	// Handle message based on type
	switch msg.Type {
	case MessageTypePing:
		c.handlePing()
	case MessageTypeJoinGroup:
		c.handleJoinGroup(msg)
	case MessageTypeLeaveGroup:
		c.handleLeaveGroup(msg)
	case MessageTypeChat:
		c.handleChat(msg)
	case MessageTypeNotification:
		c.handleNotification(msg)
	case MessageTypeStatus:
		c.handleStatus(msg)
	default:
		logrus.Warnf("unknown message type: %s", msg.Type)
		// Send error message for unknown message type
		errorMsg := Message{
			Type:      MessageTypeError,
			Data:      ErrInvalidMessageType,
			Timestamp: time.Now().Unix(),
		}
		data, _ := json.Marshal(errorMsg)
		select {
		case c.Send <- data:
		default:
		}
	}
}

// handlePing handles ping message
func (c *Connection) handlePing() {
	c.mu.Lock()
	c.LastPing = time.Now()
	c.mu.Unlock()

	// Send pong response
	response := Message{
		Type:      MessageTypePong,
		Timestamp: time.Now().Unix(),
	}

	data, _ := json.Marshal(response)
	select {
	case c.Send <- data:
	default:
		logrus.Warnf("connection %s send buffer full", c.ID)
	}
}

// handleJoinGroup handles join group message
func (c *Connection) handleJoinGroup(msg Message) {
	groupName, ok := msg.Data.(string)
	if !ok {
		// Send error message
		errorMsg := Message{
			Type:      MessageTypeError,
			Data:      ErrInvalidMessageData,
			Timestamp: time.Now().Unix(),
		}
		data, _ := json.Marshal(errorMsg)
		select {
		case c.Send <- data:
		default:
		}
		return
	}

	c.mu.Lock()
	c.Groups[groupName] = true
	c.mu.Unlock()

	// Notify Hub to update group connection mapping
	c.Hub.mu.Lock()
	if c.Hub.groupConnections[groupName] == nil {
		c.Hub.groupConnections[groupName] = make(map[string]bool)
	}
	c.Hub.groupConnections[groupName][c.ID] = true
	c.Hub.mu.Unlock()

	// Send confirmation message
	response := Message{
		Type:      MessageTypeGroupJoined,
		Data:      map[string]interface{}{"group": groupName, "message": MsgGroupJoined},
		Timestamp: time.Now().Unix(),
	}

	data, _ := json.Marshal(response)
	select {
	case c.Send <- data:
	default:
		logrus.Warnf("connection %s send buffer full", c.ID)
	}

	logrus.Infof("user %s joined group %s", c.UserID, groupName)
}

// handleLeaveGroup handles leave group message
func (c *Connection) handleLeaveGroup(msg Message) {
	groupName, ok := msg.Data.(string)
	if !ok {
		// Send error message
		errorMsg := Message{
			Type:      MessageTypeError,
			Data:      ErrInvalidMessageData,
			Timestamp: time.Now().Unix(),
		}
		data, _ := json.Marshal(errorMsg)
		select {
		case c.Send <- data:
		default:
		}
		return
	}

	c.mu.Lock()
	delete(c.Groups, groupName)
	c.mu.Unlock()

	// Notify Hub to update group connection mapping
	c.Hub.mu.Lock()
	if c.Hub.groupConnections[groupName] != nil {
		delete(c.Hub.groupConnections[groupName], c.ID)
		if len(c.Hub.groupConnections[groupName]) == 0 {
			delete(c.Hub.groupConnections, groupName)
		}
	}
	c.Hub.mu.Unlock()

	// Send confirmation message
	response := Message{
		Type:      MessageTypeGroupLeft,
		Data:      map[string]interface{}{"group": groupName, "message": MsgGroupLeft},
		Timestamp: time.Now().Unix(),
	}

	data, _ := json.Marshal(response)
	select {
	case c.Send <- data:
	default:
		logrus.Warnf("connection %s send buffer full", c.ID)
	}

	logrus.Infof("user %s left group %s", c.UserID, groupName)
}

// handleChat handles chat message
func (c *Connection) handleChat(msg Message) {
	// Validate message data
	if _, ok := msg.Data.(map[string]interface{}); !ok {
		// Send error message
		errorMsg := Message{
			Type:      MessageTypeError,
			Data:      ErrInvalidMessageData,
			Timestamp: time.Now().Unix(),
		}
		data, _ := json.Marshal(errorMsg)
		select {
		case c.Send <- data:
		default:
		}
		return
	}

	// Check if there's a target user or group
	if msg.To == "" && msg.Group == "" {
		// Send error message
		errorMsg := Message{
			Type:      MessageTypeError,
			Data:      "chat message missing target",
			Timestamp: time.Now().Unix(),
		}
		data, _ := json.Marshal(errorMsg)
		select {
		case c.Send <- data:
		default:
		}
		return
	}

	// Broadcast message
	c.Hub.broadcast <- &msg
}

// handleNotification handles notification message
func (c *Connection) handleNotification(msg Message) {
	// Validate notification data
	if _, ok := msg.Data.(map[string]interface{}); !ok {
		// Send error message
		errorMsg := Message{
			Type:      MessageTypeError,
			Data:      ErrInvalidMessageData,
			Timestamp: time.Now().Unix(),
		}
		data, _ := json.Marshal(errorMsg)
		select {
		case c.Send <- data:
		default:
		}
		return
	}

	// Broadcast notification
	c.Hub.broadcast <- &msg
}

// handleStatus handles status message
func (c *Connection) handleStatus(msg Message) {
	// Update connection status
	if statusData, ok := msg.Data.(map[string]interface{}); ok {
		c.mu.Lock()
		for key, value := range statusData {
			c.Metadata[key] = value
		}
		c.mu.Unlock()
	}

	// Send status confirmation
	response := Message{
		Type:      MessageTypeStatusUpdated,
		Timestamp: time.Now().Unix(),
	}

	data, _ := json.Marshal(response)
	select {
	case c.Send <- data:
	default:
		logrus.Warnf("connection %s send buffer full", c.ID)
	}
}

// SendMessage sends a message to the current connection
func (c *Connection) SendMessage(message *Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	select {
	case c.Send <- data:
		return nil
	default:
		return fmt.Errorf(ErrSendBufferFull)
	}
}

// JoinGroup joins a group
func (c *Connection) JoinGroup(groupName string) {
	c.mu.Lock()
	c.Groups[groupName] = true
	c.mu.Unlock()

	// Notify Hub to update group connection mapping
	c.Hub.mu.Lock()
	if c.Hub.groupConnections[groupName] == nil {
		c.Hub.groupConnections[groupName] = make(map[string]bool)
	}
	c.Hub.groupConnections[groupName][c.ID] = true
	c.Hub.mu.Unlock()
}

// LeaveGroup leaves a group
func (c *Connection) LeaveGroup(groupName string) {
	c.mu.Lock()
	delete(c.Groups, groupName)
	c.mu.Unlock()

	// Notify Hub to update group connection mapping
	c.Hub.mu.Lock()
	if c.Hub.groupConnections[groupName] != nil {
		delete(c.Hub.groupConnections[groupName], c.ID)
		if len(c.Hub.groupConnections[groupName]) == 0 {
			delete(c.Hub.groupConnections, groupName)
		}
	}
	c.Hub.mu.Unlock()
}

// IsInGroup checks if the connection is in the specified group
func (c *Connection) IsInGroup(groupName string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Groups[groupName]
}

// GetGroups gets all groups the connection belongs to
func (c *Connection) GetGroups() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	groups := make([]string, 0, len(c.Groups))
	for group := range c.Groups {
		groups = append(groups, group)
	}
	return groups
}

// GetStatus gets the connection status
func (c *Connection) GetStatus() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Status
}

// SetStatus sets the connection status
func (c *Connection) SetStatus(status string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Status = status
}

// SendError sends an error message to the connection
func (c *Connection) SendError(errorMsg string) error {
	msg := Message{
		Type:      MessageTypeError,
		Data:      errorMsg,
		Timestamp: time.Now().Unix(),
	}
	return c.SendMessage(&msg)
}

// SendSuccess sends a success message to the connection
func (c *Connection) SendSuccess(successMsg string) error {
	msg := Message{
		Type:      MessageTypeSuccess,
		Data:      successMsg,
		Timestamp: time.Now().Unix(),
	}
	return c.SendMessage(&msg)
}

// Close closes the connection
func (c *Connection) Close() error {
	c.mu.Lock()
	c.IsAlive = false
	c.Status = ConnectionStatusDisconnected
	c.mu.Unlock()
	return c.Conn.Close()
}
