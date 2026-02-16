package commands

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yourusername/devloop/internal/config"
	"github.com/yourusername/devloop/internal/storage"
)

func TestTasksListCommand(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "devloop-tasks-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .devloop directory
	devloopDir := filepath.Join(tmpDir, ".devloop")
	if err := os.MkdirAll(devloopDir, 0755); err != nil {
		t.Fatalf("failed to create .devloop directory: %v", err)
	}

	// Create sample config
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
		Files: config.FilesConfig{
			PRD:   "docs/PRD.md",
			Tasks: "docs/TASKS.md",
			Todo:  ".todo/TODO.md",
		},
	}

	// Save config
	configPath := filepath.Join(devloopDir, "config.json")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create sample tasks
	store := storage.NewStorage(cfg)

	tasks := []*storage.Task{
		{
			ID:         "1.1",
			Title:      "Setup project",
			Wave:       1,
			Status:     "completed",
			Complexity: "simple",
			Model:      "claude-haiku-4-5-20251001",
			Execution: storage.TaskExecution{
				Attempts:      []storage.Attempt{{Duration: 30}},
				TotalDuration: 30,
			},
		},
		{
			ID:         "1.2",
			Title:      "Implement feature X",
			Wave:       1,
			Status:     "in_progress",
			Complexity: "moderate",
			Model:      "claude-sonnet-4-5-20250929",
			Execution: storage.TaskExecution{
				Attempts:      []storage.Attempt{{Duration: 120}},
				TotalDuration: 120,
			},
		},
		{
			ID:         "2.1",
			Title:      "Add tests",
			Wave:       2,
			Status:     "pending",
			Complexity: "simple",
			Model:      "claude-haiku-4-5-20251001",
			Tags:       []string{"testing"},
		},
		{
			ID:         "2.2",
			Title:      "Refactor component",
			Wave:       2,
			Status:     "failed",
			Complexity: "complex",
			Model:      "claude-opus-4-6",
			Execution: storage.TaskExecution{
				Attempts:      []storage.Attempt{{Duration: 300}, {Duration: 250}},
				TotalDuration: 550,
			},
		},
	}

	// Save tasks to JSONL
	for _, task := range tasks {
		if err := store.SaveTask(task); err != nil {
			t.Fatalf("failed to save task: %v", err)
		}
	}

	// Test that tasks can be loaded
	loadedTasks, err := store.LoadTasks()
	if err != nil {
		t.Fatalf("failed to load tasks: %v", err)
	}

	if len(loadedTasks) != len(tasks) {
		t.Fatalf("expected %d tasks, got %d", len(tasks), len(loadedTasks))
	}

	// Test filtering by status
	filter := storage.Filter{Status: "completed"}
	filtered, err := store.QueryTasks(filter)
	if err != nil {
		t.Fatalf("failed to query tasks: %v", err)
	}

	if len(filtered) != 1 {
		t.Fatalf("expected 1 completed task, got %d", len(filtered))
	}

	if filtered[0].ID != "1.1" {
		t.Errorf("expected task 1.1, got %s", filtered[0].ID)
	}

	// Test filtering by wave
	filter = storage.Filter{Wave: 2}
	filtered, err = store.QueryTasks(filter)
	if err != nil {
		t.Fatalf("failed to query tasks: %v", err)
	}

	if len(filtered) != 2 {
		t.Fatalf("expected 2 tasks in wave 2, got %d", len(filtered))
	}

	// Test filtering by complexity
	filter = storage.Filter{Complexity: "simple"}
	filtered, err = store.QueryTasks(filter)
	if err != nil {
		t.Fatalf("failed to query tasks: %v", err)
	}

	if len(filtered) != 2 {
		t.Fatalf("expected 2 simple tasks, got %d", len(filtered))
	}

	// Test filtering by tags
	filter = storage.Filter{Tags: []string{"testing"}}
	filtered, err = store.QueryTasks(filter)
	if err != nil {
		t.Fatalf("failed to query tasks: %v", err)
	}

	if len(filtered) != 1 {
		t.Fatalf("expected 1 task with 'testing' tag, got %d", len(filtered))
	}

	// Test tasks are sorted by ID
	filter = storage.Filter{}
	filtered, err = store.QueryTasks(filter)
	if err != nil {
		t.Fatalf("failed to query tasks: %v", err)
	}

	if len(filtered) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(filtered))
	}

	expectedOrder := []string{"1.1", "1.2", "2.1", "2.2"}
	for i, task := range filtered {
		if task.ID != expectedOrder[i] {
			t.Errorf("task at position %d: expected %s, got %s", i, expectedOrder[i], task.ID)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		expected string
	}{
		{"zero", 0, "-"},
		{"seconds only", 45, "45s"},
		{"one minute", 60, "1m"},
		{"minutes and seconds", 125, "2m5s"},
		{"minutes only", 180, "3m"},
		{"one hour", 3600, "1h"},
		{"hours and minutes", 3900, "1h5m"},
		{"hours only", 7200, "2h"},
		{"complex", 3665, "1h1m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.seconds)
			if result != tt.expected {
				t.Errorf("formatDuration(%d) = %s, expected %s", tt.seconds, result, tt.expected)
			}
		})
	}
}

