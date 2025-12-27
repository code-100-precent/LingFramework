package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	LingEcho "github.com/code-100-precent/LingFramework"
	"github.com/code-100-precent/LingFramework/cmd/bootstrap"
	"github.com/code-100-precent/LingFramework/internal/handlers"
	"github.com/code-100-precent/LingFramework/pkg/config"
	"github.com/code-100-precent/LingFramework/pkg/constants"
	"github.com/code-100-precent/LingFramework/pkg/logger"
	"github.com/code-100-precent/LingFramework/pkg/middleware"
	"github.com/code-100-precent/LingFramework/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	// Global variables for SSL certificate
	sslCert     tls.Certificate
	sslCertOnce sync.Once
	sslCertErr  error
)

type LingEchoApp struct {
	db       *gorm.DB
	handlers *handlers.Handlers
}

func NewLingEchoApp(db *gorm.DB) *LingEchoApp {
	return &LingEchoApp{
		db:       db,
		handlers: handlers.NewHandlers(db),
	}
}

func (app *LingEchoApp) RegisterRoutes(r *gin.Engine) {
	// Register system routes (with /api prefix)
	app.handlers.Register(r)
}

func main() {
	// 1. Print Banner
	if err := bootstrap.PrintBannerFromFile("banner.txt"); err != nil {
		log.Fatalf("unload banner: %v", err)
	}

	// 2. Parse Command Line Parameters
	mode := flag.String("mode", "", "running environment (development, test, production)")
	initSQL := flag.String("init-sql", "", "path to database init .sql script (optional)")
	flag.Parse()

	// 3. Set Environment Variables
	if *mode != "" {
		os.Setenv("APP_ENV", *mode)
	}

	// 4. Load Global Configuration
	if err := config.Load(); err != nil {
		panic("config load failed: " + err.Error())
	}

	// 5. Load Log Configuration
	err := logger.Init(&config.GlobalConfig.Log, config.GlobalConfig.Mode)
	if err != nil {
		panic(err)
	}

	// 6. Print Configuration
	bootstrap.LogConfigInfo()

	// 7. Load Data Source
	db, err := bootstrap.SetupDatabase(os.Stdout, &bootstrap.Options{
		InitSQLPath: *initSQL,                             // Can be specified via --init-sql
		AutoMigrate: true,                                 // Whether to migrate entities
		SeedNonProd: os.Getenv("APP_ENV") != "production", // Non-production default configuration
	})
	if err != nil {
		logger.Error("database setup failed", zap.Error(err))
		return
	}

	// 8. Load Base Configs
	var addr = config.GlobalConfig.Addr
	if addr == "" {
		addr = ":7072"
	}

	var DBDriver = config.GlobalConfig.DBDriver
	if DBDriver == "" {
		DBDriver = "sqlite"
	}

	var DSN = config.GlobalConfig.DSN
	if DSN == "" {
		DSN = "file::memory:?cache=shared"
	}
	flag.StringVar(&addr, "addr", addr, "HTTP Serve address")
	flag.StringVar(&DBDriver, "db-driver", DBDriver, "database driver")
	flag.StringVar(&DSN, "dsn", DSN, "database source name")

	logger.Info("checked config -- addr: ", zap.String("addr", addr))
	logger.Info("checked config -- db-driver: ", zap.String("db-driver", DBDriver), zap.String("dsn", DSN))
	logger.Info("checked config -- mode: ", zap.String("mode", config.GlobalConfig.Mode))
	// 11. New App
	app := NewLingEchoApp(db)
	// 15. Initialize Gin Routing
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()        // Use gin.New() instead of gin.Default() to avoid automatic redirects
	r.Use(gin.Recovery()) // Manually add Recovery middleware
	r.LoadHTMLGlob("templates/**/**")

	// Disable automatic redirects to avoid CORS issues caused by 307 redirects
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	// Set maximum memory limit for multipart forms (32MB)
	r.MaxMultipartMemory = 32 << 20 // 32 MB

	// Cookie Register
	secret := utils.GetEnv(constants.ENV_SESSION_SECRET)
	if secret != "" {
		expireDays := utils.GetIntEnv(constants.ENV_SESSION_EXPIRE_DAYS)
		if expireDays <= 0 {
			expireDays = 7
		}
		r.Use(middleware.WithCookieSession(secret, int(expireDays)*24*3600))
	} else {
		r.Use(middleware.WithMemSession(utils.RandText(32)))
	}

	// Cors Handle Middleware
	r.Use(middleware.CorsMiddleware())

	// Logger Handle Middleware
	r.Use(middleware.LoggerMiddleware(zap.L()))

	// RateLimit Middleware - Loosen rate limiting configuration
	middleware.SetRateLimiterConfig(middleware.RateLimiterConfig{
		Rate:        "1000-M", // 1000 requests per minute, much more relaxed than the default 10 per second
		Identifier:  "ip",
		AddHeaders:  true,
		DenyStatus:  429,
		DenyMessage: "Requests too frequent, please try again later",
		PerRouteRates: map[string]string{
			"/api/voice/oneshot": "100-M", // Voice interface slightly stricter
			"/api/chat/call":     "50-M",  // Real-time call interface
			"/api/assistant":     "200-M", // Assistant-related interface
		},
		SkipPaths: []string{
			"/health",
			"/metrics",
			"/static/",
			"/uploads/",
			"/media/", // keep for backward compatibility
		},
	})
	r.Use(middleware.RateLimiterMiddleware())

	// Assets Middleware
	r.Use(LingEcho.WithStaticAssets(r, utils.GetEnv(constants.ENV_STATIC_PREFIX), utils.GetEnv(constants.ENV_STATIC_ROOT)))
	apiPrefix := config.GlobalConfig.APIPrefix
	// Static service for uploaded files
	uploadDir := utils.GetEnv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	if apiPrefix == "" {
		apiPrefix = "/api"
	}
	r.Static(apiPrefix+"/uploads", uploadDir)
	r.Static(apiPrefix+"/media", uploadDir)
	// Add /api/static route to serve static files under API prefix
	// This is needed for SDK files accessed via /api/static/js/lingecho-sdk.js
	staticRootDir := utils.GetEnv(constants.ENV_STATIC_ROOT)
	if staticRootDir == "" {
		staticRootDir = "static"
	}
	staticAssets := LingEcho.NewCombineEmbedFS(LingEcho.HintAssetsRoot(staticRootDir), LingEcho.EmbedFS{"static", LingEcho.EmbedStaticAssets})
	apiPrefix = config.GlobalConfig.APIPrefix
	if apiPrefix == "" {
		apiPrefix = "/api"
	}
	r.StaticFS(apiPrefix+"/static", http.FS(staticAssets))

	// 18. Register Routes
	app.RegisterRoutes(r)

	// 18.6. Register Metrics Monitor Routes
	// Get API prefix from config (default: /api)
	apiPrefix = config.GlobalConfig.APIPrefix
	if apiPrefix == "" {
		apiPrefix = "/api"
	}

	// 22. Start HTTP/HTTPS Server
	httpServer := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Check if SSL is enabled
	if config.GlobalConfig.SSLEnabled {
		loadSSLCertificates()
		tlsConfig, err := GetTLSConfig()
		if err != nil {
			logger.Error("failed to get TLS config", zap.Error(err))
			return
		}

		if tlsConfig != nil {
			httpServer.TLSConfig = tlsConfig
			logger.Info("Starting HTTPS server", zap.String("addr", addr))
			if err := httpServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				logger.Error("HTTPS server run failed", zap.Error(err))
			}
		} else {
			logger.Warn("SSL enabled but TLS config is nil, falling back to HTTP")
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("HTTP server run failed", zap.Error(err))
			}
		}
	} else {
		logger.Info("Starting HTTP server", zap.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server run failed", zap.Error(err))
		}
	}
}

