package cache

import (
	"context"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// LRUCacheConfig LRU缓存配置
type LRUCacheConfig struct {
	// 最大缓存项数
	MaxSize int `json:"max_size" yaml:"max_size" env:"LRU_CACHE_MAX_SIZE" default:"1000"`

	// 默认过期时间
	DefaultExpiration time.Duration `json:"default_expiration" yaml:"default_expiration" env:"LRU_CACHE_DEFAULT_EXPIRATION" default:"5m"`

	// 清理间隔
	CleanupInterval time.Duration `json:"cleanup_interval" yaml:"cleanup_interval" env:"LRU_CACHE_CLEANUP_INTERVAL" default:"10m"`
}

// lruCache LRU缓存实现
type lruCacheImpl struct {
	cache    *lru.Cache[string, *lruCacheItem]
	config   LRUCacheConfig
	mu       sync.RWMutex
	stopChan chan struct{}
}

// lruCacheItem LRU缓存项
type lruCacheItem struct {
	value      interface{}
	expiration time.Time
}

// NewLRUCache 创建基于hashicorp/golang-lru的缓存
func NewLRUCache(config LRUCacheConfig) Cache {
	c, err := lru.New[string, *lruCacheItem](config.MaxSize)
	if err != nil {
		// 如果创建失败，使用默认大小
		c, _ = lru.New[string, *lruCacheItem](1000)
	}

	lc := &lruCacheImpl{
		cache:    c,
		config:   config,
		stopChan: make(chan struct{}),
	}

	// 启动清理协程
	go lc.startCleanup()

	return lc
}

// Get 获取缓存值
func (lc *lruCacheImpl) Get(ctx context.Context, key string) (interface{}, bool) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	item, ok := lc.cache.Get(key)
	if !ok {
		return nil, false
	}

	// 检查是否过期
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		lc.cache.Remove(key)
		return nil, false
	}

	return item.value, true
}

// Set 设置缓存值
func (lc *lruCacheImpl) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	var exp time.Time
	if expiration > 0 {
		exp = time.Now().Add(expiration)
	} else if lc.config.DefaultExpiration > 0 {
		exp = time.Now().Add(lc.config.DefaultExpiration)
	}

	item := &lruCacheItem{
		value:      value,
		expiration: exp,
	}

	lc.cache.Add(key, item)
	return nil
}

// Delete 删除缓存
func (lc *lruCacheImpl) Delete(ctx context.Context, key string) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.cache.Remove(key)
	return nil
}

// Exists 检查键是否存在
func (lc *lruCacheImpl) Exists(ctx context.Context, key string) bool {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	item, ok := lc.cache.Get(key)
	if !ok {
		return false
	}

	// 检查是否过期
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		lc.cache.Remove(key)
		return false
	}

	return true
}

// Clear 清空所有缓存
func (lc *lruCacheImpl) Clear(ctx context.Context) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	// 清空所有项
	for lc.cache.Len() > 0 {
		keys := lc.cache.Keys()
		for _, key := range keys {
			lc.cache.Remove(key)
		}
	}
	return nil
}

// GetMulti 批量获取
func (lc *lruCacheImpl) GetMulti(ctx context.Context, keys ...string) map[string]interface{} {
	result := make(map[string]interface{})
	for _, key := range keys {
		if value, exists := lc.Get(ctx, key); exists {
			result[key] = value
		}
	}
	return result
}

// SetMulti 批量设置
func (lc *lruCacheImpl) SetMulti(ctx context.Context, data map[string]interface{}, expiration time.Duration) error {
	for key, value := range data {
		if err := lc.Set(ctx, key, value, expiration); err != nil {
			return err
		}
	}
	return nil
}

// DeleteMulti 批量删除
func (lc *lruCacheImpl) DeleteMulti(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		if err := lc.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// Increment 自增
func (lc *lruCacheImpl) Increment(ctx context.Context, key string, value int64) (int64, error) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	item, ok := lc.cache.Get(key)
	if !ok {
		// 如果不存在，创建新值
		newValue := value
		lc.cache.Add(key, &lruCacheItem{
			value:      newValue,
			expiration: time.Now().Add(lc.config.DefaultExpiration),
		})
		return newValue, nil
	}

	// 检查是否过期
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		lc.cache.Remove(key)
		newValue := value
		lc.cache.Add(key, &lruCacheItem{
			value:      newValue,
			expiration: time.Now().Add(lc.config.DefaultExpiration),
		})
		return newValue, nil
	}

	// 尝试转换为数字并自增
	switch v := item.value.(type) {
	case int:
		newValue := int64(v) + value
		item.value = newValue
		lc.cache.Add(key, item) // 更新
		return newValue, nil
	case int64:
		newValue := v + value
		item.value = newValue
		lc.cache.Add(key, item) // 更新
		return newValue, nil
	case float64:
		newValue := int64(v) + value
		item.value = newValue
		lc.cache.Add(key, item) // 更新
		return newValue, nil
	default:
		// 如果类型不支持，重置为指定值
		item.value = value
		lc.cache.Add(key, item) // 更新
		return value, nil
	}
}

// Decrement 自减
func (lc *lruCacheImpl) Decrement(ctx context.Context, key string, value int64) (int64, error) {
	return lc.Increment(ctx, key, -value)
}

// GetWithTTL 获取值并返回剩余TTL
func (lc *lruCacheImpl) GetWithTTL(ctx context.Context, key string) (interface{}, time.Duration, bool) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	item, ok := lc.cache.Get(key)
	if !ok {
		return nil, 0, false
	}

	// 检查是否过期
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		lc.cache.Remove(key)
		return nil, 0, false
	}

	var ttl time.Duration
	if !item.expiration.IsZero() {
		ttl = item.expiration.Sub(time.Now())
		if ttl < 0 {
			ttl = 0
		}
	}

	return item.value, ttl, true
}

// Close 关闭缓存连接
func (lc *lruCacheImpl) Close() error {
	close(lc.stopChan)
	return nil
}

// startCleanup 启动清理协程
func (lc *lruCacheImpl) startCleanup() {
	if lc.config.CleanupInterval <= 0 {
		return
	}

	ticker := time.NewTicker(lc.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lc.cleanup()
		case <-lc.stopChan:
			return
		}
	}
}

// cleanup 清理过期项
func (lc *lruCacheImpl) cleanup() {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	now := time.Now()
	keys := lc.cache.Keys()
	for _, key := range keys {
		if item, ok := lc.cache.Peek(key); ok {
			if !item.expiration.IsZero() && now.After(item.expiration) {
				lc.cache.Remove(key)
			}
		}
	}
}
