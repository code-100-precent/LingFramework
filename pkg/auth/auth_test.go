package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAuthManager_JWT(t *testing.T) {
	config := &AuthConfig{
		AuthType:       AuthTypeJWT,
		PermissionType: PermissionTypeRBAC,
		JWTSecretKey:   "test-secret-key",
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, AuthTypeJWT, manager.authType)
	assert.Equal(t, PermissionTypeRBAC, manager.permissionType)
	assert.NotNil(t, manager.jwtManager)
	assert.NotNil(t, manager.rbac)
}

func TestNewAuthManager_JWT_NoSecret(t *testing.T) {
	config := &AuthConfig{
		AuthType:       AuthTypeJWT,
		PermissionType: PermissionTypeRBAC,
		JWTSecretKey:   "", // Missing secret
	}

	_, err := NewAuthManager(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWTSecretKey is required")
}

func TestNewAuthManager_OAuth2(t *testing.T) {
	config := &AuthConfig{
		AuthType:              AuthTypeOAuth2,
		PermissionType:        PermissionTypeABAC,
		OAuth2AccessTokenTTL:  15 * time.Minute,
		OAuth2RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, AuthTypeOAuth2, manager.authType)
	assert.Equal(t, PermissionTypeABAC, manager.permissionType)
	assert.NotNil(t, manager.oauth2Server)
	assert.NotNil(t, manager.abac)
}

func TestNewAuthManager_NilConfig(t *testing.T) {
	_, err := NewAuthManager(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestNewAuthManager_DefaultTTL(t *testing.T) {
	config := &AuthConfig{
		AuthType:       AuthTypeJWT,
		PermissionType: PermissionTypeRBAC,
		JWTSecretKey:   "test-secret",
		// No TTL set, should use defaults
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	jwtManager := manager.GetJWTManager()
	assert.NotNil(t, jwtManager)
	assert.Equal(t, 15*time.Minute, jwtManager.config.AccessTokenTTL)
	assert.Equal(t, 7*24*time.Hour, jwtManager.config.RefreshTokenTTL)
}

func TestAuthManager_GenerateToken_JWT(t *testing.T) {
	config := &AuthConfig{
		AuthType:       AuthTypeJWT,
		PermissionType: PermissionTypeRBAC,
		JWTSecretKey:   "test-secret",
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)

	accessToken, refreshToken, err := manager.GenerateToken(1, "testuser", []string{"admin"}, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
}

func TestAuthManager_GenerateToken_OAuth2(t *testing.T) {
	config := &AuthConfig{
		AuthType:       AuthTypeOAuth2,
		PermissionType: PermissionTypeRBAC,
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)

	_, _, err = manager.GenerateToken(1, "testuser", []string{"admin"}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authorization code flow")
}

func TestAuthManager_ValidateToken_JWT(t *testing.T) {
	config := &AuthConfig{
		AuthType:       AuthTypeJWT,
		PermissionType: PermissionTypeRBAC,
		JWTSecretKey:   "test-secret",
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)

	accessToken, _, err := manager.GenerateToken(1, "testuser", []string{"admin"}, nil)
	assert.NoError(t, err)

	userID, username, roles, err := manager.ValidateToken(accessToken)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), userID)
	assert.Equal(t, "testuser", username)
	assert.Equal(t, []string{"admin"}, roles)
}

func TestAuthManager_ValidateToken_Invalid(t *testing.T) {
	config := &AuthConfig{
		AuthType:       AuthTypeJWT,
		PermissionType: PermissionTypeRBAC,
		JWTSecretKey:   "test-secret",
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)

	_, _, _, err = manager.ValidateToken("invalid-token")
	assert.Error(t, err)
}

func TestAuthManager_RefreshToken_JWT(t *testing.T) {
	config := &AuthConfig{
		AuthType:       AuthTypeJWT,
		PermissionType: PermissionTypeRBAC,
		JWTSecretKey:   "test-secret",
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)

	_, refreshToken, err := manager.GenerateToken(1, "testuser", []string{"admin"}, nil)
	assert.NoError(t, err)

	newAccessToken, err := manager.RefreshToken(refreshToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, newAccessToken)

	// Validate new token
	userID, username, _, err := manager.ValidateToken(newAccessToken)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), userID)
	assert.Equal(t, "testuser", username)
}

func TestAuthManager_CheckPermission_RBAC(t *testing.T) {
	config := &AuthConfig{
		AuthType:       AuthTypeJWT,
		PermissionType: PermissionTypeRBAC,
		JWTSecretKey:   "test-secret",
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)

	rbac := manager.GetRBAC()
	rbac.AddRole("admin", []Permission{
		{Resource: "user", Action: "read"},
	})
	rbac.AssignRole(1, "admin")

	err = manager.CheckPermission(1, "user", "read", nil)
	assert.NoError(t, err)

	err = manager.CheckPermission(1, "user", "delete", nil)
	assert.Error(t, err)
}

func TestAuthManager_CheckPermission_ABAC(t *testing.T) {
	config := &AuthConfig{
		AuthType:       AuthTypeJWT,
		PermissionType: PermissionTypeABAC,
		JWTSecretKey:   "test-secret",
		UserLoader: func(userID uint) (UserInfo, error) {
			return UserInfo{
				ID:       userID,
				Username: "testuser",
				Roles:    []string{"admin"},
				Attrs:    map[string]interface{}{"role": "admin"},
			}, nil
		},
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)

	abac := manager.GetABAC()
	policy := &Policy{
		ID:        "policy-1",
		Subjects:  []Attribute{{Key: "role", Value: "admin"}},
		Resources: []Attribute{{Key: "type", Value: "article"}},
		Actions:   []string{"read"},
		Effect:    "allow",
	}
	abac.AddPolicy(policy)

	resourceAttrs := map[string]interface{}{
		"type": "article",
	}

	err = manager.CheckPermission(1, "article", "read", resourceAttrs)
	assert.NoError(t, err)

	err = manager.CheckPermission(1, "article", "delete", resourceAttrs)
	assert.Error(t, err)
}

func TestAuthManager_CheckPermission_ABAC_NoUserLoader(t *testing.T) {
	config := &AuthConfig{
		AuthType:       AuthTypeJWT,
		PermissionType: PermissionTypeABAC,
		JWTSecretKey:   "test-secret",
		// No UserLoader
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)

	abac := manager.GetABAC()
	policy := &Policy{
		ID:        "policy-1",
		Subjects:  []Attribute{{Key: "role", Value: "admin"}},
		Resources: []Attribute{{Key: "type", Value: "article"}},
		Actions:   []string{"read"},
		Effect:    "allow",
	}
	abac.AddPolicy(policy)

	resourceAttrs := map[string]interface{}{
		"type": "article",
	}

	// Should fail because no user attributes available
	err = manager.CheckPermission(1, "article", "read", resourceAttrs)
	assert.Error(t, err)
}

func TestAuthManager_ValidateToken_OAuth2(t *testing.T) {
	config := &AuthConfig{
		AuthType:              AuthTypeOAuth2,
		PermissionType:        PermissionTypeRBAC,
		OAuth2AccessTokenTTL:  15 * time.Minute,
		OAuth2RefreshTokenTTL: 7 * 24 * time.Hour,
		UserLoader: func(userID uint) (UserInfo, error) {
			return UserInfo{
				ID:       userID,
				Username: "testuser",
				Roles:    []string{"admin"},
			}, nil
		},
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)

	oauth2Server := manager.GetOAuth2Server()
	oauth2Server.RegisterClient("client-1", "secret-1", "http://example.com/callback", []string{"read"})

	code, err := oauth2Server.GenerateAuthorizationCode("client-1", 1, "http://example.com/callback", []string{"read"})
	assert.NoError(t, err)

	tokenInfo, err := oauth2Server.ExchangeCode(code, "client-1", "secret-1")
	assert.NoError(t, err)

	userID, username, roles, err := manager.ValidateToken(tokenInfo.AccessToken)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), userID)
	assert.Equal(t, "testuser", username)
	assert.Equal(t, []string{"admin"}, roles)
}

