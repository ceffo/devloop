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

func TestArchiveEntryJSONRoundtrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	e := ArchiveEntry{
		TaskID:      "task-123",
		Title:       "Test Task",
		Status:      "completed",
		CompletedAt: &now,
		Wave:        2,
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var e2 ArchiveEntry
	if err := json.Unmarshal(b, &e2); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if e2.TaskID != e.TaskID || e2.Title != e.Title || e2.Status != e.Status || e2.Wave != e.Wave {
		t.Fatalf("mismatch: got %+v want %+v", e2, e)
	}
	if e2.CompletedAt == nil || !e2.CompletedAt.Equal(now) {
		t.Fatalf("completed_at mismatch: got %v want %v", e2.CompletedAt, now)
	}
}

func TestArchiveResultAndOptionsJSON(t *testing.T) {
	r := ArchiveResult{
		Total:      10,
		Completed:  7,
		ByWave:     map[string]int{"1": 3, "2": 4},
		OutputPath: "/tmp/out.md",
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal result failed: %v", err)
	}
	var r2 ArchiveResult
	if err := json.Unmarshal(b, &r2); err != nil {
		t.Fatalf("unmarshal result failed: %v", err)
	}
	if r2.Total != r.Total || r2.Completed != r.Completed || r2.OutputPath != r.OutputPath {
		t.Fatalf("mismatch: got %+v want %+v", r2, r)
	}
	o := ArchiveOptions{Wave: 2, Auto: true}
	bo, err := json.Marshal(o)
	if err != nil {
		t.Fatalf("marshal options failed: %v", err)
	}
	var o2 ArchiveOptions
	if err := json.Unmarshal(bo, &o2); err != nil {
		t.Fatalf("unmarshal options failed: %v", err)
	}
	if o2.Wave != o.Wave || o2.Auto != o.Auto {
		t.Fatalf("mismatch options: got %+v want %+v", o2, o)
	}
}

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
func createSampleTask(id, title, status string, wave int) *storage.Task {
	now := time.Now()
	task := &storage.Task{
		ID:          id,
		Title:       title,
		Wave:        wave,
		Status:      status,
		Complexity:  "simple",
		Description: "Test task description",
		AcceptanceCriteria: []string{
			"Criterion 1",
			"Criterion 2",
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

func TestNewArchiver(t *testing.T) {
	cfg, _ := createTestConfig(t)
	store := storage.NewStorage(cfg)
	archiver := NewArchiver(cfg, store)

	if archiver == nil {
		t.Fatal("NewArchiver returned nil")
	}
	if archiver.cfg != cfg {
		t.Error("Archiver config not set correctly")
	}
	if archiver.storage != store {
		t.Error("Archiver storage not set correctly")
	}
}

func TestExportWave_NoCompletedTasks(t *testing.T) {
	cfg, _ := createTestConfig(t)
	store := storage.NewStorage(cfg)
	archiver := NewArchiver(cfg, store)

	// Export wave with no completed tasks
	result, err := archiver.ExportWave(1)
	if err != nil {
		t.Fatalf("ExportWave failed: %v", err)
	}

	if result.Total != 0 {
		t.Errorf("Expected 0 total tasks, got %d", result.Total)
	}
	if result.Completed != 0 {
		t.Errorf("Expected 0 completed tasks, got %d", result.Completed)
	}
}

func TestExportWave_Success(t *testing.T) {
	cfg, tmpDir := createTestConfig(t)
	store := storage.NewStorage(cfg)
	archiver := NewArchiver(cfg, store)

	// Create and save completed tasks for wave 1
	task1 := createSampleTask("1.1", "Task 1", "completed", 1)
	task2 := createSampleTask("1.2", "Task 2", "completed", 1)
	task3 := createSampleTask("1.3", "Task 3", "pending", 1)   // Not completed
	task4 := createSampleTask("2.1", "Task 4", "completed", 2) // Different wave

	if err := store.SaveTask(task1); err != nil {
		t.Fatalf("Failed to save task1: %v", err)
	}
	if err := store.SaveTask(task2); err != nil {
		t.Fatalf("Failed to save task2: %v", err)
	}
	if err := store.SaveTask(task3); err != nil {
		t.Fatalf("Failed to save task3: %v", err)
	}
	if err := store.SaveTask(task4); err != nil {
		t.Fatalf("Failed to save task4: %v", err)
	}

	// Export wave 1
	result, err := archiver.ExportWave(1)
	if err != nil {
		t.Fatalf("ExportWave failed: %v", err)
	}

	// Verify result
	if result.Total != 2 {
		t.Errorf("Expected 2 total tasks, got %d", result.Total)
	}
	if result.Completed != 2 {
		t.Errorf("Expected 2 completed tasks, got %d", result.Completed)
	}

	// Verify archive file was created
	archivePath := filepath.Join(tmpDir, ".devloop", "archive", "wave-1.jsonl")
	if result.OutputPath != archivePath {
		t.Errorf("Expected output path %s, got %s", archivePath, result.OutputPath)
	}

	// Verify archive file exists and contains correct data
	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("Failed to open archive file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var entries []ArchiveEntry
	for scanner.Scan() {
		var entry ArchiveEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("Failed to parse archive entry: %v", err)
		}
		entries = append(entries, entry)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries in archive file, got %d", len(entries))
	}

	// Verify entry content
	for _, entry := range entries {
		if entry.Wave != 1 {
			t.Errorf("Expected wave 1, got %d", entry.Wave)
		}
		if entry.Status != "completed" {
			t.Errorf("Expected status 'completed', got %s", entry.Status)
		}
		if entry.CompletedAt == nil {
			t.Error("Expected CompletedAt to be set")
		}
	}

	// Verify tasks were updated to archived status
	updatedTasks, err := store.LoadTasks()
	if err != nil {
		t.Fatalf("Failed to load tasks: %v", err)
	}

	archivedCount := 0
	for _, task := range updatedTasks {
		if task.ID == "1.1" || task.ID == "1.2" {
			if task.Status != "archived" {
				t.Errorf("Task %s should have status 'archived', got %s", task.ID, task.Status)
			}
			archivedCount++
		}
	}
	if archivedCount != 2 {
		t.Errorf("Expected 2 tasks to be archived, got %d", archivedCount)
	}
}

func TestExportWave_ArchiveIndex(t *testing.T) {
	cfg, tmpDir := createTestConfig(t)
	store := storage.NewStorage(cfg)
	archiver := NewArchiver(cfg, store)

	// Create and save completed tasks for wave 1
	task1 := createSampleTask("1.1", "Task 1", "completed", 1)
	task2 := createSampleTask("1.2", "Task 2", "completed", 1)

	if err := store.SaveTask(task1); err != nil {
		t.Fatalf("Failed to save task1: %v", err)
	}
	if err := store.SaveTask(task2); err != nil {
		t.Fatalf("Failed to save task2: %v", err)
	}

	// Export wave 1
	if _, err := archiver.ExportWave(1); err != nil {
		t.Fatalf("ExportWave failed: %v", err)
	}

	// Verify index file was created
	indexPath := filepath.Join(tmpDir, ".devloop", "archive", "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Failed to read index file: %v", err)
	}

	var index ArchiveIndex
	if err := json.Unmarshal(data, &index); err != nil {
		t.Fatalf("Failed to parse index file: %v", err)
	}

	if len(index.Waves) != 1 {
		t.Errorf("Expected 1 wave in index, got %d", len(index.Waves))
	}

	entry := index.Waves[0]
	if entry.Wave != 1 {
		t.Errorf("Expected wave 1, got %d", entry.Wave)
	}
	if entry.TaskCount != 2 {
		t.Errorf("Expected 2 tasks, got %d", entry.TaskCount)
	}
	if entry.ArchivedAt.IsZero() {
		t.Error("Expected ArchivedAt to be set")
	}
	expectedPath := filepath.Join(tmpDir, ".devloop", "archive", "wave-1.jsonl")
	if entry.ArchivePath != expectedPath {
		t.Errorf("Expected archive path %s, got %s", expectedPath, entry.ArchivePath)
	}
}

func TestExportWave_MultipleWaves(t *testing.T) {
	cfg, tmpDir := createTestConfig(t)
	store := storage.NewStorage(cfg)
	archiver := NewArchiver(cfg, store)

	// Create and save completed tasks for different waves
	task1 := createSampleTask("1.1", "Task 1", "completed", 1)
	task2 := createSampleTask("2.1", "Task 2", "completed", 2)
	task3 := createSampleTask("2.2", "Task 3", "completed", 2)

	if err := store.SaveTask(task1); err != nil {
		t.Fatalf("Failed to save task1: %v", err)
	}
	if err := store.SaveTask(task2); err != nil {
		t.Fatalf("Failed to save task2: %v", err)
	}
	if err := store.SaveTask(task3); err != nil {
		t.Fatalf("Failed to save task3: %v", err)
	}

	// Export wave 1
	result1, err := archiver.ExportWave(1)
	if err != nil {
		t.Fatalf("ExportWave(1) failed: %v", err)
	}
	if result1.Total != 1 {
		t.Errorf("Expected 1 task in wave 1, got %d", result1.Total)
	}

	// Export wave 2
	result2, err := archiver.ExportWave(2)
	if err != nil {
		t.Fatalf("ExportWave(2) failed: %v", err)
	}
	if result2.Total != 2 {
		t.Errorf("Expected 2 tasks in wave 2, got %d", result2.Total)
	}

	// Verify index contains both waves
	indexPath := filepath.Join(tmpDir, ".devloop", "archive", "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Failed to read index file: %v", err)
	}

	var index ArchiveIndex
	if err := json.Unmarshal(data, &index); err != nil {
		t.Fatalf("Failed to parse index file: %v", err)
	}

	if len(index.Waves) != 2 {
		t.Errorf("Expected 2 waves in index, got %d", len(index.Waves))
	}
}

func TestExportWave_ReArchiving(t *testing.T) {
	cfg, tmpDir := createTestConfig(t)
	store := storage.NewStorage(cfg)
	archiver := NewArchiver(cfg, store)

	// Create and save completed tasks for wave 1
	task1 := createSampleTask("1.1", "Task 1", "completed", 1)

	if err := store.SaveTask(task1); err != nil {
		t.Fatalf("Failed to save task1: %v", err)
	}

	// Export wave 1 first time
	result1, err := archiver.ExportWave(1)
	if err != nil {
		t.Fatalf("First ExportWave failed: %v", err)
	}
	if result1.Total != 1 {
		t.Errorf("Expected 1 task, got %d", result1.Total)
	}

	// Mark task as completed again (simulating re-completion)
	task1.Status = "completed"
	if err := store.UpdateTask(task1); err != nil {
		t.Fatalf("Failed to update task: %v", err)
	}

	// Export wave 1 second time
	result2, err := archiver.ExportWave(1)
	if err != nil {
		t.Fatalf("Second ExportWave failed: %v", err)
	}
	if result2.Total != 1 {
		t.Errorf("Expected 1 task on re-archive, got %d", result2.Total)
	}

	// Verify index was updated (not duplicated)
	indexPath := filepath.Join(tmpDir, ".devloop", "archive", "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Failed to read index file: %v", err)
	}

	var index ArchiveIndex
	if err := json.Unmarshal(data, &index); err != nil {
		t.Fatalf("Failed to parse index file: %v", err)
	}

	if len(index.Waves) != 1 {
		t.Errorf("Expected 1 wave in index (no duplicate), got %d", len(index.Waves))
	}
}

