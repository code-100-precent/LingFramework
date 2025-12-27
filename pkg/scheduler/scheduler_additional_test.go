package scheduler

import (
	"context"
	"errors"
	"testing"
)

func TestScheduler_LoadTasksFromStorage(t *testing.T) {
	storage := NewMemoryStorage()

	task := &Task{
		ID:       "persisted-task",
		Name:     "Persisted Task",
		Schedule: "0 * * * * *",
		HandlerFunc: func(ctx context.Context) error {
			return nil
		},
		Enabled: true,
	}

	storage.Save(task)

	config := &Config{
		Storage: storage,
	}
	scheduler := NewScheduler(config)

	// Verify task was loaded
	retrieved, err := scheduler.GetTask("persisted-task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.ID != "persisted-task" {
		t.Errorf("expected task ID 'persisted-task', got '%s'", retrieved.ID)
	}
}

func TestScheduler_ExecuteTask(t *testing.T) {
	scheduler := NewScheduler(nil)

	executed := false
	task := &Task{
		ID:       "exec-task",
		Name:     "Exec Task",
		Schedule: "0 * * * * *",
		HandlerFunc: func(ctx context.Context) error {
			executed = true
			return nil
		},
		Enabled:  true,
		Metadata: make(map[string]interface{}),
	}

	// Manually execute task
	scheduler.executeTask(task)

	if !executed {
		t.Error("task should have been executed")
	}

	// Verify metadata was updated
	if task.LastRun == nil {
		t.Error("expected LastRun to be set")
	}
	if task.NextRun == nil {
		t.Error("expected NextRun to be set")
	}
}

func TestScheduler_ExecuteTask_WithError(t *testing.T) {
	scheduler := NewScheduler(nil)

	task := &Task{
		ID:       "error-task",
		Name:     "Error Task",
		Schedule: "0 * * * * *",
		HandlerFunc: func(ctx context.Context) error {
			return errors.New("task error")
		},
		Enabled:  true,
		Metadata: make(map[string]interface{}),
	}

	// Manually execute task
	scheduler.executeTask(task)

	// Verify error was recorded
	if task.Metadata["lastError"] == nil {
		t.Error("expected error to be recorded in metadata")
	}

	errMsg, ok := task.Metadata["lastError"].(string)
	if !ok || errMsg != "task error" {
		t.Errorf("expected error message 'task error', got %v", errMsg)
	}
}

func TestScheduler_ExecuteTask_Disabled(t *testing.T) {
	scheduler := NewScheduler(nil)

	executed := false
	task := &Task{
		ID:       "disabled-task",
		Name:     "Disabled Task",
		Schedule: "0 * * * * *",
		HandlerFunc: func(ctx context.Context) error {
			executed = true
			return nil
		},
		Enabled:  false,
		Metadata: make(map[string]interface{}),
	}

	// Manually execute task (should not execute because disabled)
	scheduler.executeTask(task)

	if executed {
		t.Error("disabled task should not execute")
	}
}

func TestScheduler_TaskHandler(t *testing.T) {
	scheduler := NewScheduler(nil)

	executed := false
	handler := &testHandler{executed: &executed}

	task := &Task{
		ID:       "handler-task",
		Name:     "Handler Task",
		Schedule: "0 * * * * *",
		Handler:  handler,
		Enabled:  true,
	}

	if err := scheduler.AddTask(task); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Manually execute task
	scheduler.executeTask(task)

	if !executed {
		t.Error("handler should have been executed")
	}
}

type testHandler struct {
	executed *bool
}

func (h *testHandler) Execute(ctx context.Context) error {
	*h.executed = true
	return nil
}

func TestScheduler_AddTask_NoHandler(t *testing.T) {
	scheduler := NewScheduler(nil)

	task := &Task{
		ID:       "no-handler",
		Name:     "No Handler",
		Schedule: "0 * * * * *",
		Enabled:  true,
	}

	err := scheduler.AddTask(task)
	if err == nil {
		t.Error("expected error when task has no handler")
	}
}

func TestScheduler_AddTask_NoID(t *testing.T) {
	scheduler := NewScheduler(nil)

	task := &Task{
		Name:     "No ID",
		Schedule: "0 * * * * *",
		HandlerFunc: func(ctx context.Context) error {
			return nil
		},
	}

	err := scheduler.AddTask(task)
	if err == nil {
		t.Error("expected error when task has no ID")
	}
}

