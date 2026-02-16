package commands

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ceffo/devloop/internal/archiver"
	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/storage"
)

// Helper function to create test config
func createTestArchiveConfig(t *testing.T) (*config.Config, string) {
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

// Helper function to create test task
func createTestArchiveTask(id, title, status string, wave int) *storage.Task {
	now := time.Now()
	task := &storage.Task{
		ID:          id,
		Title:       title,
		Wave:        wave,
		Status:      status,
		Complexity:  "simple",
		Description: "Test task",
		AcceptanceCriteria: []string{
			"Criterion 1",
		},
		BlockedBy: []string{},
		Tags:      []string{"test"},
		Metadata: storage.TaskMetadata{
			CreatedAt:   now,
			UpdatedAt:   now,
			SourceType:  "manual",
			MaxAttempts: 2,
		},
		Execution: storage.TaskExecution{
			Attempts:      []storage.Attempt{},
			TotalDuration: 0,
		},
	}

	if status == "completed" {
		task.Results = &storage.TaskResults{
			VerificationOutput: "Test passed",
			CompletedAt:        now,
		}
	}

	return task
}

func TestIsWaveFullyCompleted(t *testing.T) {
	tests := []struct {
		name     string
		tasks    []*storage.Task
		expected bool
	}{
		{
			name:     "empty wave",
			tasks:    []*storage.Task{},
			expected: false,
		},
		{
			name: "all completed",
			tasks: []*storage.Task{
				createTestArchiveTask("1.1", "Task 1", "completed", 1),
				createTestArchiveTask("1.2", "Task 2", "completed", 1),
			},
			expected: true,
		},
		{
			name: "all archived",
			tasks: []*storage.Task{
				createTestArchiveTask("1.1", "Task 1", "archived", 1),
				createTestArchiveTask("1.2", "Task 2", "archived", 1),
			},
			expected: true,
		},
		{
			name: "mix of completed and archived",
			tasks: []*storage.Task{
				createTestArchiveTask("1.1", "Task 1", "completed", 1),
				createTestArchiveTask("1.2", "Task 2", "archived", 1),
			},
			expected: true,
		},
		{
			name: "has pending task",
			tasks: []*storage.Task{
				createTestArchiveTask("1.1", "Task 1", "completed", 1),
				createTestArchiveTask("1.2", "Task 2", "pending", 1),
			},
			expected: false,
		},
		{
			name: "has in_progress task",
			tasks: []*storage.Task{
				createTestArchiveTask("1.1", "Task 1", "completed", 1),
				createTestArchiveTask("1.2", "Task 2", "in_progress", 1),
			},
			expected: false,
		},
		{
			name: "has failed task",
			tasks: []*storage.Task{
				createTestArchiveTask("1.1", "Task 1", "completed", 1),
				createTestArchiveTask("1.2", "Task 2", "failed", 1),
			},
			expected: false,
		},
		{
			name: "has blocked task",
			tasks: []*storage.Task{
				createTestArchiveTask("1.1", "Task 1", "completed", 1),
				createTestArchiveTask("1.2", "Task 2", "blocked", 1),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWaveFullyCompleted(tt.tasks)
			if result != tt.expected {
				t.Errorf("isWaveFullyCompleted() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestArchiveWave_NoTasks(t *testing.T) {
	cfg, _ := createTestArchiveConfig(t)
	store := storage.NewStorage(cfg)

	// Try to archive wave that doesn't exist
	err := archiveWave(nil, store, 999)
	if err == nil {
		t.Error("Expected error for non-existent wave, got nil")
	}
	if err != nil && err.Error() != "no tasks found for wave 999" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestArchiveWave_NoCompletedTasks(t *testing.T) {
	cfg, _ := createTestArchiveConfig(t)
	store := storage.NewStorage(cfg)

	// Create pending tasks for wave 1
	task1 := createTestArchiveTask("1.1", "Task 1", "pending", 1)
	task2 := createTestArchiveTask("1.2", "Task 2", "in_progress", 1)

	if err := store.SaveTask(task1); err != nil {
		t.Fatalf("Failed to save task1: %v", err)
	}
	if err := store.SaveTask(task2); err != nil {
		t.Fatalf("Failed to save task2: %v", err)
	}

	// Try to archive wave with no completed tasks
	err := archiveWave(nil, store, 1)
	if err == nil {
		t.Error("Expected error for wave with no completed tasks, got nil")
	}
	if err != nil && err.Error() != "no completed tasks found for wave 1" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestArchiveAuto_NoTasks(t *testing.T) {
	cfg, _ := createTestArchiveConfig(t)
	store := storage.NewStorage(cfg)

	// Run auto archive with no tasks
	err := archiveAuto(nil, store)
	if err != nil {
		t.Errorf("Expected no error for empty task list, got: %v", err)
	}
}

func TestArchiveAuto_NoFullyCompletedWaves(t *testing.T) {
	cfg, _ := createTestArchiveConfig(t)
	store := storage.NewStorage(cfg)

	// Create partially completed waves
	task1 := createTestArchiveTask("1.1", "Task 1", "completed", 1)
	task2 := createTestArchiveTask("1.2", "Task 2", "pending", 1)
	task3 := createTestArchiveTask("2.1", "Task 3", "in_progress", 2)

	if err := store.SaveTask(task1); err != nil {
		t.Fatalf("Failed to save task1: %v", err)
	}
	if err := store.SaveTask(task2); err != nil {
		t.Fatalf("Failed to save task2: %v", err)
	}
	if err := store.SaveTask(task3); err != nil {
		t.Fatalf("Failed to save task3: %v", err)
	}

	// Run auto archive - should find no fully completed waves
	err := archiveAuto(nil, store)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestArchiveCmd_ValidationErrors(t *testing.T) {
	cfg, tmpDir := createTestArchiveConfig(t)
	store := storage.NewStorage(cfg)
	arc := archiver.NewArchiver(cfg, store)

	tests := []struct {
		name        string
		runFunc     func() error
		expectError bool
		errorMsg    string
	}{
		{
			name: "wave with no tasks",
			runFunc: func() error {
				return archiveWave(arc, store, 999)
			},
			expectError: true,
			errorMsg:    "no tasks found for wave 999",
		},
		{
			name: "wave with no completed tasks",
			runFunc: func() error {
				// Create pending task
				task := createTestArchiveTask("1.1", "Task 1", "pending", 1)
				if err := store.SaveTask(task); err != nil {
					return err
				}
				return archiveWave(arc, store, 1)
			},
			expectError: true,
			errorMsg:    "no completed tasks found for wave 1",
		},
		{
			name: "auto with no tasks",
			runFunc: func() error {
				return archiveAuto(arc, store)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh config and storage for each test
			cfg, _ := createTestArchiveConfig(t)
			store := storage.NewStorage(cfg)
			arc := archiver.NewArchiver(cfg, store)

			// Update runFunc to use new instances if needed
			var err error
			switch tt.name {
			case "wave with no tasks":
				err = archiveWave(arc, store, 999)
			case "wave with no completed tasks":
				task := createTestArchiveTask("1.1", "Task 1", "pending", 1)
				if serr := store.SaveTask(task); serr != nil {
					t.Fatalf("Failed to save task: %v", serr)
				}
				err = archiveWave(arc, store, 1)
			case "auto with no tasks":
				err = archiveAuto(arc, store)
			default:
				err = tt.runFunc()
			}

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.expectError && err != nil && tt.errorMsg != "" {
				if err.Error() != tt.errorMsg {
					t.Errorf("Expected error message %q, got %q", tt.errorMsg, err.Error())
				}
			}
		})
	}

	// Cleanup
	_ = tmpDir
}

func TestArchiveIntegration_SpecificWave(t *testing.T) {
	cfg, tmpDir := createTestArchiveConfig(t)
	store := storage.NewStorage(cfg)
	arc := archiver.NewArchiver(cfg, store)

	// Create completed tasks for wave 1
	task1 := createTestArchiveTask("1.1", "Task 1", "completed", 1)
	task2 := createTestArchiveTask("1.2", "Task 2", "completed", 1)

	if err := store.SaveTask(task1); err != nil {
		t.Fatalf("Failed to save task1: %v", err)
	}
	if err := store.SaveTask(task2); err != nil {
		t.Fatalf("Failed to save task2: %v", err)
	}

	// Run archive for wave 1
	err := archiveWave(arc, store, 1)
	if err != nil {
		t.Fatalf("Archive command failed: %v", err)
	}

	// Verify archive files were created
	archiveDir := filepath.Join(tmpDir, ".devloop", "archive")
	jsonlPath := filepath.Join(archiveDir, "wave-1.jsonl")
	mdPath := filepath.Join(archiveDir, "wave-1.md")
	indexPath := filepath.Join(archiveDir, "index.json")

	if _, err := os.Stat(jsonlPath); os.IsNotExist(err) {
		t.Error("JSONL archive file was not created")
	}

	if _, err := os.Stat(mdPath); os.IsNotExist(err) {
		t.Error("Markdown archive file was not created")
	}

	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("Archive index file was not created")
	}

	// Verify tasks were updated to archived status
	updatedTasks, err := store.LoadTasks()
	if err != nil {
		t.Fatalf("Failed to load tasks: %v", err)
	}

	for _, task := range updatedTasks {
		if task.ID == "1.1" || task.ID == "1.2" {
			if task.Status != "archived" {
				t.Errorf("Task %s should be archived, got status: %s", task.ID, task.Status)
			}
		}
	}
}

func TestArchiveIntegration_AutoMode(t *testing.T) {
	cfg, tmpDir := createTestArchiveConfig(t)
	store := storage.NewStorage(cfg)
	arc := archiver.NewArchiver(cfg, store)

	// Create fully completed wave 1
	task1 := createTestArchiveTask("1.1", "Task 1", "completed", 1)
	task2 := createTestArchiveTask("1.2", "Task 2", "completed", 1)

	// Create partially completed wave 2
	task3 := createTestArchiveTask("2.1", "Task 3", "completed", 2)
	task4 := createTestArchiveTask("2.2", "Task 4", "pending", 2)

	// Create fully completed wave 3
	task5 := createTestArchiveTask("3.1", "Task 5", "completed", 3)

	tasks := []*storage.Task{task1, task2, task3, task4, task5}
	for _, task := range tasks {
		if err := store.SaveTask(task); err != nil {
			t.Fatalf("Failed to save task %s: %v", task.ID, err)
		}
	}

	// Run archive in auto mode
	err := archiveAuto(arc, store)
	if err != nil {
		t.Fatalf("Archive command failed: %v", err)
	}

	// Verify that wave 1 and wave 3 were archived, but not wave 2
	archiveDir := filepath.Join(tmpDir, ".devloop", "archive")

	// Wave 1 should be archived
	wave1JSONL := filepath.Join(archiveDir, "wave-1.jsonl")
	wave1MD := filepath.Join(archiveDir, "wave-1.md")
	if _, err := os.Stat(wave1JSONL); os.IsNotExist(err) {
		t.Error("Wave 1 JSONL archive should exist")
	}
	if _, err := os.Stat(wave1MD); os.IsNotExist(err) {
		t.Error("Wave 1 markdown archive should exist")
	}

	// Wave 2 should NOT be archived (has pending task)
	wave2JSONL := filepath.Join(archiveDir, "wave-2.jsonl")
	wave2MD := filepath.Join(archiveDir, "wave-2.md")
	if _, err := os.Stat(wave2JSONL); !os.IsNotExist(err) {
		t.Error("Wave 2 JSONL archive should NOT exist")
	}
	if _, err := os.Stat(wave2MD); !os.IsNotExist(err) {
		t.Error("Wave 2 markdown archive should NOT exist")
	}

	// Wave 3 should be archived
	wave3JSONL := filepath.Join(archiveDir, "wave-3.jsonl")
	wave3MD := filepath.Join(archiveDir, "wave-3.md")
	if _, err := os.Stat(wave3JSONL); os.IsNotExist(err) {
		t.Error("Wave 3 JSONL archive should exist")
	}
	if _, err := os.Stat(wave3MD); os.IsNotExist(err) {
		t.Error("Wave 3 markdown archive should exist")
	}

	// Verify task statuses
	updatedTasks, err := store.LoadTasks()
	if err != nil {
		t.Fatalf("Failed to load tasks: %v", err)
	}

	expectedStatuses := map[string]string{
		"1.1": "archived",  // Wave 1 fully completed, so archived
		"1.2": "archived",  // Wave 1 fully completed, so archived
		"2.1": "completed", // Wave 2 not fully completed, so stays completed
		"2.2": "pending",   // Wave 2 not fully completed, pending task stays pending
		"3.1": "archived",  // Wave 3 fully completed, so archived
	}

	for _, task := range updatedTasks {
		if expected, ok := expectedStatuses[task.ID]; ok {
			if task.Status != expected {
				t.Errorf("Task %s: expected status %s, got %s", task.ID, expected, task.Status)
			}
		}
	}
}
