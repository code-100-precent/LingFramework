package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLRUCache(t *testing.T) {
	config := LRUCacheConfig{
		MaxSize:           100,
		DefaultExpiration: 5 * time.Minute,
		CleanupInterval:   10 * time.Minute,
	}
	cache := NewLRUCache(config)
	assert.NotNil(t, cache)
}

func TestLRUCache_GetSet(t *testing.T) {
	config := LRUCacheConfig{
		MaxSize:           100,
		DefaultExpiration: 5 * time.Minute,
		CleanupInterval:   10 * time.Minute,
	}
	cache := NewLRUCache(config)
	defer cache.Close()
	ctx := context.Background()

	// Set value
	err := cache.Set(ctx, "key1", "value1", 0)
	assert.NoError(t, err)

	// Get value
	val, exists := cache.Get(ctx, "key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", val)

	// Get non-existing key
	val, exists = cache.Get(ctx, "key_not_exists")
	assert.False(t, exists)
	assert.Nil(t, val)
}

func TestLRUCache_Expiration(t *testing.T) {
	config := LRUCacheConfig{
		MaxSize:           100,
		DefaultExpiration: 5 * time.Minute,
		CleanupInterval:   10 * time.Minute,
	}
	cache := NewLRUCache(config)
	defer cache.Close()
	ctx := context.Background()

	// Set value with expiration
	err := cache.Set(ctx, "key1", "value1", 50*time.Millisecond)
	assert.NoError(t, err)

	// Get value immediately
	val, exists := cache.Get(ctx, "key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", val)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Get value after expiration
	val, exists = cache.Get(ctx, "key1")
	assert.False(t, exists)
	assert.Nil(t, val)
}

func TestLRUCache_Delete(t *testing.T) {
	config := LRUCacheConfig{
		MaxSize:           100,
		DefaultExpiration: 5 * time.Minute,
		CleanupInterval:   10 * time.Minute,
	}
	cache := NewLRUCache(config)
	defer cache.Close()
	ctx := context.Background()

	// Set value
	err := cache.Set(ctx, "key1", "value1", 0)
	assert.NoError(t, err)

	// Delete value
	err = cache.Delete(ctx, "key1")
	assert.NoError(t, err)

	// Get deleted value
	val, exists := cache.Get(ctx, "key1")
	assert.False(t, exists)
	assert.Nil(t, val)
}

func TestLRUCache_Exists(t *testing.T) {
	config := LRUCacheConfig{
		MaxSize:           100,
		DefaultExpiration: 5 * time.Minute,
		CleanupInterval:   10 * time.Minute,
	}
	cache := NewLRUCache(config)
	defer cache.Close()
	ctx := context.Background()

	// Set value
	err := cache.Set(ctx, "key1", "value1", 0)
	assert.NoError(t, err)

	// Check exists
	assert.True(t, cache.Exists(ctx, "key1"))
	assert.False(t, cache.Exists(ctx, "key_not_exists"))
}

func TestLRUCache_Clear(t *testing.T) {
	config := LRUCacheConfig{
		MaxSize:           100,
		DefaultExpiration: 5 * time.Minute,
		CleanupInterval:   10 * time.Minute,
	}
	cache := NewLRUCache(config)
	defer cache.Close()
	ctx := context.Background()

	// Set multiple values
	cache.Set(ctx, "key1", "value1", 0)
	cache.Set(ctx, "key2", "value2", 0)
	cache.Set(ctx, "key3", "value3", 0)

	// Clear all
	err := cache.Clear(ctx)
	assert.NoError(t, err)

	// Check all values are gone
	assert.False(t, cache.Exists(ctx, "key1"))
	assert.False(t, cache.Exists(ctx, "key2"))
	assert.False(t, cache.Exists(ctx, "key3"))
}

func TestLRUCache_GetMulti(t *testing.T) {
	config := LRUCacheConfig{
		MaxSize:           100,
		DefaultExpiration: 5 * time.Minute,
		CleanupInterval:   10 * time.Minute,
	}
	cache := NewLRUCache(config)
	defer cache.Close()
	ctx := context.Background()

	// Set multiple values
	cache.Set(ctx, "key1", "value1", 0)
	cache.Set(ctx, "key2", "value2", 0)
	cache.Set(ctx, "key3", "value3", 0)

	// Get multiple values
	result := cache.GetMulti(ctx, "key1", "key2", "key_not_exists")
	assert.Equal(t, "value1", result["key1"])
	assert.Equal(t, "value2", result["key2"])
	_, exists := result["key_not_exists"]
	assert.False(t, exists)
}

func TestLRUCache_SetMulti(t *testing.T) {
	config := LRUCacheConfig{
		MaxSize:           100,
		DefaultExpiration: 5 * time.Minute,
		CleanupInterval:   10 * time.Minute,
	}
	cache := NewLRUCache(config)
	defer cache.Close()
	ctx := context.Background()

	// Set multiple values
	data := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := cache.SetMulti(ctx, data, 0)
	assert.NoError(t, err)

	// Verify all values
	assert.True(t, cache.Exists(ctx, "key1"))
	assert.True(t, cache.Exists(ctx, "key2"))
	assert.True(t, cache.Exists(ctx, "key3"))
}