func TestGenerateMarkdownSummary_EmptyTasks(t *testing.T) {
	cfg, _ := createTestConfig(t)
	store := storage.NewStorage(cfg)
	archiver := NewArchiver(cfg, store)

	// Generate summary with no tasks
	markdown, err := archiver.GenerateMarkdownSummary(1, []*storage.Task{})
	if err != nil {
		t.Fatalf("GenerateMarkdownSummary failed: %v", err)
	}

	if markdown != "" {
		t.Errorf("Expected empty markdown for empty tasks, got %q", markdown)
	}
}

func TestGenerateMarkdownSummary_SingleTask(t *testing.T) {
	cfg, _ := createTestConfig(t)
	store := storage.NewStorage(cfg)
	archiver := NewArchiver(cfg, store)

	// Create a completed task
	task := createSampleTask("1.1", "Sample Task", "completed", 1)
	task.Execution.TotalDuration = 120

	markdown, err := archiver.GenerateMarkdownSummary(1, []*storage.Task{task})
	if err != nil {
		t.Fatalf("GenerateMarkdownSummary failed: %v", err)
	}

	// Verify markdown contains expected sections
	if !containsString(markdown, "# Wave 1 Archive Summary") {
		t.Error("Markdown missing title section")
	}
	if !containsString(markdown, "## Tasks") {
		t.Error("Markdown missing Tasks section")
	}
	if !containsString(markdown, "## Statistics") {
		t.Error("Markdown missing Statistics section")
	}

	// Verify task is listed
	if !containsString(markdown, "1.1") {
		t.Error("Markdown missing task ID")
	}
	if !containsString(markdown, "Sample Task") {
		t.Error("Markdown missing task title")
	}

	// Verify statistics
	if !containsString(markdown, "**Total Tasks:** 1") {
		t.Error("Markdown missing total tasks count")
	}
	if !containsString(markdown, "**Completed:** 1") {
		t.Error("Markdown missing completion count")
	}
	if !containsString(markdown, "**Completion Rate:** 100.0%") {
		t.Error("Markdown missing completion rate")
	}
	if !containsString(markdown, "**Total Duration:** 120 seconds") {
		t.Error("Markdown missing duration")
	}
}

