package archiver

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/storage"
)

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

func createSampleTask(id, title, status string) *storage.Task {
	now := time.Now()
	task := &storage.Task{
		ID:          id,
		Title:       title,
		Status:      status,
		Complexity:  "simple",
		Description: "Test task description",
		AcceptanceCriteria: []string{"Criterion 1"},
		BlockedBy:   []string{},
		Tags:        []string{"test"},
		Metadata: storage.TaskMetadata{
			CreatedAt: now, UpdatedAt: now, SourceType: "manual", MaxAttempts: 2,
		},
		Execution: storage.TaskExecution{Attempts: []storage.Attempt{}, TotalDuration: 0},
	}
	if status == "completed" {
		task.Results = &storage.TaskResults{VerificationOutput: "Test passed", CompletedAt: now}
	}
	return task
}

func TestNewArchiver(t *testing.T) {
	cfg, _ := createTestConfig(t)
	store := storage.NewStorage(cfg)
	arc := NewArchiver(cfg, store)
	if arc == nil {
		t.Fatal("NewArchiver returned nil")
	}
}

func TestArchiveEntryJSONRoundtrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	e := ArchiveEntry{TaskID: "DEV-1", Title: "Test Task", Status: "completed", CompletedAt: &now}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var e2 ArchiveEntry
	if err := json.Unmarshal(b, &e2); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if e2.TaskID != e.TaskID || e2.Title != e.Title || e2.Status != e.Status {
		t.Fatalf("mismatch: got %+v want %+v", e2, e)
	}
	if e2.CompletedAt == nil || !e2.CompletedAt.Equal(now) {
		t.Fatalf("completed_at mismatch: got %v want %v", e2.CompletedAt, now)
	}
}

func TestExportCompleted_NoTasks(t *testing.T) {
	cfg, _ := createTestConfig(t)
	store := storage.NewStorage(cfg)
	arc := NewArchiver(cfg, store)

	result, err := arc.ExportCompleted()
	if err != nil {
		t.Fatalf("ExportCompleted failed: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("Expected 0 total tasks, got %d", result.Total)
	}
}

func TestExportCompleted_Success(t *testing.T) {
	cfg, tmpDir := createTestConfig(t)
	store := storage.NewStorage(cfg)
	arc := NewArchiver(cfg, store)

	task1 := createSampleTask("DEV-1", "Task 1", "completed")
	task2 := createSampleTask("DEV-2", "Task 2", "completed")
	task3 := createSampleTask("DEV-3", "Task 3", "pending") // should not be archived

	for _, task := range []*storage.Task{task1, task2, task3} {
		if err := store.SaveTask(task); err != nil {
			t.Fatalf("Failed to save task: %v", err)
		}
	}

	result, err := arc.ExportCompleted()
	if err != nil {
		t.Fatalf("ExportCompleted failed: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("Expected 2 archived tasks, got %d", result.Total)
	}

	// Verify archive file exists and has 2 entries
	file, err := os.Open(result.OutputPath)
	if err != nil {
		t.Fatalf("Failed to open archive file: %v", err)
	}
	defer file.Close()

	var entries []ArchiveEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry ArchiveEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("Failed to parse entry: %v", err)
		}
		entries = append(entries, entry)
	}
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries in archive file, got %d", len(entries))
	}

	// Verify tasks were updated to archived
	updatedTasks, err := store.LoadTasks()
	if err != nil {
		t.Fatalf("Failed to load tasks: %v", err)
	}
	for _, task := range updatedTasks {
		if task.ID == "DEV-1" || task.ID == "DEV-2" {
			if task.Status != "archived" {
				t.Errorf("Task %s should be archived, got %s", task.ID, task.Status)
			}
		}
		if task.ID == "DEV-3" && task.Status != "pending" {
			t.Errorf("Pending task DEV-3 should stay pending, got %s", task.Status)
		}
	}

	// Verify archive file is in the archive directory
	archiveDir := filepath.Join(tmpDir, ".devloop", "archive")
	if !strings.HasPrefix(result.OutputPath, archiveDir) {
		t.Errorf("Archive file not in expected directory: %s", result.OutputPath)
	}
}

func TestGenerateMarkdownSummary_Empty(t *testing.T) {
	cfg, _ := createTestConfig(t)
	store := storage.NewStorage(cfg)
	arc := NewArchiver(cfg, store)

	markdown, err := arc.GenerateMarkdownSummary([]*storage.Task{})
	if err != nil {
		t.Fatalf("GenerateMarkdownSummary failed: %v", err)
	}
	if markdown != "" {
		t.Errorf("Expected empty markdown, got %q", markdown)
	}
}

func TestGenerateMarkdownSummary_WithTasks(t *testing.T) {
	cfg, _ := createTestConfig(t)
	store := storage.NewStorage(cfg)
	arc := NewArchiver(cfg, store)

	task1 := createSampleTask("DEV-1", "Task One", "completed")
	task1.Execution.TotalDuration = 100
	task2 := createSampleTask("DEV-2", "Task Two", "completed")
	task2.Execution.TotalDuration = 200

	markdown, err := arc.GenerateMarkdownSummary([]*storage.Task{task1, task2})
	if err != nil {
		t.Fatalf("GenerateMarkdownSummary failed: %v", err)
	}

	for _, substr := range []string{"# Archive Summary", "## Tasks", "## Statistics", "DEV-1", "DEV-2", "Task One", "Task Two", "**Total Tasks:** 2", "300 seconds"} {
		if !strings.Contains(markdown, substr) {
			t.Errorf("Markdown missing %q", substr)
		}
	}
}

func TestWriteMarkdownSummary(t *testing.T) {
	cfg, tmpDir := createTestConfig(t)
	store := storage.NewStorage(cfg)
	arc := NewArchiver(cfg, store)

	markdown := "# Archive Summary\n\nTest content"
	mdPath, err := arc.WriteMarkdownSummary(markdown)
	if err != nil {
		t.Fatalf("WriteMarkdownSummary failed: %v", err)
	}

	archiveDir := filepath.Join(tmpDir, ".devloop", "archive")
	if !strings.HasPrefix(mdPath, archiveDir) {
		t.Errorf("Markdown file not in expected directory: %s", mdPath)
	}

	data, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("Failed to read markdown file: %v", err)
	}
	if string(data) != markdown {
		t.Errorf("Markdown content mismatch: got %q, want %q", string(data), markdown)
	}
}
