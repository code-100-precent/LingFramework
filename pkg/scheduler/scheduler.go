package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/code-100-precent/LingFramework/pkg/logger"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Task represents a scheduled task
type Task struct {
	ID          string                          `json:"id"`
	Name        string                          `json:"name"`
	Schedule    string                          `json:"schedule"` // Cron expression
	Handler     TaskHandler                     `json:"-"`        // Task handlers function
	HandlerFunc func(ctx context.Context) error `json:"-"`        // Alternative handlers
	Enabled     bool                            `json:"enabled"`
	LastRun     *time.Time                      `json:"last_run"`
	NextRun     *time.Time                      `json:"next_run"`
	Metadata    map[string]interface{}          `json:"metadata"`
	mu          sync.RWMutex
}

// TaskHandler is the interface for task handlers
type TaskHandler interface {
	Execute(ctx context.Context) error
}

// Scheduler manages scheduled tasks
type Scheduler struct {
	cron        *cron.Cron
	tasks       map[string]*Task
	mu          sync.RWMutex
	storage     Storage
	distributed bool
	nodeID      string
	ctx         context.Context
	cancel      context.CancelFunc
}

// Storage interface for task persistence
type Storage interface {
	Save(task *Task) error
	Load(id string) (*Task, error)
	LoadAll() ([]*Task, error)
	Delete(id string) error
}

// MemoryStorage is an in-memory storage implementation
type MemoryStorage struct {
	tasks map[string]*Task
	mu    sync.RWMutex
}

// NewMemoryStorage creates a new memory storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		tasks: make(map[string]*Task),
	}
}

// Save saves a task to memory
func (m *MemoryStorage) Save(task *Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks[task.ID] = task
	return nil
}

// Load loads a task from memory
func (m *MemoryStorage) Load(id string) (*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	task, ok := m.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}
	return task, nil
}

// LoadAll loads all tasks from memory
func (m *MemoryStorage) LoadAll() ([]*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tasks := make([]*Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// Delete deletes a task from memory
func (m *MemoryStorage) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tasks, id)
	return nil
}

// Config represents scheduler configuration
type Config struct {
	Storage     Storage
	Distributed bool
	NodeID      string
}

// NewScheduler creates a new scheduler
func NewScheduler(config *Config) *Scheduler {
	if config == nil {
		config = &Config{
			Storage: NewMemoryStorage(),
		}
	}
	if config.Storage == nil {
		config.Storage = NewMemoryStorage()
	}
	if config.NodeID == "" {
		config.NodeID = fmt.Sprintf("node-%d", time.Now().UnixNano())
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &Scheduler{
		cron:        cron.New(cron.WithSeconds()),
		tasks:       make(map[string]*Task),
		storage:     config.Storage,
		distributed: config.Distributed,
		nodeID:      config.NodeID,
		ctx:         ctx,
		cancel:      cancel,
	}

	// Load tasks from storage
	if err := s.loadTasksFromStorage(); err != nil {
		logger.Warn("failed to load tasks from storage", zap.Error(err))
	}

	return s
}

// loadTasksFromStorage loads all tasks from storage
func (s *Scheduler) loadTasksFromStorage() error {
	tasks, err := s.storage.LoadAll()
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if task.Enabled {
			if err := s.AddTask(task); err != nil {
				logger.Warn("failed to add task from storage",
					zap.String("taskID", task.ID),
					zap.Error(err))
			}
		}
	}

	return nil
}

// AddTask adds a task to the scheduler
func (s *Scheduler) AddTask(task *Task) error {
	if task.ID == "" {
		return fmt.Errorf("task ID is required")
	}
	if task.Schedule == "" {
		return fmt.Errorf("task schedule is required")
	}
	if task.Handler == nil && task.HandlerFunc == nil {
		return fmt.Errorf("task handlers is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Parse cron expression (supports both 5 and 6 field formats)
	schedule, err := parseCronSchedule(task.Schedule)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Calculate next run time
	now := time.Now()
	nextRun := schedule.Next(now)
	task.NextRun = &nextRun

	// Store task first (even if disabled)
	task.mu.Lock()
	if task.Metadata == nil {
		task.Metadata = make(map[string]interface{})
	}
	task.mu.Unlock()

	s.tasks[task.ID] = task

	// Only add to cron if enabled
	var entryID cron.EntryID
	if task.Enabled {
		entryID, err = s.cron.AddFunc(task.Schedule, func() {
			s.executeTask(task)
		})
		if err != nil {
			return fmt.Errorf("failed to add cron job: %w", err)
		}

		task.mu.Lock()
		task.Metadata["entryID"] = entryID
		task.mu.Unlock()
	}

	// Persist to storage
	if err := s.storage.Save(task); err != nil {
		logger.Warn("failed to save task to storage",
			zap.String("taskID", task.ID),
			zap.Error(err))
	}

	logger.Info("task added",
		zap.String("taskID", task.ID),
		zap.String("name", task.Name),
		zap.String("schedule", task.Schedule),
		zap.Bool("enabled", task.Enabled))

	return nil
}

// RemoveTask removes a task from the scheduler
func (s *Scheduler) RemoveTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task not found: %s", id)
	}

	// Remove from cron
	if entryID, ok := task.Metadata["entryID"].(cron.EntryID); ok {
		s.cron.Remove(entryID)
	}

	delete(s.tasks, id)

	// Remove from storage
	if err := s.storage.Delete(id); err != nil {
		logger.Warn("failed to delete task from storage",
			zap.String("taskID", id),
			zap.Error(err))
	}

	logger.Info("task removed", zap.String("taskID", id))
	return nil
}

