package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ceffo/devloop/internal/archiver"
	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/storage"
)

func createTestArchiveConfig(t *testing.T) (*config.Config, string) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Version: "1.0",
		Project: config.ProjectConfig{
			Name: "test-project", Path: tmpDir, TechStack: "Go", MainBranch: "main",
		},
		Verification: config.VerificationConfig{Command: "go test", TimeoutSeconds: 300},
		CLI: config.CLIConfig{
			Tool: "claude",
			Models: map[string]string{
				"simple":   "claude-haiku-4-5-20251001",
				"moderate": "claude-sonnet-4-5-20250929",
				"complex":  "claude-opus-4-6",
			},
		},
		Execution: config.ExecutionConfig{MaxAttempts: 2, HaltOnFailure: true, AutoCommit: true},
	}
	return cfg, tmpDir
}

func createTestArchiveTask(id, title, status string) *storage.Task {
	now := time.Now()
	task := &storage.Task{
		ID: id, Title: title, Status: status, Complexity: "simple",
		Description: "Test task", AcceptanceCriteria: []string{"Criterion 1"},
		BlockedBy: []string{}, Tags: []string{"test"},
		Metadata:  storage.TaskMetadata{CreatedAt: now, UpdatedAt: now, SourceType: "manual", MaxAttempts: 2},
		Execution: storage.TaskExecution{Attempts: []storage.Attempt{}, TotalDuration: 0},
	}
	if status == "completed" {
		task.Results = &storage.TaskResults{VerificationOutput: "Test passed", CompletedAt: now}
	}
	return task
}

func TestArchiveCmd_NoCompletedTasks(t *testing.T) {
	cfg, _ := createTestArchiveConfig(t)
	store := storage.NewStorage(cfg)

	// Create pending tasks only
	task1 := createTestArchiveTask("DEV-1", "Task 1", "pending")
	task2 := createTestArchiveTask("DEV-2", "Task 2", "in_progress")
	for _, task := range []*storage.Task{task1, task2} {
		if err := store.SaveTask(task); err != nil {
			t.Fatalf("Failed to save task: %v", err)
		}
	}

	// ExportCompleted should return 0 results without error
	arc := archiver.NewArchiver(cfg, store)
	result, err := arc.ExportCompleted()
	if err != nil {
		t.Fatalf("ExportCompleted failed: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("Expected 0 archived tasks, got %d", result.Total)
	}
}

func TestArchiveCmd_CompletedTasks(t *testing.T) {
	cfg, tmpDir := createTestArchiveConfig(t)
	store := storage.NewStorage(cfg)

	task1 := createTestArchiveTask("DEV-1", "Task 1", "completed")
	task2 := createTestArchiveTask("DEV-2", "Task 2", "completed")
	task3 := createTestArchiveTask("DEV-3", "Task 3", "pending")
	for _, task := range []*storage.Task{task1, task2, task3} {
		if err := store.SaveTask(task); err != nil {
			t.Fatalf("Failed to save task: %v", err)
		}
	}

	arc := archiver.NewArchiver(cfg, store)
	result, err := arc.ExportCompleted()
	if err != nil {
		t.Fatalf("ExportCompleted failed: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("Expected 2 archived tasks, got %d", result.Total)
	}

	// Verify archive file is in the archive directory
	archiveDir := filepath.Join(tmpDir, ".devloop", "archive")
	if !strings.HasPrefix(result.OutputPath, archiveDir) {
		t.Errorf("Archive file not in expected directory: %s", result.OutputPath)
	}
	if _, err := os.Stat(result.OutputPath); os.IsNotExist(err) {
		t.Error("Archive JSONL file was not created")
	}

	// Verify completed tasks are now archived
	updatedTasks, err := store.LoadTasks()
	if err != nil {
		t.Fatalf("Failed to load tasks: %v", err)
	}
	for _, task := range updatedTasks {
		switch task.ID {
		case "DEV-1", "DEV-2":
			if task.Status != "archived" {
				t.Errorf("Task %s should be archived, got %s", task.ID, task.Status)
			}
		case "DEV-3":
			if task.Status != "pending" {
				t.Errorf("Task DEV-3 should stay pending, got %s", task.Status)
			}
		}
	}
}

func TestArchiveCmd_MarkdownSummary(t *testing.T) {
	cfg, tmpDir := createTestArchiveConfig(t)
	store := storage.NewStorage(cfg)

	task1 := createTestArchiveTask("DEV-1", "Task One", "completed")
	task2 := createTestArchiveTask("DEV-2", "Task Two", "completed")
	for _, task := range []*storage.Task{task1, task2} {
		if err := store.SaveTask(task); err != nil {
			t.Fatalf("Failed to save task: %v", err)
		}
	}

	arc := archiver.NewArchiver(cfg, store)
	completedTasks, _ := store.QueryTasks(storage.Filter{Status: "completed"})
	markdown, err := arc.GenerateMarkdownSummary(completedTasks)
	if err != nil {
		t.Fatalf("GenerateMarkdownSummary failed: %v", err)
	}
	mdPath, err := arc.WriteMarkdownSummary(markdown)
	if err != nil {
		t.Fatalf("WriteMarkdownSummary failed: %v", err)
	}

	archiveDir := filepath.Join(tmpDir, ".devloop", "archive")
	if !strings.HasPrefix(mdPath, archiveDir) {
		t.Errorf("Markdown file not in expected directory: %s", mdPath)
	}
	if _, err := os.Stat(mdPath); os.IsNotExist(err) {
		t.Error("Markdown file was not created")
	}
}
