package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ceffo/devloop/internal/agent"
	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/knowledge"
	"github.com/ceffo/devloop/internal/storage"
	"github.com/ceffo/devloop/internal/ui"
)

// mockAgentRunner is a mock implementation of AgentRunner for testing
type mockAgentRunner struct {
	shouldSucceed bool
	shouldError   bool
	errorMessage  string
}

func (m *mockAgentRunner) Run(model, prompt, logPath string) (*agent.Result, error) {
	return m.RunWithOutput(model, prompt, logPath, nil)
}

func (m *mockAgentRunner) RunWithOutput(model, prompt, logPath string, outputFn agent.OutputCallback) (*agent.Result, error) {
	// Create log directory and file
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(logPath, []byte("Mock agent output"), 0644); err != nil {
		return nil, err
	}

	if outputFn != nil {
		outputFn("Mock agent output")
	}

	result := &agent.Result{
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
			Agents: map[string]*config.AgentConfig{
				"claude": {
					Tool: "claude",
					Models: map[string]string{
						"simple": "test-model",
					},
				},
			},
		},
		Execution: config.ExecutionConfig{
			MaxAttempts:   2,
			AutoCommit:    false,
			HaltOnFailure: true,
		},
	}

	// Create storage
	store := storage.NewStorage(cfg)

	// Create test task
	task := &storage.Task{
		ID:          "1.1",
		Title:       "Test Task",
		Status:      "pending",
		Complexity:  "simple",
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
	success, err := executeTask(ctx, cfg, store, runner, task, "claude-haiku-4-5-20251001", newPlainNotifier(), &ui.UsageStats{}, &knowledge.NoopClient{})

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
			Agents: map[string]*config.AgentConfig{
				"claude": {
					Tool: "claude",
					Models: map[string]string{
						"simple": "test-model",
					},
				},
			},
		},
		Execution: config.ExecutionConfig{
			MaxAttempts:   2,
			AutoCommit:    false,
			HaltOnFailure: false,
		},
	}

	// Create storage
	store := storage.NewStorage(cfg)

	// Create test task
	task := &storage.Task{
		ID:          "1.1",
		Title:       "Test Task",
		Status:      "pending",
		Complexity:  "simple",
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
	success, err := executeTask(ctx, cfg, store, runner, task, "claude-haiku-4-5-20251001", newPlainNotifier(), &ui.UsageStats{}, &knowledge.NoopClient{})

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
			Agents: map[string]*config.AgentConfig{
				"claude": {
					Tool: "claude",
					Models: map[string]string{
						"simple": "test-model",
					},
				},
			},
		},
		Execution: config.ExecutionConfig{
			MaxAttempts:   2,
			AutoCommit:    false,
			HaltOnFailure: false,
		},
	}

	// Create storage
	store := storage.NewStorage(cfg)

	// Create test task
	task := &storage.Task{
		ID:          "1.1",
		Title:       "Test Task",
		Status:      "pending",
		Complexity:  "simple",
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
	success, err := executeTask(ctx, cfg, store, runner, task, "claude-haiku-4-5-20251001", newPlainNotifier(), &ui.UsageStats{}, &knowledge.NoopClient{})

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

// TestExecuteTaskContextCancellation tests that a cancelled context stops executeTask
func TestExecuteTaskContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

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
			Agents: map[string]*config.AgentConfig{
				"claude": {
					Tool: "claude",
					Models: map[string]string{
						"simple": "test-model",
					},
				},
			},
		},
		Execution: config.ExecutionConfig{
			MaxAttempts:   3,
			AutoCommit:    false,
			HaltOnFailure: false,
		},
	}

	store := storage.NewStorage(cfg)

	task := &storage.Task{
		ID:          "1.1",
		Title:       "Test Task",
		Status:      "pending",
		Complexity:  "simple",
		Description: "Test description",
		Metadata: storage.TaskMetadata{
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			MaxAttempts: 3,
		},
		Execution: storage.TaskExecution{
			Attempts: []storage.Attempt{},
		},
	}

	if err := store.SaveTask(task); err != nil {
		t.Fatalf("Failed to save task: %v", err)
	}

	runner := &mockAgentRunner{
		shouldSucceed: true,
		shouldError:   false,
	}

	// Cancel the context before calling executeTask
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	success, err := executeTask(ctx, cfg, store, runner, task, "test-model", newPlainNotifier(), &ui.UsageStats{}, &knowledge.NoopClient{})

	if err != nil {
		t.Fatalf("executeTask returned unexpected error: %v", err)
	}

	// With a pre-cancelled context, executeTask should return false without executing
	if success {
		t.Errorf("Expected task to not succeed when context is cancelled")
	}
}