// loadSSLCertificates loads SSL certificates
func loadSSLCertificates() {
	if !config.GlobalConfig.SSLEnabled {
		logger.Info("SSL is disabled, skipping SSL certificate loading")
		return
	}

	certFile := config.GlobalConfig.SSLCertFile
	keyFile := config.GlobalConfig.SSLKeyFile

	if certFile == "" || keyFile == "" {
		logger.Warn("SSL enabled but certificate files not configured",
			zap.String("certFile", certFile),
			zap.String("keyFile", keyFile))
		return
	}

	// Use sync.Once to ensure loading only once
	sslCertOnce.Do(func() {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			sslCertErr = err
			logger.Error("Failed to load SSL certificates",
				zap.String("certFile", certFile),
				zap.String("keyFile", keyFile),
				zap.Error(err))
			return
		}

		sslCert = cert
		logger.Info("SSL certificates loaded successfully",
			zap.String("certFile", certFile),
			zap.String("keyFile", keyFile))
	})
}

// GetSSLCertificate gets the loaded SSL certificate
func GetSSLCertificate() (tls.Certificate, error) {
	if sslCertErr != nil {
		return tls.Certificate{}, sslCertErr
	}
	return sslCert, nil
}

// IsSSLEnabled checks if SSL is enabled and certificates are loaded
func IsSSLEnabled() bool {
	return config.GlobalConfig.SSLEnabled && sslCertErr == nil
}

// GetTLSConfig gets TLS configuration
func GetTLSConfig() (*tls.Config, error) {
	if !IsSSLEnabled() {
		return nil, nil
	}

	cert, err := GetSSLCertificate()
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		// Recommended TLS configuration
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: true,
	}, nil
}
