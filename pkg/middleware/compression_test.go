package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDefaultCompressionConfig(t *testing.T) {
	config := DefaultCompressionConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 6, config.Level)
	assert.Equal(t, 1024, config.MinLength)
	assert.Contains(t, config.ContentTypes, "text/html")
	assert.Contains(t, config.ContentTypes, "application/json")
	assert.Contains(t, config.ExcludePaths, "/metrics")
	assert.Contains(t, config.ExcludePaths, "/health")
}

func TestCompressionMiddleware_NilConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CompressionMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCompressionMiddleware_WithConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := DefaultCompressionConfig()
	config.Level = 9
	r := gin.New()
	r.Use(CompressionMiddleware(config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "data": "test data"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCompressionMiddleware_DifferentLevels(t *testing.T) {
	levels := []int{1, 6, 9}
	for _, level := range levels {
		gin.SetMode(gin.TestMode)
		config := &CompressionConfig{Level: level}
		r := gin.New()
		r.Use(CompressionMiddleware(config))
		r.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Level %d should work", level)
	}
}

func TestCompressionMiddleware_WithoutAcceptEncoding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := DefaultCompressionConfig()
	r := gin.New()
	r.Use(CompressionMiddleware(config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	// No Accept-Encoding header
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCompressionMiddleware_LargeResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := DefaultCompressionConfig()
	r := gin.New()
	r.Use(CompressionMiddleware(config))
	r.GET("/test", func(c *gin.Context) {
		// Generate large response
		data := make([]string, 1000)
		for i := range data {
			data[i] = "test data"
		}
		c.JSON(http.StatusOK, gin.H{"data": data})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
