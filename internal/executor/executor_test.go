package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yourusername/devloop/internal/config"
	"github.com/yourusername/devloop/internal/storage"
)

// mockAgentRunner is a mock implementation of AgentRunner for testing
type mockAgentRunner struct {
	shouldSucceed bool
	shouldError   bool
	errorMessage  string
}

func (m *mockAgentRunner) Run(model, prompt, logPath string) (*AgentResult, error) {
	// Create log directory and file
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(logPath, []byte("Mock agent output"), 0644); err != nil {
		return nil, err
	}

	result := &AgentResult{
		LogPath: logPath,
		Output:  "Mock agent output",
	}

	if m.shouldError {
		result.Success = false
		result.Error = &AgentError{Message: m.errorMessage}
		return result, nil
	}

	result.Success = m.shouldSucceed
	return result, nil
}

type AgentError struct {
	Message string
}

func (e *AgentError) Error() string {
	return e.Message
}

// TestExecuteTaskSuccess tests successful task execution
func TestExecuteTaskSuccess(t *testing.T) {
	// Setup temp directory
	tmpDir := t.TempDir()

	// Create test config
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path:       tmpDir,
			Name:       "test",
			TechStack:  "Go",
			MainBranch: "main",
		},
		Verification: config.VerificationConfig{
			Command:        "echo 'verification passed'",
			TimeoutSeconds: 10,
		},
		CLI: config.CLIConfig{
			Tool: "claude",
			Models: map[string]string{
				"simple": "test-model",
			},
		},
		Execution: config.ExecutionConfig{
			MaxAttempts:   2,
			AutoCommit:    false,
			HaltOnFailure: true,
		},
		Files: config.FilesConfig{},
	}

	// Create storage
	store := storage.NewStorage(cfg)

	// Create test task
	task := &storage.Task{
		ID:          "1.1",
		Title:       "Test Task",
		Status:      "pending",
		Complexity:  "simple",
		Model:       "test-model",
		Description: "Test description",
		Metadata: storage.TaskMetadata{
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			MaxAttempts: 2,
		},
		Execution: storage.TaskExecution{
			Attempts: []storage.Attempt{},
		},
	}

	// Save task
	if err := store.SaveTask(task); err != nil {
		t.Fatalf("Failed to save task: %v", err)
	}

	// Create mock agent runner (success)
	runner := &mockAgentRunner{
		shouldSucceed: true,
		shouldError:   false,
	}

	// Execute task
	ctx := context.Background()
	success, err := executeTask(ctx, cfg, store, runner, task)

	if err != nil {
		t.Fatalf("executeTask returned error: %v", err)
	}

	if !success {
		t.Errorf("Expected task to succeed, but it failed")
	}

	// Verify task status
	updatedTask, err := store.GetTask("1.1")
	if err != nil {
		t.Fatalf("Failed to load updated task: %v", err)
	}

	if updatedTask.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", updatedTask.Status)
	}

	if len(updatedTask.Execution.Attempts) != 1 {
		t.Errorf("Expected 1 attempt, got %d", len(updatedTask.Execution.Attempts))
	}

	if !updatedTask.Execution.Attempts[0].Success {
		t.Errorf("Expected attempt to be successful")
	}
}

// TestExecuteTaskRetry tests task execution with retries
func TestExecuteTaskRetry(t *testing.T) {
	// Setup temp directory
	tmpDir := t.TempDir()

	// Create test config
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path:       tmpDir,
			Name:       "test",
			TechStack:  "Go",
			MainBranch: "main",
		},
		Verification: config.VerificationConfig{
			Command:        "exit 1", // Fail verification
			TimeoutSeconds: 10,
		},
		CLI: config.CLIConfig{
			Tool: "claude",
			Models: map[string]string{
				"simple": "test-model",
			},
		},
		Execution: config.ExecutionConfig{
			MaxAttempts:   2,
			AutoCommit:    false,
			HaltOnFailure: false,
		},
		Files: config.FilesConfig{},
	}

	// Create storage
	store := storage.NewStorage(cfg)

	// Create test task
	task := &storage.Task{
		ID:          "1.1",
		Title:       "Test Task",
		Status:      "pending",
		Complexity:  "simple",
		Model:       "test-model",
		Description: "Test description",
		Metadata: storage.TaskMetadata{
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			MaxAttempts: 2,
		},
		Execution: storage.TaskExecution{
			Attempts: []storage.Attempt{},
		},
	}

	// Save task
	if err := store.SaveTask(task); err != nil {
		t.Fatalf("Failed to save task: %v", err)
	}

	// Create mock agent runner (success but verification will fail)
	runner := &mockAgentRunner{
		shouldSucceed: true,
		shouldError:   false,
	}

	// Execute task
	ctx := context.Background()
	success, err := executeTask(ctx, cfg, store, runner, task)

	if err != nil {
		t.Fatalf("executeTask returned error: %v", err)
	}

	if success {
		t.Errorf("Expected task to fail, but it succeeded")
	}

	// Verify task status
	updatedTask, err := store.GetTask("1.1")
	if err != nil {
		t.Fatalf("Failed to load updated task: %v", err)
	}

	if updatedTask.Status != "failed" {
		t.Errorf("Expected status 'failed', got '%s'", updatedTask.Status)
	}

	if len(updatedTask.Execution.Attempts) != 2 {
		t.Errorf("Expected 2 attempts, got %d", len(updatedTask.Execution.Attempts))
	}
}

