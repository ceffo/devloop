package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ceffo/devloop/internal/config"
)

// Helper function to create a test config with a temporary directory
func createTestConfig(t *testing.T) (*config.Config, string) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Version: "1.0",
		Project: config.ProjectConfig{
			Name:       "test-project",
			Path:       tmpDir,
			TechStack:  "Go",
			MainBranch: "main",
		},
		Verification: config.VerificationConfig{
			Command:        "go test",
			TimeoutSeconds: 300,
		},
		CLI: config.CLIConfig{
			Tool: "claude",
			Models: map[string]string{
				"simple":   "claude-haiku-4-5-20251001",
				"moderate": "claude-sonnet-4-5-20250929",
				"complex":  "claude-opus-4-6",
			},
		},
		Execution: config.ExecutionConfig{
			MaxAttempts:   2,
			HaltOnFailure: true,
			AutoCommit:    true,
		},
	}

	return cfg, tmpDir
}

// Helper function to create a sample task
func createSampleTask(id, title, status string) *Task {
	now := time.Now()
	return &Task{
		ID:          id,
		Title:       title,
		Wave:        1,
		Status:      status,
		Complexity:  "simple",
		Model:       "claude-haiku-4-5-20251001",
		Description: "Test task description",
		AcceptanceCriteria: []string{
			"Criterion 1",
			"Criterion 2",
		},
		BlockedBy: []string{},
		Tags:      []string{"test"},
		Metadata: TaskMetadata{
			CreatedAt:   now,
			UpdatedAt:   now,
			SourceType:  "manual",
			MaxAttempts: 2,
		},
		Execution: TaskExecution{
			Attempts:      []Attempt{},
			TotalDuration: 0,
		},
	}
}

func TestNewStorage(t *testing.T) {
	cfg, tmpDir := createTestConfig(t)
	storage := NewStorage(cfg)

	if storage == nil {
		t.Fatal("NewStorage returned nil")
	}

	expectedTasksDir := filepath.Join(tmpDir, ".devloop")
	if storage.tasksDir != expectedTasksDir {
		t.Errorf("Expected tasksDir %s, got %s", expectedTasksDir, storage.tasksDir)
	}

	if storage.cfg != cfg {
		t.Error("Config not properly stored in Storage struct")
	}
}

func TestLoadTasks_NonExistentFile(t *testing.T) {
	cfg, _ := createTestConfig(t)
	storage := NewStorage(cfg)

	tasks, err := storage.LoadTasks()
	if err != nil {
		t.Errorf("LoadTasks should not error on non-existent file: %v", err)
	}

	if len(tasks) != 0 {
		t.Errorf("Expected empty task list, got %d tasks", len(tasks))
	}
}

func TestSaveTask(t *testing.T) {
	cfg, _ := createTestConfig(t)
	storage := NewStorage(cfg)

	task := createSampleTask("1.1", "Test Task", "pending")

	err := storage.SaveTask(task)
	if err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	// Verify file exists
	tasksFile := storage.getTasksFilePath()
	if _, err := os.Stat(tasksFile); os.IsNotExist(err) {
		t.Error("Tasks file was not created")
	}

	// Verify file permissions
	info, err := os.Stat(tasksFile)
	if err != nil {
		t.Fatalf("Failed to stat tasks file: %v", err)
	}

	expectedMode := os.FileMode(0644)
	if info.Mode().Perm() != expectedMode {
		t.Errorf("Expected file permissions %v, got %v", expectedMode, info.Mode().Perm())
	}
}

