package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/code-100-precent/LingFramework/pkg/utils"
	"github.com/gin-gonic/gin"
)

// MiddlewareConfig represents middleware configuration
type MiddlewareConfig struct {
	AuthManager    *AuthManager
	SkipPaths      []string
	TokenHeader    string
	TokenPrefix    string
	AuthType       AuthType
	PermissionType PermissionType
}

// DefaultMiddlewareConfig returns default middleware configuration
// Reads configuration from environment variables with defaults
func DefaultMiddlewareConfig(authManager *AuthManager) *MiddlewareConfig {
	if authManager == nil {
		panic("AuthManager cannot be nil")
	}

	// Read from environment variables with defaults
	tokenHeader := utils.GetEnv("AUTH_TOKEN_HEADER")
	if tokenHeader == "" {
		tokenHeader = "Authorization"
	}

	tokenPrefix := utils.GetEnv("AUTH_TOKEN_PREFIX")
	if tokenPrefix == "" {
		tokenPrefix = "Bearer "
	}

	// Get skip paths from environment (comma-separated) or use defaults
	skipPathsStr := utils.GetEnv("AUTH_SKIP_PATHS")
	skipPaths := []string{
		"/api/auth/login",
		"/api/auth/register",
		"/api/auth/refresh",
		"/health",
		"/metrics",
	}
	if skipPathsStr != "" {
		paths := strings.Split(skipPathsStr, ",")
		for _, path := range paths {
			path = strings.TrimSpace(path)
			if path != "" {
				skipPaths = append(skipPaths, path)
			}
		}
	}

	return &MiddlewareConfig{
		AuthManager:    authManager,
		SkipPaths:      skipPaths,
		TokenHeader:    tokenHeader,
		TokenPrefix:    tokenPrefix,
		AuthType:       authManager.authType,
		PermissionType: authManager.permissionType,
	}
}

// LoadAuthConfigFromEnv loads authentication configuration from environment variables
func LoadAuthConfigFromEnv() (*AuthConfig, error) {
	// Read auth type (default: jwt)
	authTypeStr := utils.GetEnv("AUTH_TYPE")
	if authTypeStr == "" {
		authTypeStr = "jwt"
	}
	authType := AuthType(strings.ToLower(authTypeStr))
	if authType != AuthTypeJWT && authType != AuthTypeOAuth2 {
		authType = AuthTypeJWT // Default to JWT if invalid
	}

	// Read permission type (default: rbac)
	permissionTypeStr := utils.GetEnv("PERMISSION_TYPE")
	if permissionTypeStr == "" {
		permissionTypeStr = "rbac"
	}
	permissionType := PermissionType(strings.ToLower(permissionTypeStr))
	if permissionType != PermissionTypeRBAC && permissionType != PermissionTypeABAC {
		permissionType = PermissionTypeRBAC // Default to RBAC if invalid
	}

	config := &AuthConfig{
		AuthType:       authType,
		PermissionType: permissionType,
	}

	// JWT configuration
	if authType == AuthTypeJWT {
		jwtSecretKey := utils.GetEnv("JWT_SECRET_KEY")
		if jwtSecretKey == "" {
			jwtSecretKey = utils.GetEnv("SESSION_SECRET") // Fallback to session secret
		}
		if jwtSecretKey == "" {
			return nil, &ConfigError{Message: "JWT_SECRET_KEY is required when using JWT authentication"}
		}

		config.JWTSecretKey = jwtSecretKey

		// JWT token TTL (in seconds, converted to duration)
		jwtAccessTTL := utils.GetIntEnv("JWT_ACCESS_TOKEN_TTL")
		if jwtAccessTTL > 0 {
			config.JWTAccessTokenTTL = time.Duration(jwtAccessTTL) * time.Second
		} else {
			config.JWTAccessTokenTTL = 15 * time.Minute // Default 15 minutes
		}

		jwtRefreshTTL := utils.GetIntEnv("JWT_REFRESH_TOKEN_TTL")
		if jwtRefreshTTL > 0 {
			config.JWTRefreshTokenTTL = time.Duration(jwtRefreshTTL) * time.Second
		} else {
			config.JWTRefreshTokenTTL = 7 * 24 * time.Hour // Default 7 days
		}

		jwtIssuer := utils.GetEnv("JWT_ISSUER")
		if jwtIssuer == "" {
			jwtIssuer = "LingFramework"
		}
		config.JWTIssuer = jwtIssuer
	}

	// OAuth2 configuration
	if authType == AuthTypeOAuth2 {
		oauth2AccessTTL := utils.GetIntEnv("OAUTH2_ACCESS_TOKEN_TTL")
		if oauth2AccessTTL > 0 {
			config.OAuth2AccessTokenTTL = time.Duration(oauth2AccessTTL) * time.Second
		} else {
			config.OAuth2AccessTokenTTL = 15 * time.Minute // Default 15 minutes
		}

		oauth2RefreshTTL := utils.GetIntEnv("OAUTH2_REFRESH_TOKEN_TTL")
		if oauth2RefreshTTL > 0 {
			config.OAuth2RefreshTokenTTL = time.Duration(oauth2RefreshTTL) * time.Second
		} else {
			config.OAuth2RefreshTokenTTL = 7 * 24 * time.Hour // Default 7 days
		}
	}

	return config, nil
}