// mockOutputRunner is a mock agent runner that returns a configurable output string.
type mockOutputRunner struct {
	output  string
	success bool
}

func (m *mockOutputRunner) Run(model, prompt, logPath string) (*agent.Result, error) {
	return m.RunWithOutput(model, prompt, logPath, nil)
}

func (m *mockOutputRunner) RunWithOutput(model, prompt, logPath string, outputFn agent.OutputCallback) (*agent.Result, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(logPath, []byte(m.output), 0644); err != nil {
		return nil, err
	}
	if outputFn != nil {
		outputFn(m.output)
	}
	return &agent.Result{
		LogPath: logPath,
		Output:  m.output,
		Success: m.success,
	}, nil
}

// TestRunCoordinatorProceed verifies that ##DEVLOOP:PROCEED## returns decomposed=false.
func TestRunCoordinatorProceed(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path:      tmpDir,
			Name:      "test",
			TechStack: "Go",
		},
		Storage: config.StorageConfig{Backend: "jsonl"},
		Execution: config.ExecutionConfig{
			MaxAttempts: 2,
			Coordinator: config.CoordinatorConfig{
				Enabled:     true,
				MaxSubtasks: 5,
			},
		},
	}

	store := storage.NewStorage(cfg)

	task := &storage.Task{
		ID:          "DEV-1",
		Title:       "Test task",
		Description: "A test task",
		Complexity:  "complex",
		Status:      "pending",
	}

	runner := &mockOutputRunner{
		output:  "Analyzing task...\n##DEVLOOP:PROCEED##\n",
		success: true,
	}

	ctx := context.Background()
	decomposed, subtasks, err := runCoordinator(ctx, cfg, store, runner, task, "test-model", newPlainNotifier())

	if err != nil {
		t.Fatalf("runCoordinator returned unexpected error: %v", err)
	}
	if decomposed {
		t.Errorf("Expected decomposed=false for PROCEED sentinel, got true")
	}
	if len(subtasks) != 0 {
		t.Errorf("Expected no subtasks for PROCEED, got %d", len(subtasks))
	}
}