func TestTruncateTitle(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		maxLen   int
		expected string
	}{
		{"no truncation needed", "Short title", 20, "Short title"},
		{"exact length", "Exactly20Characters!", 20, "Exactly20Characters!"},
		{"needs truncation", "This is a very long title that needs truncating", 20, "This is a very lo..."},
		{"very short", "Hi", 10, "Hi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateTitle(tt.title, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateTitle(%q, %d) = %q, expected %q", tt.title, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestFormatStatus(t *testing.T) {
	// Just verify that formatStatus doesn't panic and returns non-empty strings
	statuses := []string{"completed", "in_progress", "failed", "blocked", "archived", "pending"}

	for _, status := range statuses {
		result := formatStatus(status)
		if result == "" {
			t.Errorf("formatStatus(%s) returned empty string", status)
		}
	}
}

func TestTasksShowCommand(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "devloop-tasks-show-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .devloop directory
	devloopDir := filepath.Join(tmpDir, ".devloop")
	if err := os.MkdirAll(devloopDir, 0755); err != nil {
		t.Fatalf("failed to create .devloop directory: %v", err)
	}

	// Create sample config
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
		Files: config.FilesConfig{
			PRD:   "docs/PRD.md",
			Tasks: "docs/TASKS.md",
			Todo:  ".todo/TODO.md",
		},
	}

	// Save config
	configPath := filepath.Join(devloopDir, "config.json")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create sample task with full details
	store := storage.NewStorage(cfg)

	task := &storage.Task{
		ID:         "1.1",
		Title:      "Setup project structure",
		Wave:       1,
		Status:     "completed",
		Complexity: "simple",
		Model:      "claude-haiku-4-5-20251001",
		Description: "Initialize project with proper directory structure and dependencies",
		AcceptanceCriteria: []string{
			"Directory structure created",
			"Dependencies installed",
			"Tests pass",
		},
		BlockedBy: []string{},
		Tags:      []string{"setup", "infrastructure"},
		Metadata: storage.TaskMetadata{
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			SourceType:     "todo",
			SourceTodoItem: "TODO-1",
			MaxAttempts:    2,
		},
		Execution: storage.TaskExecution{
			Attempts: []storage.Attempt{
				{
					AttemptNumber: 1,
					StartedAt:     time.Now(),
					CompletedAt:   time.Now(),
					Duration:      45,
					Model:         "claude-haiku-4-5-20251001",
					Success:       true,
					Result:        "passed",
					LogPath:       ".devloop/logs/task-1.1-attempt-1.log",
					VerifyLogPath: ".devloop/logs/verify-1.1-attempt-1.log",
				},
			},
			TotalDuration: 45,
		},
		Results: &storage.TaskResults{
			VerificationOutput: "All tests passed\nBuild succeeded",
			CommitHash:         "abc123def456",
			CompletedAt:        time.Now(),
		},
	}

	// Save task
	if err := store.SaveTask(task); err != nil {
		t.Fatalf("failed to save task: %v", err)
	}

	// Test GetTask retrieves the task
	retrieved, err := store.GetTask("1.1")
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}

	if retrieved == nil {
		t.Fatal("expected task to be found, got nil")
	}

	if retrieved.ID != "1.1" {
		t.Errorf("expected task ID 1.1, got %s", retrieved.ID)
	}

	if retrieved.Title != "Setup project structure" {
		t.Errorf("expected title 'Setup project structure', got %s", retrieved.Title)
	}

	if len(retrieved.AcceptanceCriteria) != 3 {
		t.Errorf("expected 3 acceptance criteria, got %d", len(retrieved.AcceptanceCriteria))
	}

	if len(retrieved.Execution.Attempts) != 1 {
		t.Errorf("expected 1 attempt, got %d", len(retrieved.Execution.Attempts))
	}

	if retrieved.Results == nil {
		t.Error("expected results to be set")
	}

	if retrieved.Results != nil && retrieved.Results.CommitHash != "abc123def456" {
		t.Errorf("expected commit hash 'abc123def456', got %s", retrieved.Results.CommitHash)
	}

	// Test GetTask with non-existent ID
	notFound, err := store.GetTask("99.99")
	if err == nil {
		t.Error("expected error for non-existent task")
	}

	if notFound != nil {
		t.Error("expected nil task for non-existent task")
	}
}

func TestFormatAttemptResult(t *testing.T) {
	tests := []struct {
		name    string
		result  string
		success bool
	}{
		{"successful pass", "passed", true},
		{"failed attempt", "failed", false},
		{"error", "error", false},
		{"successful verify", "verify", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAttemptResult(tt.result, tt.success)
			// Just verify it doesn't panic and returns something
			if result == "" {
				t.Errorf("formatAttemptResult(%s, %v) returned empty string", tt.result, tt.success)
			}
		})
	}
}

