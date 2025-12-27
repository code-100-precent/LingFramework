package config

import (
	"log"
	"os"
	"time"

	"github.com/code-100-precent/LingFramework/pkg/cache"
	"github.com/code-100-precent/LingFramework/pkg/logger"
	"github.com/code-100-precent/LingFramework/pkg/utils"
)

// Config represents the system configuration
type Config struct {
	MachineID        int64  `env:"MACHINE_ID"`
	ServerName       string `env:"SERVER_NAME"`
	ServerDesc       string `env:"SERVER_DESC"`
	ServerUrl        string `env:"SERVER_URL"`
	ServerLogo       string `env:"SERVER_LOGO"`
	ServerTermsUrl   string `env:"SERVER_TERMS_URL"`
	DBDriver         string `env:"DB_DRIVER"`
	DSN              string `env:"DSN"`
	Log              logger.LogConfig
	Addr             string `env:"ADDR"`
	Mode             string `env:"MODE"`
	DocsPrefix       string `env:"DOCS_PREFIX"`
	APIPrefix        string `env:"API_PREFIX"`
	AdminPrefix      string `env:"ADMIN_PREFIX"`
	AuthPrefix       string `env:"AUTH_PREFIX"`
	SessionSecret    string `env:"SESSION_SECRET"`
	SecretExpireDays string `env:"SESSION_EXPIRE_DAYS"`
	LLMApiKey        string `env:"LLM_API_KEY"`
	LLMBaseURL       string `env:"LLM_BASE_URL"`
	LLMModel         string `env:"LLM_MODEL"`
	SearchEnabled    bool   `env:"SEARCH_ENABLED"`
	SearchPath       string `env:"SEARCH_PATH"`
	SearchBatchSize  int    `env:"SEARCH_BATCH_SIZE"`
	MonitorPrefix    string `env:"MONITOR_PREFIX"`
	LanguageEnabled  bool   `env:"LANGUAGE_ENABLED"`
	APISecretKey     string `env:"API_SECRET_KEY"`
	BackupEnabled    bool   `env:"BACKUP_ENABLED"`
	BackupPath       string `env:"BACKUP_PATH"`
	BackupSchedule   string `env:"BACKUP_SCHEDULE"`
	Cache            cache.Config
	SSLEnabled       bool   `env:"SSL_ENABLED"`
	SSLCertFile      string `env:"SSL_CERT_FILE"`
	SSLKeyFile       string `env:"SSL_KEY_FILE"`
}

// GlobalConfig is the global configuration instance
var GlobalConfig *Config

// Load loads configuration from environment variables
func Load() error {
	// Load .env file based on APP_ENV
	env := os.Getenv("APP_ENV")
	err := utils.LoadEnv(env)
	if err != nil {
		// Log warning if .env file not found, but don't fail startup
		log.Printf("Note: .env file not found or failed to load: %v (using default values)", err)
	}

	// Load global configuration
	GlobalConfig = &Config{
		MachineID:        utils.GetIntEnv("MACHINE_ID"),
		ServerName:       getStringOrDefault("SERVER_NAME", ""),
		ServerDesc:       getStringOrDefault("SERVER_DESC", ""),
		ServerUrl:        getStringOrDefault("SERVER_URL", ""),
		ServerLogo:       getStringOrDefault("SERVER_LOGO", ""),
		ServerTermsUrl:   getStringOrDefault("SERVER_TERMS_URL", ""),
		DBDriver:         getStringOrDefault("DB_DRIVER", "sqlite"),
		DSN:              getStringOrDefault("DSN", "./ling.db"),
		Addr:             getStringOrDefault("ADDR", ":7072"),
		Mode:             getStringOrDefault("MODE", "development"),
		DocsPrefix:       getStringOrDefault("DOCS_PREFIX", "/api/docs"),
		APIPrefix:        getStringOrDefault("API_PREFIX", "/api"),
		AdminPrefix:      getStringOrDefault("ADMIN_PREFIX", "/admin"),
		AuthPrefix:       getStringOrDefault("AUTH_PREFIX", "/auth"),
		SecretExpireDays: getStringOrDefault("SESSION_EXPIRE_DAYS", "7"),
		SessionSecret:    getStringOrDefault("SESSION_SECRET", generateDefaultSessionSecret()),
		Log: logger.LogConfig{
			Level:      getStringOrDefault("LOG_LEVEL", "info"),
			Filename:   getStringOrDefault("LOG_FILENAME", "./logs/app.log"),
			MaxSize:    getIntOrDefault("LOG_MAX_SIZE", 100),
			MaxAge:     getIntOrDefault("LOG_MAX_AGE", 30),
			MaxBackups: getIntOrDefault("LOG_MAX_BACKUPS", 5),
			Daily:      getBoolOrDefault("LOG_DAILY", true),
		},
		LLMApiKey:       getStringOrDefault("LLM_API_KEY", ""),
		LLMBaseURL:      getStringOrDefault("LLM_BASE_URL", "https://api.openai.com/v1"),
		LLMModel:        getStringOrDefault("LLM_MODEL", "gpt-3.5-turbo"),
		SearchEnabled:   getBoolOrDefault("SEARCH_ENABLED", false),
		SearchPath:      getStringOrDefault("SEARCH_PATH", "./search"),
		SearchBatchSize: getIntOrDefault("SEARCH_BATCH_SIZE", 100),
		MonitorPrefix:   getStringOrDefault("MONITOR_PREFIX", "/metrics"),
		LanguageEnabled: getBoolOrDefault("LANGUAGE_ENABLED", true),
		APISecretKey:    getStringOrDefault("API_SECRET_KEY", generateDefaultSessionSecret()),
		BackupEnabled:   getBoolOrDefault("BACKUP_ENABLED", false),
		BackupPath:      getStringOrDefault("BACKUP_PATH", "./backups"),
		BackupSchedule:  getStringOrDefault("BACKUP_SCHEDULE", "0 2 * * *"),
		Cache:           loadCacheConfig(),
		SSLEnabled:      getBoolOrDefault("SSL_ENABLED", false),
		SSLCertFile:     getStringOrDefault("SSL_CERT_FILE", ""),
		SSLKeyFile:      getStringOrDefault("SSL_KEY_FILE", ""),
	}
	return nil
}

