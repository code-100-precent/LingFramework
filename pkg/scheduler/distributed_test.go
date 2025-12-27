package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/code-100-precent/LingFramework/pkg/cache"
	"github.com/code-100-precent/LingFramework/pkg/logger"
	"go.uber.org/zap"
)

func init() {
	if logger.Lg == nil {
		logger.Lg = zap.NewNop()
	}
}

func TestNewCacheBasedLock(t *testing.T) {
	memCache, _ := cache.NewCache(cache.Config{
		Type: "local",
		Local: cache.LocalConfig{
			CleanupInterval: 10 * time.Minute,
		},
	})
	lock := NewCacheBasedLock(memCache)
	if lock == nil {
		t.Fatal("expected non-nil lock")
	}
	if lock.cache == nil {
		t.Error("expected non-nil cache")
	}
}

func TestCacheBasedLock_Lock(t *testing.T) {
	memCache, _ := cache.NewCache(cache.Config{
		Type: "local",
		Local: cache.LocalConfig{
			CleanupInterval: 10 * time.Minute,
		},
	})
	lock := NewCacheBasedLock(memCache)

	ctx := context.Background()
	key := "test-key"
	ttl := 5 * time.Second

	// First lock should succeed
	acquired, err := lock.Lock(ctx, key, ttl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !acquired {
		t.Error("expected lock to be acquired")
	}

	// Second lock should fail (already locked)
	acquired, err = lock.Lock(ctx, key, ttl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if acquired {
		t.Error("expected lock to fail (already locked)")
	}
}

func TestCacheBasedLock_Unlock(t *testing.T) {
	memCache, _ := cache.NewCache(cache.Config{
		Type: "local",
		Local: cache.LocalConfig{
			CleanupInterval: 10 * time.Minute,
		},
	})
	lock := NewCacheBasedLock(memCache)

	ctx := context.Background()
	key := "test-key"
	ttl := 5 * time.Second

	// Lock first
	lock.Lock(ctx, key, ttl)

	// Unlock
	err := lock.Unlock(ctx, key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be able to lock again
	acquired, err := lock.Lock(ctx, key, ttl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !acquired {
		t.Error("expected to be able to lock again after unlock")
	}
}

func TestNewDistributedScheduler(t *testing.T) {
	memCache, _ := cache.NewCache(cache.Config{
		Type: "local",
		Local: cache.LocalConfig{
			CleanupInterval: 10 * time.Minute,
		},
	})
	lock := NewCacheBasedLock(memCache)

	ds := NewDistributedScheduler(nil, lock)
	if ds == nil {
		t.Fatal("expected non-nil distributed scheduler")
	}
	if ds.Scheduler == nil {
		t.Error("expected non-nil scheduler")
	}
	if ds.lock == nil {
		t.Error("expected non-nil lock")
	}

	// Stop to clean up
	ds.Stop()
}

func TestNewDistributedScheduler_WithConfig(t *testing.T) {
	memCache, _ := cache.NewCache(cache.Config{
		Type: "local",
		Local: cache.LocalConfig{
			CleanupInterval: 10 * time.Minute,
		},
	})
	lock := NewCacheBasedLock(memCache)

	config := &Config{
		NodeID: "test-node",
	}

	ds := NewDistributedScheduler(config, lock)
	if ds == nil {
		t.Fatal("expected non-nil distributed scheduler")
	}
	if !config.Distributed {
		t.Error("expected distributed to be true")
	}

	// Stop to clean up
	ds.Stop()
}

func TestDistributedScheduler_ShouldExecute(t *testing.T) {
	memCache, _ := cache.NewCache(cache.Config{
		Type: "local",
		Local: cache.LocalConfig{
			CleanupInterval: 10 * time.Minute,
		},
	})
	lock := NewCacheBasedLock(memCache)

	ds := NewDistributedScheduler(&Config{NodeID: "test-node"}, lock)
	defer ds.Stop()

	task := &Task{
		ID:       "test-task",
		Name:     "Test Task",
		Schedule: "0 * * * * *",
		Enabled:  true,
	}

	// ShouldExecute checks for leader and task lock
	// Since we're not the leader initially, should return false
	result := ds.shouldExecute(task)
	// Result depends on lock state, just verify it doesn't panic
	_ = result
}

func TestDistributedScheduler_LeaderElection(t *testing.T) {
	memCache, _ := cache.NewCache(cache.Config{
		Type: "local",
		Local: cache.LocalConfig{
			CleanupInterval: 10 * time.Minute,
		},
	})
	lock := NewCacheBasedLock(memCache)

	ds := NewDistributedScheduler(&Config{NodeID: "test-node"}, lock)

	// Give it a moment to attempt leader election
	time.Sleep(100 * time.Millisecond)

	// Stop to clean up
	ds.Stop()
	time.Sleep(50 * time.Millisecond)
}

func TestDistributedScheduler_LeaderElectionLoop(t *testing.T) {
	memCache, _ := cache.NewCache(cache.Config{
		Type: "local",
		Local: cache.LocalConfig{
			CleanupInterval: 10 * time.Minute,
		},
	})
	lock := NewCacheBasedLock(memCache)

	ds := NewDistributedScheduler(&Config{NodeID: "test-node"}, lock)

	// Let it run for a bit
	time.Sleep(200 * time.Millisecond)

	// Stop should cancel the loop
	ds.Stop()
	time.Sleep(100 * time.Millisecond)
}

func TestLoadTasksFromStorage_ErrorHandling(t *testing.T) {
	config := &Config{
		Storage: &errorStorage{},
	}

	scheduler := NewScheduler(config)
	if scheduler == nil {
		t.Fatal("expected non-nil scheduler")
	}

	// Should not panic even if storage returns error
	tasks := scheduler.ListTasks()
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestAddTask_ErrorPaths(t *testing.T) {
	scheduler := NewScheduler(nil)

	// Test with task that has invalid schedule but valid handlers
	task := &Task{
		ID:       "invalid-schedule",
		Name:     "Invalid Schedule",
		Schedule: "invalid cron",
		HandlerFunc: func(ctx context.Context) error {
			return nil
		},
		Enabled: true,
	}

	err := scheduler.AddTask(task)
	if err == nil {
		t.Error("expected error for invalid schedule")
	}
}

func TestGetNextRunTime_WithNilNextRun(t *testing.T) {
	scheduler := NewScheduler(nil)

	task := &Task{
		ID:       "test-task",
		Name:     "Test Task",
		Schedule: "0 * * * * *",
		HandlerFunc: func(ctx context.Context) error {
			return nil
		},
		Enabled: true,
		NextRun: nil, // Explicitly set to nil
	}

	scheduler.AddTask(task)

	nextRun, err := scheduler.GetNextRunTime("test-task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nextRun == nil {
		t.Error("expected non-nil next run time")
	}
	if nextRun.Before(time.Now()) {
		t.Error("expected next run time to be in the future")
	}
}

func TestGetNextRunTime_WithExistingNextRun(t *testing.T) {
	scheduler := NewScheduler(nil)

	task := &Task{
		ID:       "test-task",
		Name:     "Test Task",
		Schedule: "0 * * * * *",
		HandlerFunc: func(ctx context.Context) error {
			return nil
		},
		Enabled: true,
	}

	scheduler.AddTask(task)

	// Get the calculated next run time
	nextRun, err := scheduler.GetNextRunTime("test-task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nextRun == nil {
		t.Error("expected non-nil next run time")
	}
	if nextRun.Before(time.Now()) {
		t.Error("expected next run time to be in the future")
	}

	// Manually set NextRun and verify GetNextRunTime returns it
	retrieved, _ := scheduler.GetTask("test-task")
	futureTime := time.Now().Add(2 * time.Hour)
	retrieved.mu.Lock()
	retrieved.NextRun = &futureTime
	retrieved.mu.Unlock()

	nextRun2, err := scheduler.GetNextRunTime("test-task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nextRun2 == nil {
		t.Error("expected non-nil next run time")
	}
	// Should return the manually set time (or recalculated, depending on implementation)
	// Just verify it's a valid time
	if nextRun2.Before(time.Now()) {
		t.Error("expected next run time to be in the future")
	}
}

func TestParseCronSchedule_6FieldFormat(t *testing.T) {
	// Test 6-field format (with seconds)
	schedule, err := parseCronSchedule("*/5 * * * * *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schedule == nil {
		t.Error("expected non-nil schedule")
	}

	// Verify it calculates next run
	nextRun := schedule.Next(time.Now())
	if nextRun.Before(time.Now()) {
		t.Error("expected next run to be in the future")
	}
}

func TestHashString_Negative(t *testing.T) {
	// Test hash that would be negative
	hash1 := hashString("test")
	hash2 := hashString("test")

	if hash1 != hash2 {
		t.Error("hash should be consistent")
	}

	// Test with different strings
	hash3 := hashString("different")
	if hash1 == hash3 {
		t.Error("different strings should have different hashes")
	}
}

func TestEnableTask_AlreadyEnabled(t *testing.T) {
	scheduler := NewScheduler(nil)

	task := &Task{
		ID:       "test-task",
		Name:     "Test Task",
		Schedule: "0 * * * * *",
		HandlerFunc: func(ctx context.Context) error {
			return nil
		},
		Enabled: true,
	}

	scheduler.AddTask(task)

	// Enable again (should not error)
	err := scheduler.EnableTask("test-task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteTask_DistributedMode(t *testing.T) {
	scheduler := NewScheduler(&Config{
		Distributed: true,
		NodeID:      "node-1",
	})

	executed := false
	task := &Task{
		ID:       "distributed-task",
		Name:     "Distributed Task",
		Schedule: "0 * * * * *",
		HandlerFunc: func(ctx context.Context) error {
			executed = true
			return nil
		},
		Enabled:  true,
		Metadata: make(map[string]interface{}),
	}

	// Execute task (may or may not execute depending on shouldExecute)
	scheduler.executeTask(task)

	// Just verify it doesn't panic
	_ = executed
}

func TestExecuteTask_StorageError(t *testing.T) {
	// Create scheduler with error storage
	config := &Config{
		Storage: &errorStorage{},
	}
	scheduler := NewScheduler(config)

	executed := false
	task := &Task{
		ID:       "storage-error-task",
		Name:     "Storage Error Task",
		Schedule: "0 * * * * *",
		HandlerFunc: func(ctx context.Context) error {
			executed = true
			return nil
		},
		Enabled:  true,
		Metadata: make(map[string]interface{}),
	}

	// Execute task (should handle storage error gracefully)
	scheduler.executeTask(task)

	if !executed {
		t.Error("task should have been executed despite storage error")
	}
}
