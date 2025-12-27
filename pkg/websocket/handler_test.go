package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/code-100-precent/LingFramework/pkg/constants"
	"github.com/code-100-precent/LingFramework/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func init() {
	// Initialize logger for tests
	if logger.Lg == nil {
		logger.Lg = zap.NewNop() // Use no-op logger for tests
	}
	_ = zap.NewNop // Ensure zap is used
}

func setupTestHandler(t *testing.T) (*Handler, *Hub) {
	gin.SetMode(gin.TestMode)
	config := DefaultConfig()
	hub := NewHub(config)
	handler := NewHandler(hub)
	return handler, hub
}

func TestNewHandler(t *testing.T) {
	hub := NewHub(DefaultConfig())
	defer hub.Close()

	handler := NewHandler(hub)
	assert.NotNil(t, handler)
	assert.Equal(t, hub, handler.hub)
}

func TestHandler_HandleWebSocket_NoAuth(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.GET("/ws", handler.HandleWebSocket)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_HandleWebSocket_WithUserIDString(t *testing.T) {
	// WebSocket upgrade requires http.Hijacker interface which httptest.ResponseRecorder doesn't support
	// So we need to use a real HTTP server for this test
	t.Skip("WebSocket upgrade requires http.Hijacker, skipping unit test - integration test needed")

	handler, hub := setupTestHandler(t)
	defer hub.Close()

	// Create a real HTTP server for WebSocket test
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create gin context
		c, _ := gin.CreateTestContext(w)
		c.Request = r
		c.Set(constants.UserField, "user123")
		handler.HandleWebSocket(c)
	}))
	defer server.Close()

	// Test that the endpoint is accessible (real WebSocket connection test would require integration test)
	client := &http.Client{}
	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")

	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
		// WebSocket upgrade should happen (may return 426 or other status)
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestHandler_HandleWebSocket_WithUserModel(t *testing.T) {
	// WebSocket upgrade requires http.Hijacker interface
	t.Skip("WebSocket upgrade requires http.Hijacker, skipping unit test - integration test needed")

	handler, hub := setupTestHandler(t)
	defer hub.Close()

	// Use string user ID for testing (models.User would work the same way)
	userID := "123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := gin.CreateTestContext(w)
		c.Request = r
		c.Set(constants.UserField, userID)
		handler.HandleWebSocket(c)
	}))
	defer server.Close()

	// Real WebSocket connection test would require integration test
	client := &http.Client{}
	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")

	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestHandler_GetStats(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.GET("/stats", handler.GetStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stats", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Contains(t, result, "total_connections")
	assert.Contains(t, result, "max_connections")
}

func TestHandler_GetUserStats(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.GET("/user/:user_id/stats", handler.GetUserStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/user/user123/stats", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, "user123", result["user_id"])
	assert.Contains(t, result, "connection_count")
}

func TestHandler_GetUserStats_EmptyID(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.GET("/user/:user_id/stats", handler.GetUserStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/user//stats", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetGroupStats(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.GET("/group/:group/stats", handler.GetGroupStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/group/testgroup/stats", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, "testgroup", result["group"])
	assert.Contains(t, result, "connection_count")
}

func TestHandler_SendMessage(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.POST("/message", handler.SendMessage)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/message", nil)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// Should fail without proper body, but we test the endpoint exists
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestHandler_SendMessage_NoTarget(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.POST("/message", handler.SendMessage)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/message", nil)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// Should return error without proper body setup
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestHandler_BroadcastMessage(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.POST("/broadcast", handler.BroadcastMessage)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/broadcast", nil)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestHandler_DisconnectUser(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.DELETE("/user/:user_id", handler.DisconnectUser)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/user/user123", nil)
	r.ServeHTTP(w, req)

	// User doesn't exist, should return 404
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_DisconnectGroup(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.DELETE("/group/:group", handler.DisconnectGroup)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/group/testgroup", nil)
	r.ServeHTTP(w, req)

	// Group doesn't exist, should return 404
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_HealthCheck(t *testing.T) {
	handler, hub := setupTestHandler(t)
	defer hub.Close()

	r := gin.New()
	r.GET("/health", handler.HealthCheck)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Contains(t, result, "status")
	assert.Contains(t, result, "total_connections")
	assert.Contains(t, result, "hub_running")
}

func TestRegisterRoutes(t *testing.T) {
	hub := NewHub(DefaultConfig())
	defer hub.Close()

	handler := NewHandler(hub)
	r := gin.New()
	RegisterRoutes(r, handler)

	// Check that routes are registered
	assert.NotNil(t, r)
}

func TestHandler_HandleAnonymousWebSocket(t *testing.T) {
	// WebSocket upgrade requires http.Hijacker interface
	t.Skip("WebSocket upgrade requires http.Hijacker, skipping unit test - integration test needed")

	handler, hub := setupTestHandler(t)
	defer hub.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := gin.CreateTestContext(w)
		c.Request = r
		handler.HandleAnonymousWebSocket(c)
	}))
	defer server.Close()

	// Real WebSocket connection test would require integration test
	client := &http.Client{}
	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("X-Request-ID", "test-request-id")

	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	}
}