func TestGenerateMarkdownSummary_MultipleTasks(t *testing.T) {
	cfg, _ := createTestConfig(t)
	store := storage.NewStorage(cfg)
	archiver := NewArchiver(cfg, store)

	// Create multiple completed tasks
	task1 := createSampleTask("1.1", "Task One", "completed", 1)
	task1.Execution.TotalDuration = 100
	task2 := createSampleTask("1.2", "Task Two", "completed", 1)
	task2.Execution.TotalDuration = 200

	markdown, err := archiver.GenerateMarkdownSummary(1, []*storage.Task{task1, task2})
	if err != nil {
		t.Fatalf("GenerateMarkdownSummary failed: %v", err)
	}

	// Verify both tasks are listed
	if !containsString(markdown, "1.1") {
		t.Error("Markdown missing task 1.1")
	}
	if !containsString(markdown, "1.2") {
		t.Error("Markdown missing task 1.2")
	}
	if !containsString(markdown, "Task One") {
		t.Error("Markdown missing task one title")
	}
	if !containsString(markdown, "Task Two") {
		t.Error("Markdown missing task two title")
	}

	// Verify statistics
	if !containsString(markdown, "**Total Tasks:** 2") {
		t.Error("Markdown missing total count")
	}
	if !containsString(markdown, "**Completion Rate:** 100.0%") {
		t.Error("Markdown missing completion rate")
	}
	if !containsString(markdown, "**Total Duration:** 300 seconds") {
		t.Error("Markdown missing total duration (100 + 200)")
	}
}

