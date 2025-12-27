package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AuthType represents authentication type
type AuthType string

const (
	AuthTypeJWT    AuthType = "jwt"
	AuthTypeOAuth2 AuthType = "oauth2"
)

// PermissionType represents permission management type
type PermissionType string

const (
	PermissionTypeRBAC PermissionType = "rbac"
	PermissionTypeABAC PermissionType = "abac"
)

// AuthManager represents the main authentication manager
type AuthManager struct {
	jwtManager     *JWTManager
	oauth2Server   *OAuth2Server
	rbac           *RBAC
	abac           *ABAC
	authType       AuthType
	permissionType PermissionType
	loadUserInfo   func(userID uint) (UserInfo, error)
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	AuthType       AuthType
	PermissionType PermissionType

	// JWT configuration
	JWTSecretKey       string
	JWTAccessTokenTTL  time.Duration
	JWTRefreshTokenTTL time.Duration
	JWTIssuer          string

	// OAuth2 server configuration
	OAuth2AccessTokenTTL  time.Duration
	OAuth2RefreshTokenTTL time.Duration

	// User loader function (loads user info by ID)
	UserLoader func(userID uint) (UserInfo, error)
}

// UserInfo represents user information
type UserInfo struct {
	ID       uint
	Username string
	Roles    []string
	Attrs    map[string]interface{} // Attributes for ABAC
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(config *AuthConfig) (*AuthManager, error) {
	if config == nil {
		return nil, errors.New("AuthConfig cannot be nil")
	}

	manager := &AuthManager{
		authType:       config.AuthType,
		permissionType: config.PermissionType,
		loadUserInfo:   config.UserLoader,
	}

	// Initialize JWT manager if JWT is selected
	if config.AuthType == AuthTypeJWT {
		if config.JWTSecretKey == "" {
			return nil, errors.New("JWTSecretKey is required when using JWT")
		}

		jwtConfig := &JWTConfig{
			SecretKey:       config.JWTSecretKey,
			AccessTokenTTL:  config.JWTAccessTokenTTL,
			RefreshTokenTTL: config.JWTRefreshTokenTTL,
			Issuer:          config.JWTIssuer,
			SigningMethod:   jwt.SigningMethodHS256,
		}

		if jwtConfig.AccessTokenTTL == 0 {
			jwtConfig.AccessTokenTTL = 15 * time.Minute
		}
		if jwtConfig.RefreshTokenTTL == 0 {
			jwtConfig.RefreshTokenTTL = 7 * 24 * time.Hour
		}
		if jwtConfig.Issuer == "" {
			jwtConfig.Issuer = "LingFramework"
		}

		manager.jwtManager = NewJWTManager(jwtConfig)
	}

	// Initialize OAuth2 server if OAuth2 is selected
	if config.AuthType == AuthTypeOAuth2 {
		accessTTL := config.OAuth2AccessTokenTTL
		if accessTTL == 0 {
			accessTTL = 15 * time.Minute
		}
		refreshTTL := config.OAuth2RefreshTokenTTL
		if refreshTTL == 0 {
			refreshTTL = 7 * 24 * time.Hour
		}

		manager.oauth2Server = NewOAuth2Server(accessTTL, refreshTTL)
	}

	// Initialize RBAC if RBAC is selected
	if config.PermissionType == PermissionTypeRBAC {
		manager.rbac = NewRBAC()
	}

	// Initialize ABAC if ABAC is selected
	if config.PermissionType == PermissionTypeABAC {
		manager.abac = NewABAC()
	}

	return manager, nil
}

// GenerateToken generates a token based on auth type
func (m *AuthManager) GenerateToken(userID uint, username string, roles []string, extra map[string]interface{}) (accessToken, refreshToken string, err error) {
	switch m.authType {
	case AuthTypeJWT:
		accessToken, err = m.jwtManager.GenerateAccessToken(userID, username, roles, extra)
		if err != nil {
			return "", "", err
		}
		refreshToken, err = m.jwtManager.GenerateRefreshToken(userID, username)
		if err != nil {
			return "", "", err
		}
		return accessToken, refreshToken, nil

	case AuthTypeOAuth2:
		// For OAuth2, tokens are generated through the authorization flow
		return "", "", errors.New("OAuth2 tokens must be generated through authorization code flow")

	default:
		return "", "", fmt.Errorf("unsupported auth type: %s", m.authType)
	}
}

// ValidateToken validates a token based on auth type
func (m *AuthManager) ValidateToken(token string) (userID uint, username string, roles []string, err error) {
	switch m.authType {
	case AuthTypeJWT:
		claims, err := m.jwtManager.ValidateToken(token)
		if err != nil {
			return 0, "", nil, err
		}
		return claims.UserID, claims.Username, claims.Roles, nil

	case AuthTypeOAuth2:
		tokenInfo, err := m.oauth2Server.ValidateToken(token)
		if err != nil {
			return 0, "", nil, err
		}
		// Load user info to get username and roles
		if m.loadUserInfo != nil {
			userInfo, err := m.loadUserInfo(tokenInfo.UserID)
			if err == nil {
				return tokenInfo.UserID, userInfo.Username, userInfo.Roles, nil
			}
		}
		return tokenInfo.UserID, "", nil, nil

	default:
		return 0, "", nil, fmt.Errorf("unsupported auth type: %s", m.authType)
	}
}

// RefreshToken refreshes an access token
func (m *AuthManager) RefreshToken(refreshToken string) (accessToken string, err error) {
	switch m.authType {
	case AuthTypeJWT:
		return m.jwtManager.RefreshToken(refreshToken)

	case AuthTypeOAuth2:
		tokenInfo, err := m.oauth2Server.RefreshAccessToken(refreshToken)
		if err != nil {
			return "", err
		}
		return tokenInfo.AccessToken, nil

	default:
		return "", fmt.Errorf("unsupported auth type: %s", m.authType)
	}
}

// CheckPermission checks if a user has permission (using RBAC or ABAC)
func (m *AuthManager) CheckPermission(userID uint, resource, action string, resourceAttrs map[string]interface{}) error {
	if m.permissionType == PermissionTypeRBAC {
		return m.rbac.CheckPermission(userID, resource, action)
	}

	if m.permissionType == PermissionTypeABAC {
		// Load user attributes
		var userAttrs map[string]interface{}
		if m.loadUserInfo != nil {
			userInfo, err := m.loadUserInfo(userID)
			if err == nil {
				userAttrs = userInfo.Attrs
			}
		}
		if userAttrs == nil {
			userAttrs = make(map[string]interface{})
		}
		userAttrs["id"] = userID

		return m.abac.CheckAccessWithError(userAttrs, resourceAttrs, action)
	}

	return errors.New("no permission manager configured")
}

// HasPermission checks if a user has permission (returns bool)
func (m *AuthManager) HasPermission(userID uint, resource, action string, resourceAttrs map[string]interface{}) bool {
	return m.CheckPermission(userID, resource, action, resourceAttrs) == nil
}

// GetRBAC returns RBAC manager if available
func (m *AuthManager) GetRBAC() *RBAC {
	return m.rbac
}

// GetABAC returns ABAC manager if available
func (m *AuthManager) GetABAC() *ABAC {
	return m.abac
}

// GetOAuth2Server returns OAuth2 server if available
func (m *AuthManager) GetOAuth2Server() *OAuth2Server {
	return m.oauth2Server
}

// GetJWTManager returns JWT manager if available
func (m *AuthManager) GetJWTManager() *JWTManager {
	return m.jwtManager
}

// Store user loader function
var m *AuthManager

// SetUserLoader sets the user loader function
func (m *AuthManager) SetUserLoader(loader func(userID uint) (UserInfo, error)) {
	m.loadUserInfo = loader
}