func TestSaveAndLoadTasks(t *testing.T) {
	cfg, _ := createTestConfig(t)
	storage := NewStorage(cfg)

	// Create and save multiple tasks
	tasks := []*Task{
		createSampleTask("1.1", "Task One", "pending"),
		createSampleTask("1.2", "Task Two", "in_progress"),
		createSampleTask("2.1", "Task Three", "completed"),
	}

	for _, task := range tasks {
		if err := storage.SaveTask(task); err != nil {
			t.Fatalf("SaveTask failed: %v", err)
		}
	}

	// Load tasks
	loadedTasks, err := storage.LoadTasks()
	if err != nil {
		t.Fatalf("LoadTasks failed: %v", err)
	}

	if len(loadedTasks) != len(tasks) {
		t.Errorf("Expected %d tasks, got %d", len(tasks), len(loadedTasks))
	}

	// Verify task data
	for i, task := range loadedTasks {
		if task.ID != tasks[i].ID {
			t.Errorf("Task %d: expected ID %s, got %s", i, tasks[i].ID, task.ID)
		}
		if task.Title != tasks[i].Title {
			t.Errorf("Task %d: expected title %s, got %s", i, tasks[i].Title, task.Title)
		}
		if task.Status != tasks[i].Status {
			t.Errorf("Task %d: expected status %s, got %s", i, tasks[i].Status, task.Status)
		}
	}
}

func TestGetTask(t *testing.T) {
	cfg, _ := createTestConfig(t)
	storage := NewStorage(cfg)

	// Save some tasks
	task1 := createSampleTask("1.1", "Task One", "pending")
	task2 := createSampleTask("1.2", "Task Two", "completed")

	storage.SaveTask(task1)
	storage.SaveTask(task2)

	// Test getting existing task
	found, err := storage.GetTask("1.1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if found.ID != task1.ID {
		t.Errorf("Expected task ID %s, got %s", task1.ID, found.ID)
	}

	if found.Title != task1.Title {
		t.Errorf("Expected task title %s, got %s", task1.Title, found.Title)
	}

	// Test getting non-existent task
	_, err = storage.GetTask("99.99")
	if err == nil {
		t.Error("GetTask should error for non-existent task")
	}
}

func TestUpdateTask(t *testing.T) {
	cfg, _ := createTestConfig(t)
	storage := NewStorage(cfg)

	// Save initial tasks
	task1 := createSampleTask("1.1", "Task One", "pending")
	task2 := createSampleTask("1.2", "Task Two", "pending")

	storage.SaveTask(task1)
	storage.SaveTask(task2)

	// Update task1
	task1.Status = "completed"
	task1.Title = "Updated Task One"
	task1.Metadata.UpdatedAt = time.Now()

	err := storage.UpdateTask(task1)
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	// Verify update
	updated, err := storage.GetTask("1.1")
	if err != nil {
		t.Fatalf("GetTask after update failed: %v", err)
	}

	if updated.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", updated.Status)
	}

	if updated.Title != "Updated Task One" {
		t.Errorf("Expected title 'Updated Task One', got %s", updated.Title)
	}

	// Verify other task was not affected
	other, err := storage.GetTask("1.2")
	if err != nil {
		t.Fatalf("GetTask for other task failed: %v", err)
	}

	if other.Status != "pending" {
		t.Errorf("Other task status should remain 'pending', got %s", other.Status)
	}
}

func TestUpdateTask_NonExistent(t *testing.T) {
	cfg, _ := createTestConfig(t)
	storage := NewStorage(cfg)

	// Try to update a non-existent task
	task := createSampleTask("99.99", "Non-existent", "pending")

	err := storage.UpdateTask(task)
	if err == nil {
		t.Error("UpdateTask should error for non-existent task")
	}
}

func TestLoadTasks_SkipsEmptyLines(t *testing.T) {
	cfg, tmpDir := createTestConfig(t)
	storage := NewStorage(cfg)

	// Create tasks file with empty lines
	tasksDir := filepath.Join(tmpDir, ".devloop")
	os.MkdirAll(tasksDir, 0755)

	tasksFile := filepath.Join(tasksDir, "tasks.jsonl")
	content := `{"id":"1.1","title":"Task One","wave":1,"status":"pending","complexity":"simple","model":"test","description":"Test","metadata":{"created_at":"2026-02-15T10:00:00Z","updated_at":"2026-02-15T10:00:00Z","source_type":"manual","max_attempts":2},"execution":{"total_duration":0}}

{"id":"1.2","title":"Task Two","wave":1,"status":"pending","complexity":"simple","model":"test","description":"Test","metadata":{"created_at":"2026-02-15T10:00:00Z","updated_at":"2026-02-15T10:00:00Z","source_type":"manual","max_attempts":2},"execution":{"total_duration":0}}
`
	os.WriteFile(tasksFile, []byte(content), 0644)

	// Load tasks
	tasks, err := storage.LoadTasks()
	if err != nil {
		t.Fatalf("LoadTasks failed: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks (empty lines should be skipped), got %d", len(tasks))
	}
}

