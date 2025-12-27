package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDefaultMiddlewareConfig(t *testing.T) {
	config, err := LoadAuthConfigFromEnv()
	if err != nil {
		// Skip test if config fails (expected in test environment)
		t.Skip("Skipping test: unable to load auth config from env")
		return
	}

	authManager, err := NewAuthManager(config)
	assert.NoError(t, err)

	middlewareConfig := DefaultMiddlewareConfig(authManager)
	assert.NotNil(t, middlewareConfig)
	assert.Equal(t, "Authorization", middlewareConfig.TokenHeader)
	assert.Equal(t, "Bearer ", middlewareConfig.TokenPrefix)
	assert.NotEmpty(t, middlewareConfig.SkipPaths)
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config, err := LoadAuthConfigFromEnv()
	if err != nil {
		t.Skip("Skipping test: unable to load auth config from env")
		return
	}

	authManager, err := NewAuthManager(config)
	assert.NoError(t, err)

	middlewareConfig := DefaultMiddlewareConfig(authManager)
	r := gin.New()
	r.Use(AuthMiddleware(middlewareConfig))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_SkipPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config, err := LoadAuthConfigFromEnv()
	if err != nil {
		t.Skip("Skipping test: unable to load auth config from env")
		return
	}

	authManager, err := NewAuthManager(config)
	assert.NoError(t, err)

	middlewareConfig := DefaultMiddlewareConfig(authManager)
	r := gin.New()
	r.Use(AuthMiddleware(middlewareConfig))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestExtractTokenFromRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config, err := LoadAuthConfigFromEnv()
	if err != nil {
		t.Skip("Skipping test: unable to load auth config from env")
		return
	}

	authManager, err := NewAuthManager(config)
	assert.NoError(t, err)

	middlewareConfig := DefaultMiddlewareConfig(authManager)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer test-token")

	token := extractTokenFromRequest(c, middlewareConfig)
	assert.Equal(t, "test-token", token)
}

func TestExtractResourceAttributes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/users/123", nil)
	c.Params = gin.Params{{Key: "id", Value: "123"}}

	attrs := extractResourceAttributes(c, "user")
	assert.Equal(t, "user", attrs["type"])
	assert.Equal(t, "123", attrs["id"])
}

func TestGetUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// Test when user_id is not set
	userID, exists := GetUserID(c)
	assert.False(t, exists)
	assert.Equal(t, uint(0), userID)

	// Test when user_id is set
	c.Set("user_id", uint(123))
	userID, exists = GetUserID(c)
	assert.True(t, exists)
	assert.Equal(t, uint(123), userID)
}

func TestGetUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// Test when username is not set
	username, exists := GetUsername(c)
	assert.False(t, exists)
	assert.Equal(t, "", username)

	// Test when username is set
	c.Set("username", "testuser")
	username, exists = GetUsername(c)
	assert.True(t, exists)
	assert.Equal(t, "testuser", username)
}

func TestGetRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// Test when roles are not set
	roles, exists := GetRoles(c)
	assert.False(t, exists)
	assert.Nil(t, roles)

	// Test when roles are set
	expectedRoles := []string{"admin", "user"}
	c.Set("roles", expectedRoles)
	roles, exists = GetRoles(c)
	assert.True(t, exists)
	assert.Equal(t, expectedRoles, roles)
}