func TestAuthManager_ValidateToken_OAuth2_NoUserLoader(t *testing.T) {
	config := &AuthConfig{
		AuthType:              AuthTypeOAuth2,
		PermissionType:        PermissionTypeRBAC,
		OAuth2AccessTokenTTL:  15 * time.Minute,
		OAuth2RefreshTokenTTL: 7 * 24 * time.Hour,
		// No UserLoader
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)

	oauth2Server := manager.GetOAuth2Server()
	oauth2Server.RegisterClient("client-1", "secret-1", "http://example.com/callback", []string{"read"})

	code, err := oauth2Server.GenerateAuthorizationCode("client-1", 1, "http://example.com/callback", []string{"read"})
	assert.NoError(t, err)

	tokenInfo, err := oauth2Server.ExchangeCode(code, "client-1", "secret-1")
	assert.NoError(t, err)

	userID, username, roles, err := manager.ValidateToken(tokenInfo.AccessToken)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), userID)
	assert.Equal(t, "", username) // No user loader, so empty
	assert.Nil(t, roles)
}

func TestAuthManager_RefreshToken_OAuth2(t *testing.T) {
	config := &AuthConfig{
		AuthType:              AuthTypeOAuth2,
		PermissionType:        PermissionTypeRBAC,
		OAuth2AccessTokenTTL:  15 * time.Minute,
		OAuth2RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)

	oauth2Server := manager.GetOAuth2Server()
	oauth2Server.RegisterClient("client-1", "secret-1", "http://example.com/callback", []string{"read"})

	code, err := oauth2Server.GenerateAuthorizationCode("client-1", 1, "http://example.com/callback", []string{"read"})
	assert.NoError(t, err)

	tokenInfo, err := oauth2Server.ExchangeCode(code, "client-1", "secret-1")
	assert.NoError(t, err)

	oldAccessToken := tokenInfo.AccessToken
	newAccessToken, err := manager.RefreshToken(tokenInfo.RefreshToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, newAccessToken)
	// New token should be different from old token
	assert.NotEqual(t, oldAccessToken, newAccessToken)

	// Verify old token is invalid
	_, err = oauth2Server.ValidateToken(oldAccessToken)
	assert.Error(t, err)

	// Verify new token is valid
	_, err = oauth2Server.ValidateToken(newAccessToken)
	assert.NoError(t, err)
}