func TestUpdateTask_AtomicWrite(t *testing.T) {
	cfg, _ := createTestConfig(t)
	storage := NewStorage(cfg)

	// Save initial task
	task := createSampleTask("1.1", "Task One", "pending")
	storage.SaveTask(task)

	// Update task
	task.Status = "completed"
	err := storage.UpdateTask(task)
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	// Verify no .tmp file remains
	tmpFile := storage.getTasksFilePath() + ".tmp"
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("Temporary file was not cleaned up")
	}
}

func TestSaveTask_CreatesDirectory(t *testing.T) {
	cfg, tmpDir := createTestConfig(t)
	storage := NewStorage(cfg)

	// Ensure .devloop directory doesn't exist
	devloopDir := filepath.Join(tmpDir, ".devloop")
	os.RemoveAll(devloopDir)

	// Save task should create directory
	task := createSampleTask("1.1", "Task One", "pending")
	err := storage.SaveTask(task)
	if err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(devloopDir); os.IsNotExist(err) {
		t.Error(".devloop directory was not created")
	}

	// Verify directory permissions
	info, err := os.Stat(devloopDir)
	if err != nil {
		t.Fatalf("Failed to stat .devloop directory: %v", err)
	}

	expectedMode := os.FileMode(0755)
	if info.Mode().Perm() != expectedMode {
		t.Errorf("Expected directory permissions %v, got %v", expectedMode, info.Mode().Perm())
	}
}

func TestQueryTasks_NoFilter(t *testing.T) {
	cfg, _ := createTestConfig(t)
	storage := NewStorage(cfg)

	// Create and save tasks
	tasks := []*Task{
		createSampleTask("1.1", "Task One", "pending"),
		createSampleTask("2.3", "Task Two", "completed"),
		createSampleTask("1.2", "Task Three", "in_progress"),
	}

	for _, task := range tasks {
		storage.SaveTask(task)
	}

	// Query all tasks
	results, err := storage.QueryTasks(Filter{})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(results))
	}

	// Verify sorting by ID (1.1, 1.2, 2.3)
	expectedOrder := []string{"1.1", "1.2", "2.3"}
	for i, task := range results {
		if task.ID != expectedOrder[i] {
			t.Errorf("Task %d: expected ID %s, got %s", i, expectedOrder[i], task.ID)
		}
	}
}

func TestQueryTasks_FilterByStatus(t *testing.T) {
	cfg, _ := createTestConfig(t)
	storage := NewStorage(cfg)

	// Create tasks with different statuses
	tasks := []*Task{
		createSampleTask("1.1", "Task One", "pending"),
		createSampleTask("1.2", "Task Two", "completed"),
		createSampleTask("1.3", "Task Three", "pending"),
		createSampleTask("1.4", "Task Four", "in_progress"),
	}

	for _, task := range tasks {
		storage.SaveTask(task)
	}

	// Query pending tasks
	results, err := storage.QueryTasks(Filter{Status: "pending"})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 pending tasks, got %d", len(results))
	}

	for _, task := range results {
		if task.Status != "pending" {
			t.Errorf("Expected status 'pending', got %s", task.Status)
		}
	}
}

