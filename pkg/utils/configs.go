package utils

import (
	"context"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/code-100-precent/LingFramework/pkg/cache"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Config struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	Key       string `json:"key" gorm:"size:128;uniqueIndex"`
	Desc      string `json:"desc" gorm:"size:200"`
	Autoload  bool   `json:"autoload" gorm:"index"`
	Public    bool   `json:"public" gorm:"index" default:"false"`
	Format    string `json:"format" gorm:"size:20" default:"text" comment:"json,yaml,int,float,bool,text"`
	Value     string
	CreatedAt time.Time `json:"-" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"-" gorm:"autoUpdateTime"`
}

// ConfigManager 配置管理器
type ConfigManager struct {
	configValueCache cache.Cache
	envCache         cache.Cache
}

var defaultConfigManager *ConfigManager

// InitConfigManager 初始化配置管理器
func InitConfigManager(cacheInstance cache.Cache) {
	if cacheInstance == nil {
		// 如果没有提供缓存，创建一个默认的LRU缓存
		cacheInstance = cache.NewLRUCache(cache.LRUCacheConfig{
			MaxSize:           1024,
			DefaultExpiration: 10 * time.Second,
			CleanupInterval:   1 * time.Minute,
		})
	}
	defaultConfigManager = &ConfigManager{
		configValueCache: cacheInstance,
		envCache:         cacheInstance,
	}
}

// getConfigManager 获取配置管理器（如果未初始化则使用默认值）
func getConfigManager() *ConfigManager {
	if defaultConfigManager == nil {
		// 延迟初始化，使用默认缓存
		InitConfigManager(nil)
	}
	return defaultConfigManager
}

func GetEnv(key string) string {
	v, _ := LookupEnv(key)
	return v
}

func GetBoolEnv(key string) bool {
	v, _ := strconv.ParseBool(GetEnv(key))
	return v
}

func GetFloatEnv(key string) float64 {
	v, _ := strconv.ParseFloat(GetEnv(key), 64)
	return v
}

func GetIntEnv(key string) int64 {
	v, _ := strconv.ParseInt(GetEnv(key), 10, 64)
	return v
}

func LookupEnv(key string) (value string, found bool) {
	key = strings.ToUpper(key)
	if v, ok := os.LookupEnv(key); ok {
		cm := getConfigManager()
		if cm.envCache != nil {
			cm.envCache.Set(context.Background(), key, v, 10*time.Second)
		}
		return v, true
	}
	cm := getConfigManager()
	if cm.envCache != nil {
		if val, ok := cm.envCache.Get(context.Background(), key); ok {
			if v, ok := val.(string); ok {
				return v, true
			}
		}
	}
	data, err := os.ReadFile(".env")
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for i := 0; i < len(lines); i++ {
			v := strings.TrimSpace(lines[i])
			if v == "" || v[0] == '#' || !strings.Contains(v, "=") {
				continue
			}
			vs := strings.SplitN(v, "=", 2)
			k, vv := strings.ToUpper(strings.TrimSpace(vs[0])), strings.TrimSpace(vs[1])

			cm := getConfigManager()
			if cm.envCache != nil {
				cm.envCache.Set(context.Background(), k, vv, 10*time.Second)
			}
			if k == key {
				return vv, true
			}
		}
	}
	return "", false
}

// load envs to struct
func LoadEnvs(objPtr any) {
	if objPtr == nil {
		return
	}
	elm := reflect.ValueOf(objPtr).Elem()
	elmType := elm.Type()

	for i := 0; i < elm.NumField(); i++ {
		f := elm.Field(i)
		if !f.CanSet() {
			continue
		}
		keyName := elmType.Field(i).Tag.Get("env")
		if keyName == "-" {
			continue
		}
		if keyName == "" {
			keyName = elmType.Field(i).Name
		}
		switch f.Kind() {
		case reflect.String:
			if v, ok := LookupEnv(keyName); ok {
				f.SetString(v)
			}
		case reflect.Int:
			if v, ok := LookupEnv(keyName); ok {
				if iv, err := strconv.ParseInt(v, 10, 32); err == nil {
					f.SetInt(iv)
				}
			}
		case reflect.Bool:
			if v, ok := LookupEnv(keyName); ok {
				v := strings.ToLower(v)
				if yes, err := strconv.ParseBool(v); err == nil {
					f.SetBool(yes)
				}
			}
		}
	}
}

func SetValue(db *gorm.DB, key, value, format string, autoload, public bool) {
	key = strings.ToUpper(key)
	cm := getConfigManager()
	if cm.configValueCache != nil {
		cm.configValueCache.Delete(context.Background(), key)
	}

	newV := &Config{
		Key:      key,
		Value:    value,
		Format:   format,
		Autoload: autoload,
		Public:   public,
	}
	result := db.Model(&Config{}).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "format", "autoload", "public"}),
	}).Create(newV)

	if result.Error != nil {
		logrus.WithFields(logrus.Fields{
			"key":    key,
			"value":  value,
			"format": format,
		}).WithError(result.Error).Warn("config: setValue fail")
	}
}

func GetValue(db *gorm.DB, key string) string {
	key = strings.ToUpper(key)
	cm := getConfigManager()
	if cm.configValueCache != nil {
		if val, ok := cm.configValueCache.Get(context.Background(), key); ok {
			if v, ok := val.(string); ok {
				return v
			}
		}
	}

	var v Config
	result := db.Where("key", key).Take(&v)
	if result.Error != nil {
		return ""
	}

	if cm.configValueCache != nil {
		cm.configValueCache.Set(context.Background(), key, v.Value, 10*time.Second)
	}
	return v.Value
}

func GetIntValue(db *gorm.DB, key string, defaultVal int) int {
	v := GetValue(db, key)
	if v == "" {
		return defaultVal
	}
	val, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return defaultVal
	}
	return int(val)
}

func GetBoolValue(db *gorm.DB, key string) bool {
	v := GetValue(db, key)
	if v == "" {
		return false
	}

	r, _ := strconv.ParseBool(strings.ToLower(v))
	return r
}

func CheckValue(db *gorm.DB, key, defaultValue, format string, autoload, public bool) {
	newV := &Config{
		Key:      strings.ToUpper(key),
		Value:    defaultValue,
		Format:   format,
		Autoload: autoload,
		Public:   public,
	}
	db.Model(&Config{}).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoNothing: true,
	}).Create(newV)
}

func LoadAutoloads(db *gorm.DB) {
	var configs []Config
	db.Where("autoload", true).Find(&configs)
	cm := getConfigManager()
	for _, v := range configs {
		if cm.configValueCache != nil {
			cm.configValueCache.Set(context.Background(), v.Key, v.Value, 10*time.Second)
		}
	}
}

func LoadPublicConfigs(db *gorm.DB) []Config {
	var configs []Config
	db.Where("public", true).Find(&configs)
	cm := getConfigManager()
	for _, v := range configs {
		if cm.configValueCache != nil {
			cm.configValueCache.Set(context.Background(), v.Key, v.Value, 10*time.Second)
		}
	}
	return configs
}

// LoadEnv Load .env file based on environment
func LoadEnv(env string) error {
	// Load .env file by default
	envFile := ".env"
	if env != "" {
		envFile = ".env." + env
	}

	// Read .env file
	data, err := os.ReadFile(envFile)
	if err != nil {
		return err
	}

	// Parse .env file
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		os.Setenv(key, value)
	}

	return nil
}