func TestGenerateMarkdownSummary_PartialCompletion(t *testing.T) {
	cfg, _ := createTestConfig(t)
	store := storage.NewStorage(cfg)
	archiver := NewArchiver(cfg, store)

	// Create tasks with mixed statuses
	task1 := createSampleTask("1.1", "Completed Task", "completed", 1)
	task2 := createSampleTask("1.2", "Pending Task", "pending", 1)
	task3 := createSampleTask("1.3", "Archived Task", "archived", 1)

	markdown, err := archiver.GenerateMarkdownSummary(1, []*storage.Task{task1, task2, task3})
	if err != nil {
		t.Fatalf("GenerateMarkdownSummary failed: %v", err)
	}

	// Verify statistics reflect partial completion
	if !containsString(markdown, "**Total Tasks:** 3") {
		t.Error("Markdown should show 3 total tasks")
	}
	if !containsString(markdown, "**Completed:** 2") {
		t.Error("Markdown should show 2 completed (completed + archived)")
	}
	if !containsString(markdown, "**Completion Rate:** 66.7%") {
		t.Error("Markdown should show 66.7% completion rate")
	}
}

func TestWriteMarkdownSummary_Success(t *testing.T) {
	cfg, tmpDir := createTestConfig(t)
	store := storage.NewStorage(cfg)
	archiver := NewArchiver(cfg, store)

	markdown := "# Wave 1 Summary\n\nTest content"
	mdPath, err := archiver.WriteMarkdownSummary(1, markdown)
	if err != nil {
		t.Fatalf("WriteMarkdownSummary failed: %v", err)
	}

	// Verify file was created at correct location
	expectedPath := filepath.Join(tmpDir, ".devloop", "archive", "wave-1.md")
	if mdPath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, mdPath)
	}

	// Verify file exists and contains correct content
	data, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("Failed to read markdown file: %v", err)
	}

	if string(data) != markdown {
		t.Errorf("Markdown content mismatch: got %q, want %q", string(data), markdown)
	}
}