// getStringOrDefault gets environment variable value, returns default if empty
func getStringOrDefault(key, defaultValue string) string {
	value := utils.GetEnv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getBoolOrDefault gets boolean environment variable value, returns default if empty
func getBoolOrDefault(key string, defaultValue bool) bool {
	value := utils.GetEnv(key)
	if value == "" {
		return defaultValue
	}
	return utils.GetBoolEnv(key)
}

// getIntOrDefault gets integer environment variable value, returns default if zero
func getIntOrDefault(key string, defaultValue int) int {
	value := utils.GetIntEnv(key)
	if value == 0 {
		return defaultValue
	}
	return int(value)
}

// generateDefaultSessionSecret generates a default session secret for development only
// This should only be called when SESSION_SECRET is not set in environment
func generateDefaultSessionSecret() string {
	// Generate a random string for development use only
	return "default-secret-key-change-in-production-" + utils.RandText(16)
}

// loadCacheConfig loads cache configuration with all default values
func loadCacheConfig() cache.Config {
	cacheType := utils.GetEnv("CACHE_TYPE")
	if cacheType == "" {
		cacheType = "local"
	}

	// Helper function to parse duration strings
	parseDuration := func(s string, defaultVal time.Duration) time.Duration {
		if s == "" {
			return defaultVal
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			return defaultVal
		}
		return d
	}

	// Redis configuration
	redisAddr := utils.GetEnv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisDB := int(utils.GetIntEnv("REDIS_DB"))
	// Note: redisDB can be 0, which is valid, so we don't need to check for 0

	redisPoolSize := int(utils.GetIntEnv("REDIS_POOL_SIZE"))
	if redisPoolSize == 0 {
		redisPoolSize = 10
	}

	redisMinIdleConns := int(utils.GetIntEnv("REDIS_MIN_IDLE_CONNS"))
	if redisMinIdleConns == 0 {
		redisMinIdleConns = 5
	}

	// Local cache configuration
	localMaxSize := int(utils.GetIntEnv("LOCAL_CACHE_MAX_SIZE"))
	if localMaxSize == 0 {
		localMaxSize = 1000
	}

	localDefaultExpiration := parseDuration(utils.GetEnv("LOCAL_CACHE_DEFAULT_EXPIRATION"), 5*time.Minute)
	localCleanupInterval := parseDuration(utils.GetEnv("LOCAL_CACHE_CLEANUP_INTERVAL"), 10*time.Minute)

	return cache.Config{
		Type: cacheType,
		Redis: cache.RedisConfig{
			Addr:         redisAddr,
			Password:     utils.GetEnv("REDIS_PASSWORD"),
			DB:           redisDB,
			PoolSize:     redisPoolSize,
			MinIdleConns: redisMinIdleConns,
			DialTimeout:  parseDuration(utils.GetEnv("REDIS_DIAL_TIMEOUT"), 5*time.Second),
			ReadTimeout:  parseDuration(utils.GetEnv("REDIS_READ_TIMEOUT"), 3*time.Second),
			WriteTimeout: parseDuration(utils.GetEnv("REDIS_WRITE_TIMEOUT"), 3*time.Second),
			IdleTimeout:  parseDuration(utils.GetEnv("REDIS_IDLE_TIMEOUT"), 5*time.Minute),
		},
		Local: cache.LocalConfig{
			MaxSize:           localMaxSize,
			DefaultExpiration: localDefaultExpiration,
			CleanupInterval:   localCleanupInterval,
		},
	}
}