// TestRunCoordinatorDecomposedJSONL verifies the JSONL decompose path:
// reads coordinator-output.json, saves subtasks, and returns decomposed=true.
func TestRunCoordinatorDecomposedJSONL(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path:      tmpDir,
			Name:      "test",
			TechStack: "Go",
		},
		Storage: config.StorageConfig{Backend: "jsonl"},
		Execution: config.ExecutionConfig{
			MaxAttempts: 2,
			Coordinator: config.CoordinatorConfig{
				Enabled:     true,
				MaxSubtasks: 5,
			},
		},
	}

	// Write coordinator-output.json
	devloopDir := filepath.Join(tmpDir, ".devloop")
	if err := os.MkdirAll(devloopDir, 0755); err != nil {
		t.Fatalf("Failed to create .devloop dir: %v", err)
	}
	outputJSON := `[
		{
			"title": "Subtask A",
			"description": "Do part A",
			"complexity": "simple",
			"acceptance_criteria": ["A works"],
			"blocked_by": [],
			"tags": ["part-a"]
		},
		{
			"title": "Subtask B",
			"description": "Do part B",
			"complexity": "moderate",
			"acceptance_criteria": ["B works"],
			"blocked_by": [],
			"tags": ["part-b"]
		}
	]`
	if err := os.WriteFile(filepath.Join(devloopDir, "coordinator-output.json"), []byte(outputJSON), 0644); err != nil {
		t.Fatalf("Failed to write coordinator-output.json: %v", err)
	}

	store := storage.NewStorage(cfg)

	task := &storage.Task{
		ID:          "DEV-1",
		Title:       "Complex task",
		Description: "A complex task to decompose",
		Complexity:  "complex",
		Status:      "pending",
	}

	// Save parent task so UpdateTask can find it after decomposition.
	if err := store.SaveTask(task); err != nil {
		t.Fatalf("Failed to save parent task: %v", err)
	}

	runner := &mockOutputRunner{
		output:  "Creating subtasks...\n##DEVLOOP:DECOMPOSED##\n",
		success: true,
	}

	ctx := context.Background()
	decomposed, subtasks, err := runCoordinator(ctx, cfg, store, runner, task, "test-model", newPlainNotifier())

	if err != nil {
		t.Fatalf("runCoordinator returned unexpected error: %v", err)
	}
	if !decomposed {
		t.Errorf("Expected decomposed=true for DECOMPOSED sentinel, got false")
	}
	if len(subtasks) != 2 {
		t.Fatalf("Expected 2 subtasks, got %d", len(subtasks))
	}

	// Verify subtask IDs follow the parent.N pattern
	if subtasks[0].ID != "DEV-1.1" {
		t.Errorf("Expected first subtask ID 'DEV-1.1', got %q", subtasks[0].ID)
	}
	if subtasks[1].ID != "DEV-1.2" {
		t.Errorf("Expected second subtask ID 'DEV-1.2', got %q", subtasks[1].ID)
	}

	// Verify subtasks were persisted to the store
	saved, err := store.GetTask("DEV-1.1")
	if err != nil {
		t.Errorf("Subtask DEV-1.1 was not saved to store: %v", err)
	} else if saved.Title != "Subtask A" {
		t.Errorf("Expected saved title 'Subtask A', got %q", saved.Title)
	}

	saved2, err := store.GetTask("DEV-1.2")
	if err != nil {
		t.Errorf("Subtask DEV-1.2 was not saved to store: %v", err)
	} else if saved2.Complexity != "moderate" {
		t.Errorf("Expected subtask 2 complexity 'moderate', got %q", saved2.Complexity)
	}

	// Verify parent task was updated with in_progress status and DecomposedInto.
	parent, err := store.GetTask("DEV-1")
	if err != nil {
		t.Fatalf("Failed to load parent task after decomposition: %v", err)
	}
	if parent.Status != "in_progress" {
		t.Errorf("Expected parent status 'in_progress' after decomposition, got %q", parent.Status)
	}
	if len(parent.DecomposedInto) != 2 {
		t.Errorf("Expected parent DecomposedInto to have 2 entries, got %d", len(parent.DecomposedInto))
	}
}

// TestRunCoordinatorAgentFailure verifies that an agent failure returns an error.
func TestRunCoordinatorAgentFailure(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path:      tmpDir,
			Name:      "test",
			TechStack: "Go",
		},
		Storage: config.StorageConfig{Backend: "jsonl"},
		Execution: config.ExecutionConfig{
			MaxAttempts: 2,
		},
	}

	store := storage.NewStorage(cfg)

	task := &storage.Task{
		ID:         "DEV-1",
		Title:      "Test task",
		Complexity: "complex",
		Status:     "pending",
	}

	runner := &mockOutputRunner{
		output:  "error output",
		success: false, // agent reports failure
	}

	ctx := context.Background()
	_, _, err := runCoordinator(ctx, cfg, store, runner, task, "test-model", newPlainNotifier())
	if err == nil {
		t.Error("Expected error when agent returns failure, got nil")
	}
}

