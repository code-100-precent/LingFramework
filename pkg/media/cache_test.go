package media

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/code-100-precent/LingFramework/pkg/logger"
	"go.uber.org/zap"
)

func init() {
	// Initialize logger for tests
	if logger.Lg == nil {
		logger.Lg = zap.NewNop() // Use no-op logger for tests
	}
}

func TestMediaCache(t *testing.T) {
	// Clear default cache
	_defaultMediaCache = nil

	// Set environment to avoid logger initialization issues
	os.Setenv("MEDIA_CACHE_ROOT", "/tmp/test_cache")
	defer os.Unsetenv("MEDIA_CACHE_ROOT")

	cache := MediaCache()
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}

	// Should use default /tmp if env not set
	if cache.CacheRoot == "" {
		t.Error("expected non-empty cache root")
	}
}

func TestLocalMediaCache_BuildKey(t *testing.T) {
	cache := &LocalMediaCache{}

	key1 := cache.BuildKey("param1", "param2", "param3")
	if key1 == "" {
		t.Error("expected non-empty key")
	}

	// Same params should produce same key
	key2 := cache.BuildKey("param1", "param2", "param3")
	if key1 != key2 {
		t.Error("expected same key for same params")
	}

	// Different params should produce different key
	key3 := cache.BuildKey("param1", "param2", "param4")
	if key1 == key3 {
		t.Error("expected different key for different params")
	}

	// Empty params
	key4 := cache.BuildKey()
	if key4 == "" {
		t.Error("expected non-empty key even for empty params")
	}
}

func TestLocalMediaCache_Store_Get(t *testing.T) {
	// Create temporary directory
	tmpDir := filepath.Join(os.TempDir(), "media_cache_test")
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	cache := &LocalMediaCache{
		Disabled:  false,
		CacheRoot: tmpDir,
	}

	key := cache.BuildKey("test", "data")
	data := []byte{1, 2, 3, 4, 5}

	// Store data
	err := cache.Store(key, data)
	if err != nil {
		t.Fatalf("unexpected error storing: %v", err)
	}

	// Retrieve data
	retrieved, err := cache.Get(key)
	if err != nil {
		t.Fatalf("unexpected error getting: %v", err)
	}

	if len(retrieved) != len(data) {
		t.Errorf("expected data length %d, got %d", len(data), len(retrieved))
	}

	for i := range data {
		if retrieved[i] != data[i] {
			t.Errorf("expected data[%d] = %d, got %d", i, data[i], retrieved[i])
		}
	}
}

func TestLocalMediaCache_Store_Disabled(t *testing.T) {
	cache := &LocalMediaCache{
		Disabled: true,
	}

	err := cache.Store("key", []byte{1, 2, 3})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLocalMediaCache_Get_Disabled(t *testing.T) {
	cache := &LocalMediaCache{
		Disabled: true,
	}

	data, err := cache.Get("key")
	if err == nil {
		t.Error("expected error when cache is disabled")
	}
	if data != nil {
		t.Error("expected nil data when cache is disabled")
	}
}

func TestLocalMediaCache_Get_NotExist(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "media_cache_test")
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	cache := &LocalMediaCache{
		Disabled:  false,
		CacheRoot: tmpDir,
	}

	data, err := cache.Get("nonexistent_key")
	if err == nil {
		t.Error("expected error for non-existent key")
	}
	if data != nil {
		t.Error("expected nil data for non-existent key")
	}
}

func TestLocalMediaCache_Store_Overwrite(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "media_cache_test")
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	cache := &LocalMediaCache{
		Disabled:  false,
		CacheRoot: tmpDir,
	}

	key := cache.BuildKey("test")
	data1 := []byte{1, 2, 3}
	data2 := []byte{4, 5, 6}

	// Store first data
	err := cache.Store(key, data1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Overwrite with second data
	err = cache.Store(key, data2)
	if err != nil {
		t.Fatalf("unexpected error overwriting: %v", err)
	}

	// Retrieve should get second data
	retrieved, err := cache.Get(key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(retrieved) != len(data2) {
		t.Errorf("expected data length %d, got %d", len(data2), len(retrieved))
	}

	for i := range data2 {
		if retrieved[i] != data2[i] {
			t.Errorf("expected data[%d] = %d, got %d", i, data2[i], retrieved[i])
		}
	}
}

func TestLocalMediaCache_Store_DirectoryExists(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "media_cache_test")
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	cache := &LocalMediaCache{
		Disabled:  false,
		CacheRoot: tmpDir,
	}

	// Create a directory with the same name as the key
	key := cache.BuildKey("test")
	keyDir := filepath.Join(tmpDir, key)
	os.MkdirAll(keyDir, 0755)

	// Should return error when trying to store to a directory
	err := cache.Store(key, []byte{1, 2, 3})
	if err == nil {
		t.Error("expected error when key is a directory")
	}
}

func TestLocalMediaCache_Get_DirectoryExists(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "media_cache_test")
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	cache := &LocalMediaCache{
		Disabled:  false,
		CacheRoot: tmpDir,
	}

	// Create a directory with the same name as the key
	key := cache.BuildKey("test")
	keyDir := filepath.Join(tmpDir, key)
	os.MkdirAll(keyDir, 0755)

	// Should return error when key is a directory
	data, err := cache.Get(key)
	if err == nil {
		t.Error("expected error when key is a directory")
	}
	if data != nil {
		t.Error("expected nil data when key is a directory")
	}
}
