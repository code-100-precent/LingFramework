package websocket

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/code-100-precent/LingFramework/pkg/constants"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHandler_BroadcastMessage_InvalidJSON(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.POST("/broadcast", handler.BroadcastMessage)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/broadcast", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_BroadcastMessage_Valid(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.POST("/broadcast", handler.BroadcastMessage)

	msg := map[string]interface{}{
		"type": MessageTypeChat,
		"data": "test message",
	}
	body, _ := json.Marshal(msg)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/broadcast", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "broadcast message sent", response["message"])
}

func TestHandler_DisconnectUser_EmptyID(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.DELETE("/user/:user_id", handler.DisconnectUser)

	w := httptest.NewRecorder()
	// Use a route that doesn't have user_id param set properly
	req := httptest.NewRequest("DELETE", "/user/", nil)
	r.ServeHTTP(w, req)

	// Should handle gracefully
	assert.NotEqual(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_DisconnectUser_Valid(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	// Register a connection first
	conn, wsConn := createTestConnection(t, hub, "userToDisconnect")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	r := gin.New()
	r.DELETE("/user/:user_id", handler.DisconnectUser)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/user/userToDisconnect", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "user connections disconnected", response["message"])
	assert.Equal(t, float64(1), response["disconnected_count"])
}

func TestHandler_DisconnectGroup_EmptyName(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.DELETE("/group/:group", handler.DisconnectGroup)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/group/", nil)
	r.ServeHTTP(w, req)

	// Should handle gracefully
	assert.NotEqual(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_DisconnectGroup_Valid(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	// Register a connection and add to group
	conn, wsConn := createTestConnection(t, hub, "user1")
	if conn == nil {
		return
	}
	defer wsConn.Close()

	conn.JoinGroup("groupToDisconnect")
	hub.register <- conn
	time.Sleep(100 * time.Millisecond)

	r := gin.New()
	r.DELETE("/group/:group", handler.DisconnectGroup)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/group/groupToDisconnect", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "group connections disconnected", response["message"])
	assert.Equal(t, float64(1), response["disconnected_count"])
}

func TestHandler_HealthCheck_Warning(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	// Fill connections to > 90% of max
	hub.config.MaxConnections = 10
	for i := 0; i < 9; i++ {
		conn, wsConn := createTestConnection(t, hub, "user"+string(rune(i)))
		if conn == nil {
			continue
		}
		defer wsConn.Close()
		hub.register <- conn
	}
	time.Sleep(100 * time.Millisecond)

	r := gin.New()
	r.GET("/health", handler.HealthCheck)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "warning", response["status"])
}

func TestHandler_HealthCheck_Unhealthy(t *testing.T) {
	handler, hub := setupTestHandler(t)
	hub.Close() // Close the hub to make it unhealthy

	r := gin.New()
	r.GET("/health", handler.HealthCheck)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "unhealthy", response["status"])
}

func TestHandler_SendMessage_WithTo(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.POST("/message", handler.SendMessage)

	msg := map[string]interface{}{
		"type": MessageTypeChat,
		"data": "test message",
		"to":   "user123",
	}
	body, _ := json.Marshal(msg)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/message", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_SendMessage_WithGroup(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.POST("/message", handler.SendMessage)

	msg := map[string]interface{}{
		"type":  MessageTypeChat,
		"data":  "test message",
		"group": "group123",
	}
	body, _ := json.Marshal(msg)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/message", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_SendMessage_InvalidJSON(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.POST("/message", handler.SendMessage)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/message", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_HandleWebSocket_InvalidUserIDType(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.GET("/ws", func(c *gin.Context) {
		// Set an invalid user ID type
		c.Set(constants.UserField, 12345) // int instead of string or *User
		handler.HandleWebSocket(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_HandleWebSocket_EmptyUserID(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.GET("/ws", func(c *gin.Context) {
		c.Set(constants.UserField, "") // Empty string
		handler.HandleWebSocket(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_HandleAnonymousWebSocket_WithRequestID(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.GET("/ws/anonymous", handler.HandleAnonymousWebSocket)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws/anonymous", nil)
	req.Header.Set("X-Request-ID", "test-request-123")
	r.ServeHTTP(w, req)

	// Should attempt to upgrade (might fail in test environment)
	assert.NotEqual(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_HandleAnonymousWebSocket_WithRealIP(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.GET("/ws/anonymous", handler.HandleAnonymousWebSocket)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws/anonymous", nil)
	req.Header.Set("X-Real-IP", "192.168.1.1")
	r.ServeHTTP(w, req)

	// Should attempt to upgrade (might fail in test environment)
	assert.NotEqual(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetGroupStats_EmptyGroup(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.GET("/group/:group/stats", handler.GetGroupStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/group//stats", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
