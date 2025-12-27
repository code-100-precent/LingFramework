package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig represents JWT configuration
type JWTConfig struct {
	SecretKey       string        // Secret key for signing tokens
	AccessTokenTTL  time.Duration // Access token time to live
	RefreshTokenTTL time.Duration // Refresh token time to live
	Issuer          string        // Token issuer
	SigningMethod   jwt.SigningMethod
}

// DefaultJWTConfig returns default JWT configuration
func DefaultJWTConfig(secretKey string) *JWTConfig {
	return &JWTConfig{
		SecretKey:       secretKey,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "LingFramework",
		SigningMethod:   jwt.SigningMethodHS256,
	}
}

// JWTClaims represents JWT claims
type JWTClaims struct {
	UserID   uint                   `json:"user_id"`
	Username string                 `json:"username"`
	Roles    []string               `json:"roles,omitempty"`
	Extra    map[string]interface{} `json:"extra,omitempty"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT operations
type JWTManager struct {
	config *JWTConfig
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(config *JWTConfig) *JWTManager {
	if config == nil {
		panic("JWTConfig cannot be nil")
	}
	if config.SecretKey == "" {
		panic("JWTConfig.SecretKey cannot be empty")
	}
	return &JWTManager{config: config}
}

// GenerateAccessToken generates an access token
func (m *JWTManager) GenerateAccessToken(userID uint, username string, roles []string, extra map[string]interface{}) (string, error) {
	claims := &JWTClaims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		Extra:    extra,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.config.AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    m.config.Issuer,
		},
	}

	token := jwt.NewWithClaims(m.config.SigningMethod, claims)
	return token.SignedString([]byte(m.config.SecretKey))
}

// GenerateRefreshToken generates a refresh token
func (m *JWTManager) GenerateRefreshToken(userID uint, username string) (string, error) {
	claims := &JWTClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.config.RefreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    m.config.Issuer,
		},
	}

	token := jwt.NewWithClaims(m.config.SigningMethod, claims)
	return token.SignedString([]byte(m.config.SecretKey))
}

// ValidateToken validates and parses a JWT token
func (m *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// RefreshToken refreshes an access token using a refresh token
func (m *JWTManager) RefreshToken(refreshToken string) (string, error) {
	claims, err := m.ValidateToken(refreshToken)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Generate new access token with same user info
	return m.GenerateAccessToken(claims.UserID, claims.Username, claims.Roles, claims.Extra)
}

// ExtractClaims extracts claims from token without validation (for debugging)
func (m *JWTManager) ExtractClaims(tokenString string) (*JWTClaims, error) {
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(tokenString, &JWTClaims{})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok {
		return claims, nil
	}

	return nil, errors.New("invalid token format")
}
