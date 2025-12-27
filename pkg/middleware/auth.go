package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthConfig represents authentication middleware configuration
type AuthConfig struct {
	// Token header name (default: "Authorization")
	TokenHeader string
	// Token prefix (default: "Bearer ")
	TokenPrefix string
	// Skip paths that don't require authentication
	SkipPaths []string
	// User ID extractor function
	UserIDExtractor func(*gin.Context) (uint, error)
	// User info extractor function
	UserInfoExtractor func(*gin.Context) (interface{}, error)
}

// DefaultAuthConfig returns default authentication configuration
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		TokenHeader: "Authorization",
		TokenPrefix: "Bearer ",
		SkipPaths: []string{
			"/api/auth/login",
			"/api/auth/register",
			"/api/auth/refresh",
			"/health",
			"/metrics",
		},
	}
}

// AuthMiddleware creates authentication middleware
func AuthMiddleware(config *AuthConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultAuthConfig()
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

		// Extract token from header
		token := extractToken(c, config)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			return
		}

		// Extract user information
		if config.UserInfoExtractor != nil {
			userInfo, err := config.UserInfoExtractor(c)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Invalid token",
				})
				return
			}
			c.Set("user", userInfo)
		}

		// Extract user ID
		if config.UserIDExtractor != nil {
			userID, err := config.UserIDExtractor(c)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Invalid token",
				})
				return
			}
			c.Set("user_id", userID)
		}

		c.Next()
	}
}

// extractToken extracts token from request
func extractToken(c *gin.Context, config *AuthConfig) string {
	// Try to get token from header
	authHeader := c.GetHeader(config.TokenHeader)
	if authHeader != "" {
		if strings.HasPrefix(authHeader, config.TokenPrefix) {
			return strings.TrimPrefix(authHeader, config.TokenPrefix)
		}
		return authHeader
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
