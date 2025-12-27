package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/code-100-precent/LingFramework/pkg/cache"
	"github.com/code-100-precent/LingFramework/pkg/logger"
	"go.uber.org/zap"
)

// DistributedLock provides distributed locking for task execution
type DistributedLock interface {
	Lock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Unlock(ctx context.Context, key string) error
}

// CacheBasedLock implements distributed lock using cache
type CacheBasedLock struct {
	cache cache.Cache
	mu    sync.Mutex
}

// NewCacheBasedLock creates a new cache-based lock
func NewCacheBasedLock(c cache.Cache) *CacheBasedLock {
	return &CacheBasedLock{
		cache: c,
	}
}

// Lock attempts to acquire a lock
func (l *CacheBasedLock) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	lockKey := fmt.Sprintf("scheduler:lock:%s", key)

	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if lock already exists
	if l.cache.Exists(ctx, lockKey) {
		return false, nil
	}

	// Try to set lock with TTL
	if err := l.cache.Set(ctx, lockKey, "locked", ttl); err != nil {
		return false, err
	}

	return true, nil
}

// Unlock releases a lock
func (l *CacheBasedLock) Unlock(ctx context.Context, key string) error {
	lockKey := fmt.Sprintf("scheduler:lock:%s", key)
	return l.cache.Delete(ctx, lockKey)
}

// DistributedScheduler extends Scheduler with distributed execution
type DistributedScheduler struct {
	*Scheduler
	lock           DistributedLock
	leaderElection bool
	leaderTTL      time.Duration
}

// NewDistributedScheduler creates a distributed scheduler
func NewDistributedScheduler(config *Config, lock DistributedLock) *DistributedScheduler {
	if config == nil {
		config = &Config{
			Distributed: true,
		}
	}
	config.Distributed = true

	ds := &DistributedScheduler{
		Scheduler:      NewScheduler(config),
		lock:           lock,
		leaderElection: true,
		leaderTTL:      30 * time.Second,
	}

	// Start leader election if enabled
	if ds.leaderElection {
		go ds.leaderElectionLoop()
	}

	return ds
}

// leaderElectionLoop performs leader election
func (ds *DistributedScheduler) leaderElectionLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ds.ctx.Done():
			return
		case <-ticker.C:
			ds.electLeader()
		}
	}
}

// electLeader attempts to become the leader
func (ds *DistributedScheduler) electLeader() {
	leaderKey := fmt.Sprintf("scheduler:leader:%s", ds.nodeID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	acquired, err := ds.lock.Lock(ctx, leaderKey, ds.leaderTTL)
	if err != nil {
		logger.Warn("leader election failed", zap.Error(err))
		return
	}

	if acquired {
		logger.Info("elected as leader", zap.String("nodeID", ds.nodeID))
	}
}

// shouldExecute checks if this node should execute the task
func (ds *DistributedScheduler) shouldExecute(task *Task) bool {
	// Check if we're the leader
	leaderKey := fmt.Sprintf("scheduler:leader:%s", ds.nodeID)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	acquired, err := ds.lock.Lock(ctx, leaderKey, ds.leaderTTL)
	if err != nil {
		return false
	}

	if !acquired {
		return false
	}

	// Try to acquire task lock
	taskLockKey := fmt.Sprintf("scheduler:task:%s", task.ID)
	acquired, err = ds.lock.Lock(ctx, taskLockKey, 5*time.Minute)
	if err != nil {
		return false
	}

	return acquired
}