func TestScheduler_AddTask_NoSchedule(t *testing.T) {
	scheduler := NewScheduler(nil)

	task := &Task{
		ID:   "no-schedule",
		Name: "No Schedule",
		HandlerFunc: func(ctx context.Context) error {
			return nil
		},
	}

	err := scheduler.AddTask(task)
	if err == nil {
		t.Error("expected error when task has no schedule")
	}
}

func TestScheduler_GetTask_NotFound(t *testing.T) {
	scheduler := NewScheduler(nil)

	_, err := scheduler.GetTask("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestScheduler_RemoveTask_NotFound(t *testing.T) {
	scheduler := NewScheduler(nil)

	err := scheduler.RemoveTask("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestScheduler_EnableTask_NotFound(t *testing.T) {
	scheduler := NewScheduler(nil)

	err := scheduler.EnableTask("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestScheduler_DisableTask_NotFound(t *testing.T) {
	scheduler := NewScheduler(nil)

	err := scheduler.DisableTask("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestScheduler_GetNextRunTime_NotFound(t *testing.T) {
	scheduler := NewScheduler(nil)

	_, err := scheduler.GetNextRunTime("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestScheduler_UpdateTask_NotFound(t *testing.T) {
	scheduler := NewScheduler(nil)

	task := &Task{
		ID:       "nonexistent",
		Name:     "Nonexistent",
		Schedule: "0 * * * * *",
		HandlerFunc: func(ctx context.Context) error {
			return nil
		},
	}

	err := scheduler.UpdateTask(task)
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestScheduler_ShouldExecute(t *testing.T) {
	scheduler := NewScheduler(&Config{
		Distributed: true,
		NodeID:      "node-1",
	})

	task := &Task{
		ID:   "test-task",
		Name: "Test Task",
	}

	// Test shouldExecute logic
	result := scheduler.shouldExecute(task)
	// Result depends on hash, just check it doesn't panic
	_ = result
}

func TestScheduler_ShouldExecute_Consistency(t *testing.T) {
	// Test shouldExecute consistency
	scheduler := NewScheduler(&Config{
		Distributed: true,
		NodeID:      "node-1",
	})

	task1 := &Task{ID: "test"}
	task2 := &Task{ID: "test"}
	task3 := &Task{ID: "different"}

	result1 := scheduler.shouldExecute(task1)
	result2 := scheduler.shouldExecute(task2)
	result3 := scheduler.shouldExecute(task3)

	// Same task should have same result
	if result1 != result2 {
		t.Error("same task should have same execution result")
	}

	// Different tasks may have different results (depends on hash)
	_ = result3
}

func TestTask_MarshalJSON(t *testing.T) {
	task := &Task{
		ID:       "json-task",
		Name:     "JSON Task",
		Schedule: "0 * * * * *",
		Enabled:  true,
	}

	data, err := task.MarshalJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty JSON data")
	}

	// Verify it doesn't include handler fields
	jsonStr := string(data)
	if contains(jsonStr, "Handler") || contains(jsonStr, "HandlerFunc") {
		t.Error("JSON should not include handler fields")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsHelper(s, substr))))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestScheduler_ContextCancellation(t *testing.T) {
	scheduler := NewScheduler(nil)
	scheduler.Start()

	// Stop should cancel context
	scheduler.Stop()

	// Context should be cancelled
	select {
	case <-scheduler.ctx.Done():
		// Expected
	default:
		t.Error("context should be cancelled after Stop")
	}
}

func TestScheduler_LoadTasksFromStorage_Error(t *testing.T) {
	// Test with storage that returns error
	config := &Config{
		Storage: &errorStorage{},
	}

	// Should not panic
	scheduler := NewScheduler(config)
	if scheduler == nil {
		t.Fatal("expected non-nil scheduler")
	}
}

type errorStorage struct{}

func (e *errorStorage) Save(task *Task) error {
	return nil
}

func (e *errorStorage) Load(id string) (*Task, error) {
	return nil, errors.New("storage error")
}

func (e *errorStorage) LoadAll() ([]*Task, error) {
	return nil, errors.New("storage error")
}

func (e *errorStorage) Delete(id string) error {
	return nil
}
