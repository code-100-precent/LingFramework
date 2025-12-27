package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/code-100-precent/LingFramework/pkg/logger"
	"go.uber.org/zap"
)

func init() {
	// Initialize logger for tests
	if logger.Lg == nil {
		logger.Lg = zap.NewNop() // Use no-op logger for tests
	}
}

func TestNewScheduler(t *testing.T) {
	scheduler := NewScheduler(nil)
	if scheduler == nil {
		t.Fatal("expected non-nil scheduler")
	}
	if scheduler.storage == nil {
		t.Error("expected non-nil storage")
	}
}

func TestScheduler_AddTask(t *testing.T) {
	scheduler := NewScheduler(nil)

	task := &Task{
		ID:       "test-task",
		Name:     "Test Task",
		Schedule: "0 * * * * *", // Every minute
		HandlerFunc: func(ctx context.Context) error {
			return nil
		},
		Enabled: true,
	}

	if err := scheduler.AddTask(task); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify task was added
	retrieved, err := scheduler.GetTask("test-task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.ID != "test-task" {
		t.Errorf("expected task ID 'test-task', got '%s'", retrieved.ID)
	}
}

func TestScheduler_AddTask_InvalidSchedule(t *testing.T) {
	scheduler := NewScheduler(nil)

	task := &Task{
		ID:       "test-task",
		Name:     "Test Task",
		Schedule: "invalid",
		HandlerFunc: func(ctx context.Context) error {
			return nil
		},
	}

	err := scheduler.AddTask(task)
	if err == nil {
		t.Error("expected error for invalid schedule")
	}
}

func TestScheduler_RemoveTask(t *testing.T) {
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

	if err := scheduler.RemoveTask("test-task"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify task was removed
	_, err := scheduler.GetTask("test-task")
	if err == nil {
		t.Error("expected error for removed task")
	}
}

func TestScheduler_ListTasks(t *testing.T) {
	scheduler := NewScheduler(nil)

	task1 := &Task{
		ID:       "task-1",
		Name:     "Task 1",
		Schedule: "0 * * * * *",
		HandlerFunc: func(ctx context.Context) error {
			return nil
		},
		Enabled: true,
	}

	task2 := &Task{
		ID:       "task-2",
		Name:     "Task 2",
		Schedule: "0 * * * * *",
		HandlerFunc: func(ctx context.Context) error {
			return nil
		},
		Enabled: true,
	}

	scheduler.AddTask(task1)
	scheduler.AddTask(task2)

	tasks := scheduler.ListTasks()
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestScheduler_EnableDisableTask(t *testing.T) {
	scheduler := NewScheduler(nil)

	task := &Task{
		ID:       "test-task",
		Name:     "Test Task",
		Schedule: "0 * * * * *",
		HandlerFunc: func(ctx context.Context) error {
			return nil
		},
		Enabled: false,
	}

	scheduler.AddTask(task)

	if err := scheduler.EnableTask("test-task"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, _ := scheduler.GetTask("test-task")
	if !retrieved.Enabled {
		t.Error("expected task to be enabled")
	}

	if err := scheduler.DisableTask("test-task"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, _ = scheduler.GetTask("test-task")
	if retrieved.Enabled {
		t.Error("expected task to be disabled")
	}
}

func TestScheduler_StartStop(t *testing.T) {
	scheduler := NewScheduler(nil)

	scheduler.Start()
	time.Sleep(100 * time.Millisecond)

	scheduler.Stop()
	time.Sleep(100 * time.Millisecond)
}

func TestMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage()

	task := &Task{
		ID:       "test-task",
		Name:     "Test Task",
		Schedule: "0 * * * * *",
		Enabled:  true,
	}

	// Test Save
	if err := storage.Save(task); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test Load
	loaded, err := storage.Load("test-task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded.ID != "test-task" {
		t.Errorf("expected task ID 'test-task', got '%s'", loaded.ID)
	}

	// Test LoadAll
	all, err := storage.LoadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 task, got %d", len(all))
	}

	// Test Delete
	if err := storage.Delete("test-task"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = storage.Load("test-task")
	if err == nil {
		t.Error("expected error for deleted task")
	}
}

func TestScheduler_GetNextRunTime(t *testing.T) {
	scheduler := NewScheduler(nil)

	task := &Task{
		ID:       "test-task",
		Name:     "Test Task",
		Schedule: "0 * * * * *", // Every minute
		HandlerFunc: func(ctx context.Context) error {
			return nil
		},
		Enabled: true,
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

func TestScheduler_UpdateTask(t *testing.T) {
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

	// Update task
	task.Name = "Updated Task"
	task.Schedule = "0 0 * * * *" // Every hour

	if err := scheduler.UpdateTask(task); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, _ := scheduler.GetTask("test-task")
	if retrieved.Name != "Updated Task" {
		t.Errorf("expected name 'Updated Task', got '%s'", retrieved.Name)
	}
}