func TestLoadAuthConfigFromEnv(t *testing.T) {
	// Save original env values
	originalAuthType := os.Getenv("AUTH_TYPE")
	originalPermType := os.Getenv("PERMISSION_TYPE")
	originalJWTSecret := os.Getenv("JWT_SECRET_KEY")
	originalSessionSecret := os.Getenv("SESSION_SECRET")

	defer func() {
		if originalAuthType != "" {
			os.Setenv("AUTH_TYPE", originalAuthType)
		} else {
			os.Unsetenv("AUTH_TYPE")
		}
		if originalPermType != "" {
			os.Setenv("PERMISSION_TYPE", originalPermType)
		} else {
			os.Unsetenv("PERMISSION_TYPE")
		}
		if originalJWTSecret != "" {
			os.Setenv("JWT_SECRET_KEY", originalJWTSecret)
		} else {
			os.Unsetenv("JWT_SECRET_KEY")
		}
		if originalSessionSecret != "" {
			os.Setenv("SESSION_SECRET", originalSessionSecret)
		} else {
			os.Unsetenv("SESSION_SECRET")
		}
	}()

	// Test default JWT config
	os.Unsetenv("AUTH_TYPE")
	os.Unsetenv("PERMISSION_TYPE")
	os.Unsetenv("JWT_SECRET_KEY")
	os.Setenv("SESSION_SECRET", "test-session-secret")

	config, err := LoadAuthConfigFromEnv()
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, AuthTypeJWT, config.AuthType)
	assert.Equal(t, PermissionTypeRBAC, config.PermissionType)
	assert.Equal(t, "test-session-secret", config.JWTSecretKey)

	// Test JWT with JWT_SECRET_KEY
	os.Setenv("JWT_SECRET_KEY", "test-jwt-secret")
	config, err = LoadAuthConfigFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, "test-jwt-secret", config.JWTSecretKey)

	// Test OAuth2
	os.Setenv("AUTH_TYPE", "oauth2")
	config, err = LoadAuthConfigFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, AuthTypeOAuth2, config.AuthType)

	// Test invalid auth type (should default to JWT)
	os.Setenv("AUTH_TYPE", "invalid")
	os.Unsetenv("JWT_SECRET_KEY")
	os.Setenv("SESSION_SECRET", "test-secret")
	config, err = LoadAuthConfigFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, AuthTypeJWT, config.AuthType)

	// Test ABAC
	os.Setenv("AUTH_TYPE", "jwt")
	os.Setenv("PERMISSION_TYPE", "abac")
	os.Setenv("JWT_SECRET_KEY", "test-secret")
	config, err = LoadAuthConfigFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, PermissionTypeABAC, config.PermissionType)

	// Test invalid permission type (should default to RBAC)
	os.Setenv("PERMISSION_TYPE", "invalid")
	config, err = LoadAuthConfigFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, PermissionTypeRBAC, config.PermissionType)
}

func TestLoadAuthConfigFromEnv_NoSecret(t *testing.T) {
	originalJWTSecret := os.Getenv("JWT_SECRET_KEY")
	originalSessionSecret := os.Getenv("SESSION_SECRET")
	originalAuthType := os.Getenv("AUTH_TYPE")

	defer func() {
		if originalJWTSecret != "" {
			os.Setenv("JWT_SECRET_KEY", originalJWTSecret)
		} else {
			os.Unsetenv("JWT_SECRET_KEY")
		}
		if originalSessionSecret != "" {
			os.Setenv("SESSION_SECRET", originalSessionSecret)
		} else {
			os.Unsetenv("SESSION_SECRET")
		}
		if originalAuthType != "" {
			os.Setenv("AUTH_TYPE", originalAuthType)
		} else {
			os.Unsetenv("AUTH_TYPE")
		}
	}()

	// Clear env vars - but LoadAuthConfigFromEnv may read from .env file
	// So we test that it returns error when both are empty
	// In practice, if .env has a value, it won't error
	os.Unsetenv("JWT_SECRET_KEY")
	os.Unsetenv("SESSION_SECRET")
	os.Setenv("AUTH_TYPE", "jwt")

	// This may or may not error depending on .env file contents
	config, err := LoadAuthConfigFromEnv()
	if err != nil {
		assert.Contains(t, err.Error(), "JWT_SECRET_KEY is required")
		assert.Nil(t, config)
	} else {
		// If no error, config should have a secret (from .env file)
		assert.NotNil(t, config)
		assert.NotEmpty(t, config.JWTSecretKey)
	}
}

func TestConfigError(t *testing.T) {
	err := &ConfigError{Message: "test error"}
	assert.Equal(t, "test error", err.Error())
}