func TestLRUCache_DeleteMulti(t *testing.T) {
	config := LRUCacheConfig{
		MaxSize:           100,
		DefaultExpiration: 5 * time.Minute,
		CleanupInterval:   10 * time.Minute,
	}
	cache := NewLRUCache(config)
	defer cache.Close()
	ctx := context.Background()

	// Set multiple values
	cache.Set(ctx, "key1", "value1", 0)
	cache.Set(ctx, "key2", "value2", 0)
	cache.Set(ctx, "key3", "value3", 0)

	// Delete multiple values
	err := cache.DeleteMulti(ctx, "key1", "key3")
	assert.NoError(t, err)

	// Verify deletion
	assert.False(t, cache.Exists(ctx, "key1"))
	assert.True(t, cache.Exists(ctx, "key2"))
	assert.False(t, cache.Exists(ctx, "key3"))
}

func TestLRUCache_Increment(t *testing.T) {
	config := LRUCacheConfig{
		MaxSize:           100,
		DefaultExpiration: 5 * time.Minute,
		CleanupInterval:   10 * time.Minute,
	}
	cache := NewLRUCache(config)
	defer cache.Close()
	ctx := context.Background()

	// Increment non-existing key
	val, err := cache.Increment(ctx, "counter", 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), val)

	// Increment again
	val, err = cache.Increment(ctx, "counter", 5)
	assert.NoError(t, err)
	assert.Equal(t, int64(15), val)

	// Verify value
	cachedVal, exists := cache.Get(ctx, "counter")
	assert.True(t, exists)
	counterVal, ok := cachedVal.(int64)
	assert.True(t, ok)
	assert.Equal(t, int64(15), counterVal)
}

func TestLRUCache_Decrement(t *testing.T) {
	config := LRUCacheConfig{
		MaxSize:           100,
		DefaultExpiration: 5 * time.Minute,
		CleanupInterval:   10 * time.Minute,
	}
	cache := NewLRUCache(config)
	defer cache.Close()
	ctx := context.Background()

	// Set initial value
	cache.Set(ctx, "counter", int64(20), 0)

	// Decrement
	val, err := cache.Decrement(ctx, "counter", 5)
	assert.NoError(t, err)
	assert.Equal(t, int64(15), val)

	// Verify value
	cachedVal, exists := cache.Get(ctx, "counter")
	assert.True(t, exists)
	counterVal, ok := cachedVal.(int64)
	assert.True(t, ok)
	assert.Equal(t, int64(15), counterVal)
}

func TestLRUCache_GetWithTTL(t *testing.T) {
	config := LRUCacheConfig{
		MaxSize:           100,
		DefaultExpiration: 5 * time.Minute,
		CleanupInterval:   10 * time.Minute,
	}
	cache := NewLRUCache(config)
	defer cache.Close()
	ctx := context.Background()

	// Set value with expiration
	err := cache.Set(ctx, "key1", "value1", 100*time.Millisecond)
	assert.NoError(t, err)

	// Get with TTL
	val, ttl, exists := cache.GetWithTTL(ctx, "key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", val)
	assert.True(t, ttl > 0)
	assert.True(t, ttl <= 100*time.Millisecond)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Get after expiration
	val, ttl, exists = cache.GetWithTTL(ctx, "key1")
	assert.False(t, exists)
	assert.Nil(t, val)
	assert.Equal(t, time.Duration(0), ttl)
}

func TestLRUCache_DefaultExpiration(t *testing.T) {
	config := LRUCacheConfig{
		MaxSize:           100,
		DefaultExpiration: 50 * time.Millisecond,
		CleanupInterval:   10 * time.Minute,
	}
	cache := NewLRUCache(config)
	defer cache.Close()
	ctx := context.Background()

	// Set value without expiration (should use default)
	err := cache.Set(ctx, "key1", "value1", 0)
	assert.NoError(t, err)

	// Get value immediately
	val, exists := cache.Get(ctx, "key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", val)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Get value after expiration
	val, exists = cache.Get(ctx, "key1")
	assert.False(t, exists)
	assert.Nil(t, val)
}

func TestLRUCache_Eviction(t *testing.T) {
	config := LRUCacheConfig{
		MaxSize:           2,
		DefaultExpiration: 5 * time.Minute,
		CleanupInterval:   10 * time.Minute,
	}
	cache := NewLRUCache(config)
	defer cache.Close()
	ctx := context.Background()

	// Set values up to max size
	cache.Set(ctx, "key1", "value1", 0)
	cache.Set(ctx, "key2", "value2", 0)

	// Access key1 to make it more recently used
	cache.Get(ctx, "key1")

	// Add key3, should evict key2 (least recently used)
	cache.Set(ctx, "key3", "value3", 0)

	// key1 and key3 should exist
	assert.True(t, cache.Exists(ctx, "key1"))
	assert.True(t, cache.Exists(ctx, "key3"))

	// key2 should be evicted
	assert.False(t, cache.Exists(ctx, "key2"))
}

func TestLRUCache_Close(t *testing.T) {
	config := LRUCacheConfig{
		MaxSize:           100,
		DefaultExpiration: 5 * time.Minute,
		CleanupInterval:   10 * time.Millisecond, // Short interval for testing
	}
	cache := NewLRUCache(config)
	ctx := context.Background()

	// Set value
	cache.Set(ctx, "key1", "value1", 0)

	// Close cache
	err := cache.Close()
	assert.NoError(t, err)

	// Wait a bit to ensure cleanup goroutine stops
	time.Sleep(50 * time.Millisecond)

	// Cache should still work after close (cleanup just stops)
	val, exists := cache.Get(ctx, "key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", val)
}