// GetTask gets a task by ID
func (s *Scheduler) GetTask(id string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	return task, nil
}

// ListTasks returns all tasks
func (s *Scheduler) ListTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	s.cron.Start()
	logger.Info("scheduler started", zap.String("nodeID", s.nodeID))
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
	s.cancel()
	logger.Info("scheduler stopped")
}

// executeTask executes a task
func (s *Scheduler) executeTask(task *Task) {
	// Check if task is enabled
	task.mu.RLock()
	enabled := task.Enabled
	task.mu.RUnlock()

	if !enabled {
		return
	}

	// In distributed mode, check if this node should execute
	if s.distributed {
		if !s.shouldExecute(task) {
			return
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

	startTime := time.Now()
	logger.Info("task executing",
		zap.String("taskID", task.ID),
		zap.String("name", task.Name))

	var err error
	if task.Handler != nil {
		err = task.Handler.Execute(ctx)
	} else if task.HandlerFunc != nil {
		err = task.HandlerFunc(ctx)
	}

	duration := time.Since(startTime)

	// Update task metadata
	task.mu.Lock()
	now := time.Now()
	task.LastRun = &now

	// Calculate next run
	if schedule, err2 := parseCronSchedule(task.Schedule); err2 == nil {
		nextRun := schedule.Next(now)
		task.NextRun = &nextRun
	}

	if task.Metadata == nil {
		task.Metadata = make(map[string]interface{})
	}
	task.Metadata["lastDuration"] = duration.String()
	task.Metadata["lastError"] = nil
	if err != nil {
		task.Metadata["lastError"] = err.Error()
	}
	task.mu.Unlock()

	// Persist to storage
	if err2 := s.storage.Save(task); err2 != nil {
		logger.Warn("failed to save task after execution",
			zap.String("taskID", task.ID),
			zap.Error(err2))
	}

	if err != nil {
		logger.Error("task execution failed",
			zap.String("taskID", task.ID),
			zap.String("name", task.Name),
			zap.Duration("duration", duration),
			zap.Error(err))
	} else {
		logger.Info("task executed successfully",
			zap.String("taskID", task.ID),
			zap.String("name", task.Name),
			zap.Duration("duration", duration))
	}
}

// shouldExecute determines if this node should execute the task in distributed mode
func (s *Scheduler) shouldExecute(task *Task) bool {
	// Simple hash-based distribution
	// In production, you might want to use a more sophisticated algorithm
	taskHash := hashString(task.ID)
	nodeHash := hashString(s.nodeID)

	// Use modulo to distribute tasks
	return (taskHash % 100) < (nodeHash%100 + 10) // Simple distribution logic
}

// parseCronSchedule parses cron expression supporting both 5 and 6 field formats
func parseCronSchedule(schedule string) (cron.Schedule, error) {
	// Try standard 5-field format first
	sched, err := cron.ParseStandard(schedule)
	if err == nil {
		return sched, nil
	}

	// Try 6-field format (with seconds)
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	sched, err = parser.Parse(schedule)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cron expression: %w", err)
	}
	return sched, nil
}

// hashString generates a simple hash from string
func hashString(s string) int {
	hash := 0
	for _, char := range s {
		hash = hash*31 + int(char)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

// EnableTask enables a task
func (s *Scheduler) EnableTask(id string) error {
	task, err := s.GetTask(id)
	if err != nil {
		return err
	}

	task.mu.Lock()
	wasEnabled := task.Enabled
	task.Enabled = true
	task.mu.Unlock()

	// If task wasn't enabled, add it to cron
	if !wasEnabled {
		s.mu.Lock()
		entryID, err := s.cron.AddFunc(task.Schedule, func() {
			s.executeTask(task)
		})
		s.mu.Unlock()

		if err != nil {
			return fmt.Errorf("failed to add cron job: %w", err)
		}

		task.mu.Lock()
		task.Metadata["entryID"] = entryID
		task.mu.Unlock()
	}

	// Persist to storage
	return s.storage.Save(task)
}

// DisableTask disables a task
func (s *Scheduler) DisableTask(id string) error {
	task, err := s.GetTask(id)
	if err != nil {
		return err
	}

	task.mu.Lock()
	task.Enabled = false
	// Remove from cron if it was enabled
	if entryID, ok := task.Metadata["entryID"].(cron.EntryID); ok {
		s.mu.Lock()
		s.cron.Remove(entryID)
		s.mu.Unlock()
		delete(task.Metadata, "entryID")
	}
	task.mu.Unlock()

	// Persist to storage
	return s.storage.Save(task)
}

// UpdateTask updates a task
func (s *Scheduler) UpdateTask(task *Task) error {
	// Remove old task
	if err := s.RemoveTask(task.ID); err != nil {
		return err
	}

	// Add updated task
	return s.AddTask(task)
}

// GetNextRunTime gets the next run time for a task
func (s *Scheduler) GetNextRunTime(id string) (*time.Time, error) {
	task, err := s.GetTask(id)
	if err != nil {
		return nil, err
	}

	task.mu.RLock()
	defer task.mu.RUnlock()

	if task.NextRun == nil {
		// Calculate next run
		if schedule, err := parseCronSchedule(task.Schedule); err == nil {
			nextRun := schedule.Next(time.Now())
			return &nextRun, nil
		}
	}

	return task.NextRun, nil
}

// MarshalJSON customizes JSON marshaling for Task
func (t *Task) MarshalJSON() ([]byte, error) {
	type Alias Task
	return json.Marshal(&struct {
		*Alias
		Handler     interface{} `json:"handlers,omitempty"`
		HandlerFunc interface{} `json:"handlerFunc,omitempty"`
	}{
		Alias: (*Alias)(t),
	})
}
