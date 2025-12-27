package middleware

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// CompressionConfig represents compression middleware configuration
type CompressionConfig struct {
	// Compression level (1-9, default: 6)
	Level int
	// Minimum content length to compress (default: 1024 bytes)
	MinLength int
	// Content types to compress
	ContentTypes []string
	// Exclude paths from compression
	ExcludePaths []string
}

// DefaultCompressionConfig returns default compression configuration
func DefaultCompressionConfig() *CompressionConfig {
	return &CompressionConfig{
		Level:     6,
		MinLength: 1024,
		ContentTypes: []string{
			"text/html",
			"text/css",
			"text/plain",
			"text/javascript",
			"application/javascript",
			"application/json",
			"application/xml",
			"text/xml",
		},
		ExcludePaths: []string{
			"/metrics",
			"/health",
		},
	}
}

// CompressionMiddleware creates compression middleware
func CompressionMiddleware(config *CompressionConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultCompressionConfig()
	}

	// Use default gzip middleware with level
	return gzip.Gzip(config.Level)
}