func TestQueryTasks_FilterByWave(t *testing.T) {
	cfg, _ := createTestConfig(t)
	storage := NewStorage(cfg)

	// Create tasks in different waves
	task1 := createSampleTask("1.1", "Task One", "pending")
	task1.Wave = 1

	task2 := createSampleTask("2.1", "Task Two", "pending")
	task2.Wave = 2

	task3 := createSampleTask("1.2", "Task Three", "pending")
	task3.Wave = 1

	storage.SaveTask(task1)
	storage.SaveTask(task2)
	storage.SaveTask(task3)

	// Query wave 1 tasks
	results, err := storage.QueryTasks(Filter{Wave: 1})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 wave 1 tasks, got %d", len(results))
	}

	for _, task := range results {
		if task.Wave != 1 {
			t.Errorf("Expected wave 1, got %d", task.Wave)
		}
	}
}

func TestQueryTasks_FilterByComplexity(t *testing.T) {
	cfg, _ := createTestConfig(t)
	storage := NewStorage(cfg)

	// Create tasks with different complexities
	task1 := createSampleTask("1.1", "Task One", "pending")
	task1.Complexity = "simple"

	task2 := createSampleTask("1.2", "Task Two", "pending")
	task2.Complexity = "complex"

	task3 := createSampleTask("1.3", "Task Three", "pending")
	task3.Complexity = "simple"

	storage.SaveTask(task1)
	storage.SaveTask(task2)
	storage.SaveTask(task3)

	// Query simple complexity tasks
	results, err := storage.QueryTasks(Filter{Complexity: "simple"})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 simple tasks, got %d", len(results))
	}

	for _, task := range results {
		if task.Complexity != "simple" {
			t.Errorf("Expected complexity 'simple', got %s", task.Complexity)
		}
	}
}

func TestQueryTasks_FilterByTags(t *testing.T) {
	cfg, _ := createTestConfig(t)
	storage := NewStorage(cfg)

	// Create tasks with different tags
	task1 := createSampleTask("1.1", "Task One", "pending")
	task1.Tags = []string{"backend", "api"}

	task2 := createSampleTask("1.2", "Task Two", "pending")
	task2.Tags = []string{"frontend", "ui"}

	task3 := createSampleTask("1.3", "Task Three", "pending")
	task3.Tags = []string{"backend", "database"}

	task4 := createSampleTask("1.4", "Task Four", "pending")
	task4.Tags = []string{"docs"}

	storage.SaveTask(task1)
	storage.SaveTask(task2)
	storage.SaveTask(task3)
	storage.SaveTask(task4)

	// Query tasks with "backend" tag (OR match)
	results, err := storage.QueryTasks(Filter{Tags: []string{"backend"}})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 backend tasks, got %d", len(results))
	}

	// Query tasks with "backend" OR "ui" tags
	results, err = storage.QueryTasks(Filter{Tags: []string{"backend", "ui"}})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 tasks (backend OR ui), got %d", len(results))
	}
}

func TestQueryTasks_FilterUnblocked(t *testing.T) {
	cfg, _ := createTestConfig(t)
	storage := NewStorage(cfg)

	// Create tasks with different blocked states
	task1 := createSampleTask("1.1", "Task One", "pending")
	task1.BlockedBy = []string{}

	task2 := createSampleTask("1.2", "Task Two", "pending")
	task2.BlockedBy = []string{"1.1"}

	task3 := createSampleTask("1.3", "Task Three", "pending")
	task3.BlockedBy = []string{}

	task4 := createSampleTask("1.4", "Task Four", "pending")
	task4.BlockedBy = []string{"1.1", "1.2"}

	storage.SaveTask(task1)
	storage.SaveTask(task2)
	storage.SaveTask(task3)
	storage.SaveTask(task4)

	// Query unblocked tasks (empty BlockedBy slice in filter)
	results, err := storage.QueryTasks(Filter{BlockedBy: []string{}})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 unblocked tasks, got %d", len(results))
	}

	for _, task := range results {
		if len(task.BlockedBy) > 0 {
			t.Errorf("Expected unblocked task, got task blocked by %v", task.BlockedBy)
		}
	}
}