// TestExecuteTaskAgentError tests handling of agent execution errors
func TestExecuteTaskAgentError(t *testing.T) {
	// Setup temp directory
	tmpDir := t.TempDir()

	// Create test config
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path:       tmpDir,
			Name:       "test",
			TechStack:  "Go",
			MainBranch: "main",
		},
		Verification: config.VerificationConfig{
			Command:        "echo 'verification passed'",
			TimeoutSeconds: 10,
		},
		CLI: config.CLIConfig{
			Tool: "claude",
			Models: map[string]string{
				"simple": "test-model",
			},
		},
		Execution: config.ExecutionConfig{
			MaxAttempts:   2,
			AutoCommit:    false,
			HaltOnFailure: false,
		},
		Files: config.FilesConfig{},
	}

	// Create storage
	store := storage.NewStorage(cfg)

	// Create test task
	task := &storage.Task{
		ID:          "1.1",
		Title:       "Test Task",
		Status:      "pending",
		Complexity:  "simple",
		Model:       "test-model",
		Description: "Test description",
		Metadata: storage.TaskMetadata{
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			MaxAttempts: 2,
		},
		Execution: storage.TaskExecution{
			Attempts: []storage.Attempt{},
		},
	}

	// Save task
	if err := store.SaveTask(task); err != nil {
		t.Fatalf("Failed to save task: %v", err)
	}

	// Create mock agent runner (error)
	runner := &mockAgentRunner{
		shouldSucceed: false,
		shouldError:   true,
		errorMessage:  "Mock agent error",
	}

	// Execute task
	ctx := context.Background()
	success, err := executeTask(ctx, cfg, store, runner, task)

	if err != nil {
		t.Fatalf("executeTask returned error: %v", err)
	}

	if success {
		t.Errorf("Expected task to fail, but it succeeded")
	}

	// Verify task status
	updatedTask, err := store.GetTask("1.1")
	if err != nil {
		t.Fatalf("Failed to load updated task: %v", err)
	}

	if updatedTask.Status != "failed" {
		t.Errorf("Expected status 'failed', got '%s'", updatedTask.Status)
	}

	// Should have attempted twice
	if len(updatedTask.Execution.Attempts) != 2 {
		t.Errorf("Expected 2 attempts, got %d", len(updatedTask.Execution.Attempts))
	}

	// All attempts should have errors
	for i, attempt := range updatedTask.Execution.Attempts {
		if attempt.Error == "" {
			t.Errorf("Attempt %d should have an error", i+1)
		}
	}
}

// TestFilterTasksAfterCheckpoint tests checkpoint filtering
func TestFilterTasksAfterCheckpoint(t *testing.T) {
	tasks := []*storage.Task{
		{ID: "1.1", Title: "Task 1.1"},
		{ID: "1.2", Title: "Task 1.2"},
		{ID: "1.3", Title: "Task 1.3"},
		{ID: "2.1", Title: "Task 2.1"},
	}

	// Filter after 1.2
	filtered := filterTasksAfterCheckpoint(tasks, "1.2")

	if len(filtered) != 2 {
		t.Errorf("Expected 2 tasks after checkpoint, got %d", len(filtered))
	}

	if filtered[0].ID != "1.3" {
		t.Errorf("Expected first task to be 1.3, got %s", filtered[0].ID)
	}

	if filtered[1].ID != "2.1" {
		t.Errorf("Expected second task to be 2.1, got %s", filtered[1].ID)
	}
}

// TestFilterTasksAfterCheckpointNotFound tests when checkpoint is not found
func TestFilterTasksAfterCheckpointNotFound(t *testing.T) {
	tasks := []*storage.Task{
		{ID: "1.1", Title: "Task 1.1"},
		{ID: "1.2", Title: "Task 1.2"},
	}

	// Filter after non-existent checkpoint
	filtered := filterTasksAfterCheckpoint(tasks, "9.9")

	if len(filtered) != 0 {
		t.Errorf("Expected 0 tasks when checkpoint not found, got %d", len(filtered))
	}
}