func TestWriteMarkdownSummary_CreateDirectory(t *testing.T) {
	cfg, tmpDir := createTestConfig(t)
	store := storage.NewStorage(cfg)
	archiver := NewArchiver(cfg, store)

	// Verify archive directory doesn't exist initially
	archiveDir := filepath.Join(tmpDir, ".devloop", "archive")
	if _, err := os.Stat(archiveDir); !os.IsNotExist(err) {
		// Directory might exist if other tests ran first, that's okay
	}

	markdown := "# Test"
	_, err := archiver.WriteMarkdownSummary(1, markdown)
	if err != nil {
		t.Fatalf("WriteMarkdownSummary failed: %v", err)
	}

	// Verify archive directory was created
	if _, err := os.Stat(archiveDir); os.IsNotExist(err) {
		t.Error("Archive directory was not created")
	}
}

func TestGenerateAndWriteMarkdownSummary_Integration(t *testing.T) {
	cfg, tmpDir := createTestConfig(t)
	store := storage.NewStorage(cfg)
	archiver := NewArchiver(cfg, store)

	// Create and save completed tasks
	task1 := createSampleTask("2.1", "Feature A", "completed", 2)
	task1.Execution.TotalDuration = 300
	task2 := createSampleTask("2.2", "Feature B", "completed", 2)
	task2.Execution.TotalDuration = 450

	if err := store.SaveTask(task1); err != nil {
		t.Fatalf("Failed to save task1: %v", err)
	}
	if err := store.SaveTask(task2); err != nil {
		t.Fatalf("Failed to save task2: %v", err)
	}

	// Generate markdown summary
	markdown, err := archiver.GenerateMarkdownSummary(2, []*storage.Task{task1, task2})
	if err != nil {
		t.Fatalf("GenerateMarkdownSummary failed: %v", err)
	}

	// Write markdown summary
	mdPath, err := archiver.WriteMarkdownSummary(2, markdown)
	if err != nil {
		t.Fatalf("WriteMarkdownSummary failed: %v", err)
	}

	// Verify file exists and has correct content
	expectedPath := filepath.Join(tmpDir, ".devloop", "archive", "wave-2.md")
	if mdPath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, mdPath)
	}

	data, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("Failed to read markdown file: %v", err)
	}

	mdContent := string(data)
	if !containsString(mdContent, "Wave 2") {
		t.Error("Markdown missing wave number")
	}
	if !containsString(mdContent, "2.1") || !containsString(mdContent, "2.2") {
		t.Error("Markdown missing task IDs")
	}
	if !containsString(mdContent, "Feature A") || !containsString(mdContent, "Feature B") {
		t.Error("Markdown missing task titles")
	}
	if !containsString(mdContent, "**Total Tasks:** 2") {
		t.Error("Markdown missing statistics")
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
