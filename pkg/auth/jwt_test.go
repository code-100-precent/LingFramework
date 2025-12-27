package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultJWTConfig(t *testing.T) {
	config := DefaultJWTConfig("test-secret-key")
	assert.NotNil(t, config)
	assert.Equal(t, "test-secret-key", config.SecretKey)
	assert.Equal(t, 15*time.Minute, config.AccessTokenTTL)
	assert.Equal(t, 7*24*time.Hour, config.RefreshTokenTTL)
	assert.Equal(t, "LingFramework", config.Issuer)
	assert.NotNil(t, config.SigningMethod)
}

func TestJWTManager_GenerateAndValidateToken(t *testing.T) {
	config := DefaultJWTConfig("test-secret-key")
	manager := NewJWTManager(config)

	// Generate access token
	token, err := manager.GenerateAccessToken(1, "testuser", []string{"admin", "user"}, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate token
	claims, err := manager.ValidateToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, uint(1), claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, []string{"admin", "user"}, claims.Roles)
}

func TestJWTManager_GenerateRefreshToken(t *testing.T) {
	config := DefaultJWTConfig("test-secret-key")
	manager := NewJWTManager(config)

	// Generate refresh token
	token, err := manager.GenerateRefreshToken(1, "testuser")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate token
	claims, err := manager.ValidateToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, uint(1), claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
}

func TestJWTManager_RefreshToken(t *testing.T) {
	config := DefaultJWTConfig("test-secret-key")
	manager := NewJWTManager(config)

	// Generate refresh token
	refreshToken, err := manager.GenerateRefreshToken(1, "testuser")
	assert.NoError(t, err)

	// Refresh access token
	newAccessToken, err := manager.RefreshToken(refreshToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, newAccessToken)

	// Validate new token
	claims, err := manager.ValidateToken(newAccessToken)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
}

func TestJWTManager_InvalidToken(t *testing.T) {
	config := DefaultJWTConfig("test-secret-key")
	manager := NewJWTManager(config)

	// Try to validate invalid token
	_, err := manager.ValidateToken("invalid-token")
	assert.Error(t, err)
}

func TestJWTManager_DifferentSecret(t *testing.T) {
	config1 := DefaultJWTConfig("secret-key-1")
	manager1 := NewJWTManager(config1)

	config2 := DefaultJWTConfig("secret-key-2")
	manager2 := NewJWTManager(config2)

	// Generate token with manager1
	token, err := manager1.GenerateAccessToken(1, "testuser", nil, nil)
	assert.NoError(t, err)

	// Try to validate with manager2 (different secret)
	_, err = manager2.ValidateToken(token)
	assert.Error(t, err)
}
