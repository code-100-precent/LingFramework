package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDefaultAuthConfig(t *testing.T) {
	config := DefaultAuthConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "Authorization", config.TokenHeader)
	assert.Equal(t, "Bearer ", config.TokenPrefix)
	assert.Contains(t, config.SkipPaths, "/api/auth/login")
	assert.Contains(t, config.SkipPaths, "/api/auth/register")
	assert.Contains(t, config.SkipPaths, "/health")
	assert.Contains(t, config.SkipPaths, "/metrics")
}

func TestAuthMiddleware_NilConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware(nil))
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
	config := DefaultAuthConfig()
	r := gin.New()
	r.Use(AuthMiddleware(config))
	r.GET("/api/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/auth/login", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := DefaultAuthConfig()
	r := gin.New()
	r.Use(AuthMiddleware(config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_TokenFromHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := DefaultAuthConfig()
	config.UserIDExtractor = func(c *gin.Context) (uint, error) {
		return 1, nil
	}
	r := gin.New()
	r.Use(AuthMiddleware(config))
	r.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_TokenFromHeader_WithoutPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := DefaultAuthConfig()
	config.UserIDExtractor = func(c *gin.Context) (uint, error) {
		return 1, nil
	}
	r := gin.New()
	r.Use(AuthMiddleware(config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "test-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_TokenFromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := DefaultAuthConfig()
	config.UserIDExtractor = func(c *gin.Context) (uint, error) {
		return 1, nil
	}
	r := gin.New()
	r.Use(AuthMiddleware(config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test?token=test-token", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_TokenFromCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := DefaultAuthConfig()
	config.UserIDExtractor = func(c *gin.Context) (uint, error) {
		return 1, nil
	}
	r := gin.New()
	r.Use(AuthMiddleware(config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "test-token"})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_UserInfoExtractor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := DefaultAuthConfig()
	config.UserInfoExtractor = func(c *gin.Context) (interface{}, error) {
		return map[string]interface{}{"id": 1, "name": "test"}, nil
	}
	r := gin.New()
	r.Use(AuthMiddleware(config))
	r.GET("/test", func(c *gin.Context) {
		user, _ := c.Get("user")
		c.JSON(http.StatusOK, gin.H{"user": user})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_UserInfoExtractor_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := DefaultAuthConfig()
	config.UserInfoExtractor = func(c *gin.Context) (interface{}, error) {
		return nil, assert.AnError
	}
	r := gin.New()
	r.Use(AuthMiddleware(config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_UserIDExtractor_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := DefaultAuthConfig()
	config.UserIDExtractor = func(c *gin.Context) (uint, error) {
		return 0, assert.AnError
	}
	r := gin.New()
	r.Use(AuthMiddleware(config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_CustomTokenHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := &AuthConfig{
		TokenHeader: "X-Auth-Token",
		TokenPrefix: "",
		SkipPaths:   []string{},
	}
	config.UserIDExtractor = func(c *gin.Context) (uint, error) {
		return 1, nil
	}
	r := gin.New()
	r.Use(AuthMiddleware(config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Auth-Token", "test-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestExtractToken_FromHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer test-token")

	config := DefaultAuthConfig()
	token := extractToken(c, config)
	assert.Equal(t, "test-token", token)
}

func TestExtractToken_FromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/test?token=query-token", nil)

	config := DefaultAuthConfig()
	token := extractToken(c, config)
	assert.Equal(t, "query-token", token)
}

func TestExtractToken_FromCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "cookie-token"})
	c.Request = req

	config := DefaultAuthConfig()
	token := extractToken(c, config)
	assert.Equal(t, "cookie-token", token)
}

func TestExtractToken_Priority(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/test?token=query-token", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	req.AddCookie(&http.Cookie{Name: "token", Value: "cookie-token"})
	c.Request = req

	config := DefaultAuthConfig()
	token := extractToken(c, config)
	// Header should have priority
	assert.Equal(t, "header-token", token)
}