func TestQueryTasks_FilterByBlockedBy(t *testing.T) {
	cfg, _ := createTestConfig(t)
	storage := NewStorage(cfg)

	// Create tasks with different blocked states
	task1 := createSampleTask("1.1", "Task One", "pending")
	task1.BlockedBy = []string{"1.0"}

	task2 := createSampleTask("1.2", "Task Two", "pending")
	task2.BlockedBy = []string{"1.1"}

	task3 := createSampleTask("1.3", "Task Three", "pending")
	task3.BlockedBy = []string{"1.1", "1.2"}

	task4 := createSampleTask("1.4", "Task Four", "pending")
	task4.BlockedBy = []string{"2.0"}

	storage.SaveTask(task1)
	storage.SaveTask(task2)
	storage.SaveTask(task3)
	storage.SaveTask(task4)

	// Query tasks blocked by "1.1"
	results, err := storage.QueryTasks(Filter{BlockedBy: []string{"1.1"}})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 tasks blocked by 1.1, got %d", len(results))
	}

	for _, task := range results {
		hasBlocker := false
		for _, blocker := range task.BlockedBy {
			if blocker == "1.1" {
				hasBlocker = true
				break
			}
		}
		if !hasBlocker {
			t.Errorf("Expected task blocked by 1.1, got task blocked by %v", task.BlockedBy)
		}
	}
}

func TestQueryTasks_CombinedFilters(t *testing.T) {
	cfg, _ := createTestConfig(t)
	storage := NewStorage(cfg)

	// Create diverse tasks
	task1 := createSampleTask("1.1", "Task One", "pending")
	task1.Wave = 1
	task1.Complexity = "simple"
	task1.Tags = []string{"backend"}
	task1.BlockedBy = []string{}

	task2 := createSampleTask("1.2", "Task Two", "pending")
	task2.Wave = 1
	task2.Complexity = "complex"
	task2.Tags = []string{"backend"}
	task2.BlockedBy = []string{}

	task3 := createSampleTask("1.3", "Task Three", "completed")
	task3.Wave = 1
	task3.Complexity = "simple"
	task3.Tags = []string{"backend"}
	task3.BlockedBy = []string{}

	task4 := createSampleTask("2.1", "Task Four", "pending")
	task4.Wave = 2
	task4.Complexity = "simple"
	task4.Tags = []string{"backend"}
	task4.BlockedBy = []string{}

	storage.SaveTask(task1)
	storage.SaveTask(task2)
	storage.SaveTask(task3)
	storage.SaveTask(task4)

	// Query: wave=1, status=pending, complexity=simple, unblocked
	results, err := storage.QueryTasks(Filter{
		Wave:       1,
		Status:     "pending",
		Complexity: "simple",
		BlockedBy:  []string{},
	})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 task matching all filters, got %d", len(results))
	}

	if len(results) > 0 && results[0].ID != "1.1" {
		t.Errorf("Expected task 1.1, got %s", results[0].ID)
	}
}

func TestQueryTasks_IDSorting(t *testing.T) {
	cfg, _ := createTestConfig(t)
	storage := NewStorage(cfg)

	// Create tasks in random order with hierarchical IDs
	tasks := []*Task{
		createSampleTask("2.3", "Task", "pending"),
		createSampleTask("1.10", "Task", "pending"),
		createSampleTask("1.2", "Task", "pending"),
		createSampleTask("10.1", "Task", "pending"),
		createSampleTask("1.1", "Task", "pending"),
		createSampleTask("2.1", "Task", "pending"),
	}

	for _, task := range tasks {
		storage.SaveTask(task)
	}

	// Query all tasks
	results, err := storage.QueryTasks(Filter{})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	// Expected order: 1.1, 1.2, 1.10, 2.1, 2.3, 10.1
	expectedOrder := []string{"1.1", "1.2", "1.10", "2.1", "2.3", "10.1"}

	if len(results) != len(expectedOrder) {
		t.Errorf("Expected %d tasks, got %d", len(expectedOrder), len(results))
	}

	for i, expected := range expectedOrder {
		if i < len(results) && results[i].ID != expected {
			t.Errorf("Position %d: expected ID %s, got %s", i, expected, results[i].ID)
		}
	}
}