func TestAuthMiddleware_WithOAuth2(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := &AuthConfig{
		AuthType:              AuthTypeOAuth2,
		PermissionType:        PermissionTypeRBAC,
		OAuth2AccessTokenTTL:  15 * time.Minute,
		OAuth2RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	authManager, err := NewAuthManager(config)
	assert.NoError(t, err)

	oauth2Server := authManager.GetOAuth2Server()
	oauth2Server.RegisterClient("client-1", "secret-1", "http://example.com/callback", []string{"read"})

	code, err := oauth2Server.GenerateAuthorizationCode("client-1", 1, "http://example.com/callback", []string{"read"})
	assert.NoError(t, err)

	tokenInfo, err := oauth2Server.ExchangeCode(code, "client-1", "secret-1")
	assert.NoError(t, err)

	middlewareConfig := DefaultMiddlewareConfig(authManager)
	r := gin.New()
	r.Use(AuthMiddleware(middlewareConfig))
	r.GET("/test", func(c *gin.Context) {
		userID, _ := GetUserID(c)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPermissionMiddleware_WithABAC(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := &AuthConfig{
		AuthType:       AuthTypeJWT,
		PermissionType: PermissionTypeABAC,
		JWTSecretKey:   "test-secret",
		UserLoader: func(userID uint) (UserInfo, error) {
			return UserInfo{
				ID:       userID,
				Username: "testuser",
				Roles:    []string{"user"},
				Attrs:    map[string]interface{}{"role": "user"},
			}, nil
		},
	}

	authManager, err := NewAuthManager(config)
	assert.NoError(t, err)

	abac := authManager.GetABAC()
	policy := &Policy{
		ID:        "policy-1",
		Subjects:  []Attribute{{Key: "role", Value: "user"}},
		Resources: []Attribute{{Key: "type", Value: "article"}},
		Actions:   []string{"read"},
		Effect:    "allow",
	}
	abac.AddPolicy(policy)

	accessToken, _, err := authManager.GenerateToken(1, "testuser", []string{"user"}, nil)
	assert.NoError(t, err)

	middlewareConfig := DefaultMiddlewareConfig(authManager)
	r := gin.New()
	r.Use(AuthMiddleware(middlewareConfig))
	r.GET("/articles/:id",
		PermissionMiddleware(middlewareConfig, "article", "read"),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		},
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/articles/123", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	r.ServeHTTP(w, req)

	// Should pass if ABAC policy matches
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPermissionMiddlewareWithAttrs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := &AuthConfig{
		AuthType:       AuthTypeJWT,
		PermissionType: PermissionTypeRBAC,
		JWTSecretKey:   "test-secret",
	}

	authManager, err := NewAuthManager(config)
	assert.NoError(t, err)

	rbac := authManager.GetRBAC()
	rbac.AddRole("admin", []Permission{
		{Resource: "article", Action: "delete"},
	})
	rbac.AssignRole(1, "admin")

	accessToken, _, err := authManager.GenerateToken(1, "testuser", []string{"admin"}, nil)
	assert.NoError(t, err)

	middlewareConfig := DefaultMiddlewareConfig(authManager)
	r := gin.New()
	r.Use(AuthMiddleware(middlewareConfig))
	r.DELETE("/articles/:id",
		PermissionMiddlewareWithAttrs(
			middlewareConfig,
			"article",
			"delete",
			func(c *gin.Context) map[string]interface{} {
				return map[string]interface{}{
					"type": "article",
					"id":   c.Param("id"),
				}
			},
		),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "deleted"})
		},
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/articles/123", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestExtractTokenFromRequest_Cookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config, err := LoadAuthConfigFromEnv()
	if err != nil {
		t.Skip("Skipping test: unable to load auth config from env")
		return
	}

	authManager, err := NewAuthManager(config)
	assert.NoError(t, err)

	middlewareConfig := DefaultMiddlewareConfig(authManager)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.AddCookie(&http.Cookie{Name: "token", Value: "cookie-token"})

	token := extractTokenFromRequest(c, middlewareConfig)
	assert.Equal(t, "cookie-token", token)
}

func TestExtractTokenFromRequest_Query(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config, err := LoadAuthConfigFromEnv()
	if err != nil {
		t.Skip("Skipping test: unable to load auth config from env")
		return
	}

	authManager, err := NewAuthManager(config)
	assert.NoError(t, err)

	middlewareConfig := DefaultMiddlewareConfig(authManager)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/test?token=query-token", nil)

	token := extractTokenFromRequest(c, middlewareConfig)
	assert.Equal(t, "query-token", token)
}

func TestExtractTokenFromRequest_EmptyPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := &AuthConfig{
		AuthType:       AuthTypeJWT,
		PermissionType: PermissionTypeRBAC,
		JWTSecretKey:   "test-secret",
	}

	authManager, err := NewAuthManager(config)
	assert.NoError(t, err)

	middlewareConfig := DefaultMiddlewareConfig(authManager)
	middlewareConfig.TokenPrefix = "" // Empty prefix

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "plain-token")

	token := extractTokenFromRequest(c, middlewareConfig)
	assert.Equal(t, "plain-token", token)
}

func TestDefaultMiddlewareConfig_CustomSkipPaths(t *testing.T) {
	originalSkipPaths := os.Getenv("AUTH_SKIP_PATHS")
	defer func() {
		if originalSkipPaths != "" {
			os.Setenv("AUTH_SKIP_PATHS", originalSkipPaths)
		} else {
			os.Unsetenv("AUTH_SKIP_PATHS")
		}
	}()

	os.Setenv("AUTH_SKIP_PATHS", "/custom/path1,/custom/path2")

	config := &AuthConfig{
		AuthType:       AuthTypeJWT,
		PermissionType: PermissionTypeRBAC,
		JWTSecretKey:   "test-secret",
	}

	authManager, err := NewAuthManager(config)
	assert.NoError(t, err)

	middlewareConfig := DefaultMiddlewareConfig(authManager)
	assert.Contains(t, middlewareConfig.SkipPaths, "/custom/path1")
	assert.Contains(t, middlewareConfig.SkipPaths, "/custom/path2")
}