// TestResumeSkipsDecomposedParent verifies that an in_progress parent task with
// DecomposedInto set is NOT reset to pending during resume. Remaining pending
// subtasks should be returned as ready, and the coordinator will not be re-invoked
// because the parent never re-enters the ready queue.
func TestResumeSkipsDecomposedParent(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path:      tmpDir,
			Name:      "test",
			TechStack: "Go",
		},
		Storage: config.StorageConfig{Backend: "jsonl"},
		Execution: config.ExecutionConfig{
			MaxAttempts: 2,
		},
	}

	store := storage.NewStorage(cfg)

	// Parent task was already decomposed: in_progress with DecomposedInto set.
	parent := &storage.Task{
		ID:             "DEV-5",
		Title:          "Complex parent",
		Complexity:     "complex",
		Status:         "in_progress",
		DecomposedInto: []string{"DEV-5.1", "DEV-5.2"},
		Metadata:       storage.TaskMetadata{MaxAttempts: 2},
		Execution:      storage.TaskExecution{Attempts: []storage.Attempt{}},
	}

	// First subtask already completed.
	sub1 := &storage.Task{
		ID:        "DEV-5.1",
		Title:     "Subtask 1",
		Complexity: "simple",
		Status:    "completed",
		Metadata:  storage.TaskMetadata{MaxAttempts: 2},
		Execution: storage.TaskExecution{Attempts: []storage.Attempt{}},
	}

	// Second subtask still pending.
	sub2 := &storage.Task{
		ID:        "DEV-5.2",
		Title:     "Subtask 2",
		Complexity: "simple",
		Status:    "pending",
		Metadata:  storage.TaskMetadata{MaxAttempts: 2},
		Execution: storage.TaskExecution{Attempts: []storage.Attempt{}},
	}

	for _, task := range []*storage.Task{parent, sub1, sub2} {
		if err := store.SaveTask(task); err != nil {
			t.Fatalf("Failed to save task %s: %v", task.ID, err)
		}
	}

	// Build a session that reflects DEV-5.1 was already completed.
	session := &Session{
		ID:             "test-session",
		TasksCompleted: []string{"DEV-5.1"},
		TasksFailed:    []string{},
	}

	tasks, err := getReadyTasksForExecution(store, storage.Filter{}, "", true, session)
	if err != nil {
		t.Fatalf("getReadyTasksForExecution returned error: %v", err)
	}

	// Only DEV-5.2 should be ready; DEV-5 must NOT be in the list.
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 ready task, got %d: %v", len(tasks), taskIDs(tasks))
	}
	if tasks[0].ID != "DEV-5.2" {
		t.Errorf("Expected ready task to be DEV-5.2, got %q", tasks[0].ID)
	}

	// Verify DEV-5 is still in_progress (not reset to pending).
	savedParent, err := store.GetTask("DEV-5")
	if err != nil {
		t.Fatalf("Failed to load parent task: %v", err)
	}
	if savedParent.Status != "in_progress" {
		t.Errorf("Expected parent status to remain 'in_progress', got %q", savedParent.Status)
	}
	if len(savedParent.DecomposedInto) != 2 {
		t.Errorf("Expected parent DecomposedInto to have 2 entries, got %d", len(savedParent.DecomposedInto))
	}
}

// TestResumeNormalTasksAreReset verifies that ordinary in_progress/failed tasks
// (without DecomposedInto) are still reset to pending on resume.
func TestResumeNormalTasksAreReset(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path:      tmpDir,
			Name:      "test",
			TechStack: "Go",
		},
		Storage:   config.StorageConfig{Backend: "jsonl"},
		Execution: config.ExecutionConfig{MaxAttempts: 2},
	}

	store := storage.NewStorage(cfg)

	inProgress := &storage.Task{
		ID:        "DEV-1",
		Title:     "In-progress task",
		Complexity: "simple",
		Status:    "in_progress",
		Metadata:  storage.TaskMetadata{MaxAttempts: 2},
		Execution: storage.TaskExecution{Attempts: []storage.Attempt{}},
	}

	failed := &storage.Task{
		ID:        "DEV-2",
		Title:     "Failed task",
		Complexity: "simple",
		Status:    "failed",
		Metadata:  storage.TaskMetadata{MaxAttempts: 2},
		Execution: storage.TaskExecution{Attempts: []storage.Attempt{}},
	}

	for _, task := range []*storage.Task{inProgress, failed} {
		if err := store.SaveTask(task); err != nil {
			t.Fatalf("Failed to save task %s: %v", task.ID, err)
		}
	}

	session := &Session{
		ID:             "test-session",
		TasksCompleted: []string{},
		TasksFailed:    []string{},
	}

	tasks, err := getReadyTasksForExecution(store, storage.Filter{}, "", true, session)
	if err != nil {
		t.Fatalf("getReadyTasksForExecution returned error: %v", err)
	}

	// Both DEV-1 and DEV-2 should be reset to pending and returned as ready.
	if len(tasks) != 2 {
		t.Fatalf("Expected 2 ready tasks after reset, got %d: %v", len(tasks), taskIDs(tasks))
	}

	ids := map[string]bool{}
	for _, task := range tasks {
		ids[task.ID] = true
	}
	if !ids["DEV-1"] {
		t.Errorf("Expected DEV-1 to be in the ready list")
	}
	if !ids["DEV-2"] {
		t.Errorf("Expected DEV-2 to be in the ready list")
	}
}

// taskIDs is a test helper that returns a slice of task IDs for error messages.
func taskIDs(tasks []*storage.Task) []string {
	ids := make([]string, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	return ids
}