func TestAuthManager_CheckPermission_NoManager(t *testing.T) {
	// Test with an invalid permission type by directly creating manager
	manager := &AuthManager{
		authType:       AuthTypeJWT,
		permissionType: "", // Invalid
		jwtManager:     NewJWTManager(DefaultJWTConfig("test")),
	}

	err := manager.CheckPermission(1, "resource", "action", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no permission manager")
}

func TestAuthManager_HasPermission(t *testing.T) {
	config := &AuthConfig{
		AuthType:       AuthTypeJWT,
		PermissionType: PermissionTypeRBAC,
		JWTSecretKey:   "test-secret",
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)

	rbac := manager.GetRBAC()
	rbac.AddRole("admin", []Permission{
		{Resource: "user", Action: "read"},
	})
	rbac.AssignRole(1, "admin")

	assert.True(t, manager.HasPermission(1, "user", "read", nil))
	assert.False(t, manager.HasPermission(1, "user", "delete", nil))
}

func TestAuthManager_GetRBAC(t *testing.T) {
	config := &AuthConfig{
		AuthType:       AuthTypeJWT,
		PermissionType: PermissionTypeRBAC,
		JWTSecretKey:   "test-secret",
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)

	rbac := manager.GetRBAC()
	assert.NotNil(t, rbac)

	// Manager without RBAC should return nil
	config2 := &AuthConfig{
		AuthType:       AuthTypeJWT,
		PermissionType: PermissionTypeABAC,
		JWTSecretKey:   "test-secret",
	}
	manager2, err := NewAuthManager(config2)
	assert.NoError(t, err)
	assert.Nil(t, manager2.GetRBAC())
	assert.NotNil(t, manager2.GetABAC())
}

func TestAuthManager_GetABAC(t *testing.T) {
	config := &AuthConfig{
		AuthType:       AuthTypeJWT,
		PermissionType: PermissionTypeABAC,
		JWTSecretKey:   "test-secret",
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)

	abac := manager.GetABAC()
	assert.NotNil(t, abac)
}

func TestAuthManager_GetOAuth2Server(t *testing.T) {
	config := &AuthConfig{
		AuthType:              AuthTypeOAuth2,
		OAuth2AccessTokenTTL:  15 * time.Minute,
		OAuth2RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)

	server := manager.GetOAuth2Server()
	assert.NotNil(t, server)
}

func TestAuthManager_GetJWTManager(t *testing.T) {
	config := &AuthConfig{
		AuthType:     AuthTypeJWT,
		JWTSecretKey: "test-secret",
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)

	jwtManager := manager.GetJWTManager()
	assert.NotNil(t, jwtManager)
}

func TestAuthManager_SetUserLoader(t *testing.T) {
	config := &AuthConfig{
		AuthType:     AuthTypeJWT,
		JWTSecretKey: "test-secret",
		UserLoader: func(userID uint) (UserInfo, error) {
			return UserInfo{
				ID:       userID,
				Username: "testuser",
				Roles:    []string{"admin"},
			}, nil
		},
	}

	manager, err := NewAuthManager(config)
	assert.NoError(t, err)
	assert.NotNil(t, manager.loadUserInfo)

	userInfo, err := manager.loadUserInfo(1)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), userInfo.ID)
	assert.Equal(t, "testuser", userInfo.Username)
}