// ConfigError represents a configuration error
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}

// AuthMiddleware creates authentication middleware using AuthManager
func AuthMiddleware(config *MiddlewareConfig) gin.HandlerFunc {
	if config == nil {
		panic("MiddlewareConfig cannot be nil")
	}
	if config.AuthManager == nil {
		panic("AuthManager cannot be nil")
	}

	return func(c *gin.Context) {
		// Check if path should be skipped
		path := c.Request.URL.Path
		for _, skipPath := range config.SkipPaths {
			if strings.HasPrefix(path, skipPath) {
				c.Next()
				return
			}
		}

		// Extract token
		token := extractTokenFromRequest(c, config)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			return
		}

		// Validate token
		userID, username, roles, err := config.AuthManager.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			return
		}

		// Set user information in context
		c.Set("user_id", userID)
		c.Set("username", username)
		c.Set("roles", roles)
		c.Set("token", token)

		// Set user object for compatibility
		userInfo := map[string]interface{}{
			"id":       userID,
			"username": username,
			"roles":    roles,
		}
		c.Set("user", userInfo)

		c.Next()
	}
}

// PermissionMiddleware creates permission check middleware
func PermissionMiddleware(config *MiddlewareConfig, resource, action string) gin.HandlerFunc {
	if config == nil {
		panic("MiddlewareConfig cannot be nil")
	}
	if config.AuthManager == nil {
		panic("AuthManager cannot be nil")
	}

	return func(c *gin.Context) {
		// Get user ID from context (should be set by AuthMiddleware)
		userIDInterface, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			return
		}

		userID, ok := userIDInterface.(uint)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid user ID",
			})
			return
		}

		// Extract resource attributes from request (for ABAC)
		resourceAttrs := extractResourceAttributes(c, resource)

		// Check permission
		err := config.AuthManager.CheckPermission(userID, resource, action, resourceAttrs)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Permission denied",
			})
			return
		}

		c.Next()
	}
}

// PermissionMiddlewareWithAttrs creates permission check middleware with custom resource attributes
func PermissionMiddlewareWithAttrs(config *MiddlewareConfig, resource, action string, getResourceAttrs func(*gin.Context) map[string]interface{}) gin.HandlerFunc {
	if config == nil {
		panic("MiddlewareConfig cannot be nil")
	}
	if config.AuthManager == nil {
		panic("AuthManager cannot be nil")
	}

	return func(c *gin.Context) {
		// Get user ID from context
		userIDInterface, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			return
		}

		userID, ok := userIDInterface.(uint)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid user ID",
			})
			return
		}

		// Get resource attributes
		var resourceAttrs map[string]interface{}
		if getResourceAttrs != nil {
			resourceAttrs = getResourceAttrs(c)
		} else {
			resourceAttrs = extractResourceAttributes(c, resource)
		}

		// Check permission
		err := config.AuthManager.CheckPermission(userID, resource, action, resourceAttrs)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Permission denied",
			})
			return
		}

		c.Next()
	}
}

// extractTokenFromRequest extracts token from request
func extractTokenFromRequest(c *gin.Context, config *MiddlewareConfig) string {
	// Try to get token from header
	authHeader := c.GetHeader(config.TokenHeader)
	if authHeader != "" {
		if config.TokenPrefix != "" && strings.HasPrefix(authHeader, config.TokenPrefix) {
			return strings.TrimPrefix(authHeader, config.TokenPrefix)
		}
		// If prefix is empty, return the whole header
		if config.TokenPrefix == "" {
			return authHeader
		}
	}

	// Try to get token from query parameter
	token := c.Query("token")
	if token != "" {
		return token
	}

	// Try to get token from cookie
	token, _ = c.Cookie("token")
	return token
}

// extractResourceAttributes extracts resource attributes from request context
func extractResourceAttributes(c *gin.Context, resource string) map[string]interface{} {
	attrs := make(map[string]interface{})
	attrs["type"] = resource

	// Try to get resource ID from URL parameters or context
	if resourceID := c.Param("id"); resourceID != "" {
		attrs["id"] = resourceID
	}

	// Try to get owner_id from context (set by other middleware)
	if ownerID, exists := c.Get("owner_id"); exists {
		attrs["owner_id"] = ownerID
	}

	return attrs
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) (uint, bool) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	userID, ok := userIDInterface.(uint)
	return userID, ok
}

// GetUsername extracts username from context
func GetUsername(c *gin.Context) (string, bool) {
	username, exists := c.Get("username")
	if !exists {
		return "", false
	}

	usernameStr, ok := username.(string)
	return usernameStr, ok
}

// GetRoles extracts roles from context
func GetRoles(c *gin.Context) ([]string, bool) {
	roles, exists := c.Get("roles")
	if !exists {
		return nil, false
	}

	rolesSlice, ok := roles.([]string)
	return rolesSlice, ok
}
