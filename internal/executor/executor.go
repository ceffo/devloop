package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ceffo/devloop/internal/agent"
	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/knowledge"
	"github.com/ceffo/devloop/internal/prompts"
	"github.com/ceffo/devloop/internal/storage"
	"github.com/ceffo/devloop/internal/ui"
	"github.com/google/uuid"
)

// ExecuteDevLoop runs the main dev loop execution engine
// It executes tasks matching the filter criteria, handling retries, verification,
// auto-commits, and session checkpointing.
//
// Parameters:
//   - ctx: Context for cancellation (should be signal-aware from the caller)
//   - cfg: Configuration
//   - taskID: Filter by specific task ID (empty = all tasks)
//   - continueSession: Whether to resume from last checkpoint
//   - dryRun: If true, only show what would be executed without running
//   - agentName: Name of agent to use (empty = default agent)
//   - coordinateEnabled: If true, enable coordinator for this run (overrides config, does not persist)
func ExecuteDevLoop(ctx context.Context, cfg *config.Config, taskID string, continueSession bool, dryRun bool, agentName string, coordinateEnabled bool) error {
	// Apply --coordinate flag override without modifying the config file
	if coordinateEnabled {
		cfg.Execution.Coordinator.Enabled = true
	}
	// Use a derived context so we can cancel internally if needed (e.g. from TUI)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Initialize storage
	store, err := storage.NewTaskStore(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Load or create session
	session := LoadSession(cfg)
	if !continueSession {
		// Start fresh session - always create new on non-resume runs
		session = &Session{
			ID:             uuid.New().String(),
			StartedAt:      time.Now(),
			LastCheckpoint: "",
			TasksCompleted: []string{},
			TasksFailed:    []string{},
		}
	}

	// Save initial session state
	if err := SaveSession(cfg, session); err != nil {
		return fmt.Errorf("failed to save initial session: %w", err)
	}

	if dryRun {
		fmt.Printf("🔍 Dry run mode - showing what would be executed (Session: %s)\n\n", session.ID[:8])
	} else {
		fmt.Printf("🚀 Starting dev loop execution (Session: %s)\n\n", session.ID[:8])
	}

	// Build initial task list
	tasks, err := getReadyTasksForExecution(store, storage.Filter{}, taskID, continueSession, session)
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		fmt.Println("✓ No pending tasks to execute")
		return nil
	}

	fmt.Printf("Found %d task(s) ready to execute\n\n", len(tasks))

	// Get agent config
	agentConfig, err := cfg.CLI.GetAgent(agentName)
	if err != nil {
		return fmt.Errorf("failed to get agent configuration: %w", err)
	}

	// Resolve effective agent name for display and session metadata
	selectedAgentName := cfg.CLI.GetDefaultAgentName()
	if agentName != "" {
		selectedAgentName = agentName
	}

	// Populate session metadata: agent/model info and task snapshot
	session.AgentName = selectedAgentName
	session.ModelMap = agentConfig.Models
	taskSnapshots := make([]TaskSnapshot, 0, len(tasks))
	for _, t := range tasks {
		taskSnapshots = append(taskSnapshots, TaskSnapshot{
			ID:         t.ID,
			Title:      t.Title,
			Status:     t.Status,
			Complexity: t.Complexity,
		})
	}
	session.TaskSnapshot = taskSnapshots
	if err := SaveSession(cfg, session); err != nil {
		return fmt.Errorf("failed to save session metadata: %w", err)
	}

	// If dry-run mode, just print tasks and exit
	if dryRun {
		printDryRunSummary(tasks, agentConfig)
		return nil
	}

	// Initialize agent runner
	agentRunner, err := agent.NewAgentRunner(agentConfig.Tool)
	if err != nil {
		return fmt.Errorf("failed to create agent runner: %w", err)
	}

	// Initialize knowledge client
	knowledgeClient := knowledge.NewClient(cfg.Knowledge.Backend)

	// Initialize TUI notifier (falls back to plain text for non-TTY)
	notifier := NewNotifier()
	notifier.SetCancelFunc(cancel)
	notifier.Init(tasks, session.ID, selectedAgentName)

	// Execute tasks with dynamic dependency reassessment
	successCount := 0
	failureCount := 0
	executedTaskIDs := make(map[string]bool) // Track what we've already executed
	taskNumber := 0
	sessionStats := &ui.UsageStats{}

	for len(tasks) > 0 {
		// Get next task
		task := tasks[0]
		tasks = tasks[1:]

		// Skip if already executed
		if executedTaskIDs[task.ID] {
			continue
		}
		executedTaskIDs[task.ID] = true
		taskNumber++

		// Check for interrupts
		select {
		case <-ctx.Done():
			notifier.Log(ui.Warning("Execution interrupted by user"))
			notifier.Done(nil)
			_ = notifier.Wait()
			return printSummary(successCount, failureCount, session)
		default:
		}

		// Get model for this task based on its complexity
		model, err := agentConfig.GetModel(task.Complexity)
		if err != nil {
			notifier.TaskFailed(task.ID)
			notifier.Log(fmt.Sprintf("Invalid task complexity for %s: %v", task.ID, err))
			failureCount++
			session.TasksFailed = append(session.TasksFailed, task.ID)

			// Save session state
			CheckpointSession(session, task.ID)
			if err := SaveSession(cfg, session); err != nil {
				notifier.Log(fmt.Sprintf("Warning: failed to save session: %v", err))
			}

			if cfg.Execution.HaltOnFailure {
				notifier.Log("Halting execution due to task failure (halt_on_failure=true)")
				notifier.Done(nil)
				_ = notifier.Wait()
				return printSummary(successCount, failureCount, session)
			}
			continue
		}

		notifier.TaskStarted(task.ID, 1, task.Metadata.MaxAttempts)
		notifier.AgentStatusUpdate(selectedAgentName, model)
		notifier.Log(fmt.Sprintf("Task %d: %s - %s [%s/%s/%s]",
			taskNumber, task.ID, task.Title, task.Complexity, model, selectedAgentName))

		// Run coordinator (if enabled) for complex tasks only
		if cfg.Execution.Coordinator.Enabled && task.Complexity == "complex" {
			coordinatorModel := cfg.Execution.Coordinator.Model
			if coordinatorModel == "" {
				coordinatorModel = model
			}
			notifier.Log(fmt.Sprintf("  Running coordinator for %s...", task.ID))
			decomposed, subtasks, coordErr := runCoordinator(ctx, cfg, store, agentRunner, task, coordinatorModel, notifier)
			if coordErr != nil {
				notifier.Log(fmt.Sprintf("  ⚠ Coordinator failed for %s: %v (proceeding with direct execution)", task.ID, coordErr))
			} else if decomposed {
				notifier.Log(fmt.Sprintf("  ✓ Task %s decomposed into %d subtask(s)", task.ID, len(subtasks)))
				for _, st := range subtasks {
					if !executedTaskIDs[st.ID] {
						tasks = append(tasks, st)
					}
				}
				// Skip direct execution of the original task — subtasks handle the work
				continue
			}
		}

		// Execute task with retries, passing notifier for status updates
		success, err := executeTask(ctx, cfg, store, agentRunner, task, model, notifier, sessionStats, knowledgeClient)
		if err != nil {
			notifier.TaskFailed(task.ID)
			notifier.Log(fmt.Sprintf("Task %s error: %v", task.ID, err))
			failureCount++
			sessionStats.TasksFailed++
			notifier.UsageStatsUpdate(*sessionStats)
			session.TasksFailed = append(session.TasksFailed, task.ID)

			// Save session state
			CheckpointSession(session, task.ID)
			if err := SaveSession(cfg, session); err != nil {
				notifier.Log(fmt.Sprintf("Warning: failed to save session: %v", err))
			}

			if cfg.Execution.HaltOnFailure {
				notifier.Log("Halting execution due to task failure (halt_on_failure=true)")
				notifier.Done(nil)
				_ = notifier.Wait()
				return printSummary(successCount, failureCount, session)
			}
			continue
		}

		if success {
			notifier.TaskCompleted(task.ID)
			successCount++
			sessionStats.TasksCompleted++
			notifier.UsageStatsUpdate(*sessionStats)
			session.TasksCompleted = append(session.TasksCompleted, task.ID)

			// After successful completion, check for newly-unblocked tasks
			newlyReadyTasks, err := getReadyTasksForExecution(store, storage.Filter{}, taskID, false, session)
			if err != nil {
				notifier.Log(fmt.Sprintf("Warning: failed to check for newly ready tasks: %v", err))
			} else {
				// Add newly ready tasks that we haven't executed yet
				for _, newTask := range newlyReadyTasks {
					if !executedTaskIDs[newTask.ID] {
						// Check if already in queue
						alreadyQueued := false
						for _, queuedTask := range tasks {
							if queuedTask.ID == newTask.ID {
								alreadyQueued = true
								break
							}
						}
						if !alreadyQueued {
							tasks = append(tasks, newTask)
							notifier.Log(fmt.Sprintf("Task %s is now ready (dependencies satisfied)", newTask.ID))
						}
					}
				}
			}
		} else {
			notifier.TaskFailed(task.ID)
			notifier.Log(fmt.Sprintf("Task %s failed after all attempts", task.ID))
			failureCount++
			sessionStats.TasksFailed++
			notifier.UsageStatsUpdate(*sessionStats)
			session.TasksFailed = append(session.TasksFailed, task.ID)

			if cfg.Execution.HaltOnFailure {
				notifier.Log("Halting execution due to task failure (halt_on_failure=true)")
				// Save session state
				CheckpointSession(session, task.ID)
				if err := SaveSession(cfg, session); err != nil {
					notifier.Log(fmt.Sprintf("Warning: failed to save session: %v", err))
				}
				notifier.Done(nil)
				_ = notifier.Wait()
				return printSummary(successCount, failureCount, session)
			}
		}

		// Checkpoint after each task
		CheckpointSession(session, task.ID)
		if err := SaveSession(cfg, session); err != nil {
			notifier.Log(fmt.Sprintf("Warning: failed to save session: %v", err))
		}
	}

	// Sync storage if backend supports it; failures are non-fatal warnings.
	if syncer, ok := store.(storage.Syncer); ok {
		if err := syncer.Sync(); err != nil {
			notifier.Log(fmt.Sprintf("Warning: sync failed: %v", err))
		}
	}

	notifier.AgentStatusUpdate("", "")
	notifier.Done(nil)
	_ = notifier.Wait()
	return printSummary(successCount, failureCount, session)
}

// getReadyTasksForExecution returns the list of tasks ready to execute based on filters
func getReadyTasksForExecution(store storage.TaskStore, filter storage.Filter, taskID string, continueSession bool, session *Session) ([]*storage.Task, error) {
	// On resume, reset in_progress and failed tasks to pending so they get retried.
	// Exception: in_progress tasks that were previously decomposed by the coordinator
	// retain their status so the coordinator is not re-invoked; their subtasks remain
	// pending and will be discovered via QueryReadyTasks naturally.
	if continueSession {
		for _, status := range []string{"failed", "in_progress"} {
			extraTasks, err := store.QueryTasks(storage.Filter{Status: status})
			if err != nil {
				return nil, fmt.Errorf("failed to query %s tasks for resume: %w", status, err)
			}
			for _, t := range extraTasks {
				// Skip decomposed parent tasks — subtasks handle the remaining work.
				if len(t.DecomposedInto) > 0 {
					continue
				}
				t.Status = "pending"
				if err := store.UpdateTask(t); err != nil {
					return nil, fmt.Errorf("failed to reset %s task %s: %w", status, t.ID, err)
				}
			}
		}
	}

	// Get all ready tasks (pending and unblocked)
	tasks, err := store.QueryReadyTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to query ready tasks: %w", err)
	}

	// Filter specific task if requested
	if taskID != "" {
		var filteredTasks []*storage.Task
		for _, task := range tasks {
			if task.ID == taskID {
				filteredTasks = []*storage.Task{task}
				break
			}
		}
		tasks = filteredTasks
	}

	// On resume, skip tasks already completed in this session rather than
	// using position-based filtering (which breaks when the checkpoint task
	// is not in the pending list, e.g. it failed or was interrupted).
	if continueSession {
		tasks = filterTasksForResume(tasks, session)
	}

	return tasks, nil
}

// executeTask executes a single task with retry logic
// Returns (success, error)
// sessionStats is updated in place and UsageStatsUpdate is called after each agent call.
func executeTask(ctx context.Context, cfg *config.Config, store storage.TaskStore, runner agent.Runner, task *storage.Task, model string, notifier Notifier, sessionStats *ui.UsageStats, knowledgeClient knowledge.Client) (bool, error) {
	// Mark task as in progress
	task.Status = "in_progress"
	task.Metadata.UpdatedAt = time.Now()
	if err := store.UpdateTask(task); err != nil {
		return false, fmt.Errorf("failed to update task status: %w", err)
	}

	// Track previous error for retry prompts
	var previousError string

	// Attempt execution up to MaxAttempts times; fall back to config default when unset.
	maxAttempts := task.Metadata.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = cfg.Execution.MaxAttempts
	}
	for attemptNum := 1; attemptNum <= maxAttempts; attemptNum++ {
		// Check for interrupts
		select {
		case <-ctx.Done():
			return false, nil
		default:
		}

		notifier.TaskStarted(task.ID, attemptNum, task.Metadata.MaxAttempts)
		notifier.Log(fmt.Sprintf("  Attempt %d/%d for %s...", attemptNum, task.Metadata.MaxAttempts, task.ID))

		// Generate prompt with context
		prompt, err := prompts.RenderTaskPrompt(cfg, task, attemptNum, previousError, knowledgeClient)
		if err != nil {
			return false, fmt.Errorf("failed to generate prompt: %w", err)
		}

		// Create log path
		timestamp := time.Now().Format("20060102-150405")
		logPath := filepath.Join(cfg.Project.Path, ".devloop", "logs",
			fmt.Sprintf("agent-%s-attempt%d-%s.log", task.ID, attemptNum, timestamp))

		// Execute agent
		startTime := time.Now()

		notifier.Log(fmt.Sprintf("  Running AI agent (%s)...", model))
		agentResult, err := runner.RunWithOutput(model, prompt, logPath, notifier.AgentOutput)
		duration := int(time.Since(startTime).Seconds())

		// Update and emit usage stats after each agent call
		sessionStats.AgentCalls++
		sessionStats.TotalDuration += duration
		sessionStats.LastTaskDuration = duration
		notifier.UsageStatsUpdate(*sessionStats)

		if err != nil {
			return false, fmt.Errorf("agent execution failed: %w", err)
		}

		// Create attempt record
		attempt := storage.Attempt{
			AttemptNumber: attemptNum,
			StartedAt:     startTime,
			CompletedAt:   time.Now(),
			Duration:      duration,
			Model:         model,
			Success:       agentResult.Success,
			LogPath:       logPath,
		}

		if !agentResult.Success {
			// Agent execution failed
			attempt.Result = "error"
			attempt.Error = fmt.Sprintf("Agent failed: %v", agentResult.Error)
			task.AddAttempt(attempt)
			previousError = attempt.Error

			notifier.Log(fmt.Sprintf("  ✗ Agent execution failed: %v", agentResult.Error))

			// Save task state
			if err := store.UpdateTask(task); err != nil {
				return false, fmt.Errorf("failed to save task: %w", err)
			}

			// Continue to next attempt
			continue
		}

		notifier.Log(fmt.Sprintf("  ✓ Agent execution completed (%ds)", duration))

		// Run verification
		notifier.Log("  Running verification...")
		verifyResult, err := RunVerification(cfg, task.ID)
		if err != nil {
			return false, fmt.Errorf("verification execution failed: %w", err)
		}

		attempt.VerifyLogPath = verifyResult.LogPath

		if !verifyResult.Success {
			// Verification failed
			attempt.Result = "failed"
			attempt.Success = false
			attempt.Error = "Verification failed (exit code non-zero)"
			task.AddAttempt(attempt)
			previousError = fmt.Sprintf("Verification failed:\n%s", verifyResult.Output)

			notifier.Log(fmt.Sprintf("  ✗ Verification failed (%ds)", verifyResult.Duration))

			// Save task state
			if err := store.UpdateTask(task); err != nil {
				return false, fmt.Errorf("failed to save task: %w", err)
			}

			// Continue to next attempt
			continue
		}

		notifier.Log(fmt.Sprintf("  ✓ Verification passed (%ds)", verifyResult.Duration))

		// Success! Mark attempt as successful
		attempt.Result = "passed"
		attempt.Success = true
		task.AddAttempt(attempt)

		// Auto-commit if enabled
		var commitHash string
		if cfg.Execution.AutoCommit {
			notifier.Log("  Creating git commit...")
			hash, err := AutoCommit(cfg, task)
			if err != nil {
				// Commit failure shouldn't fail the task, just warn
				notifier.Log(fmt.Sprintf("  ⚠ Auto-commit failed: %v", err))
			} else {
				commitHash = hash
				notifier.Log(fmt.Sprintf("  ✓ Committed as %s", commitHash))
			}
		}

		// Mark task as completed
		task.Status = "completed"
		task.Results = &storage.TaskResults{
			VerificationOutput: verifyResult.Output,
			CommitHash:         commitHash,
			CompletedAt:        time.Now(),
		}
		task.Metadata.UpdatedAt = time.Now()

		// Save final task state
		if err := store.UpdateTask(task); err != nil {
			return false, fmt.Errorf("failed to save completed task: %w", err)
		}

		return true, nil
	}

	// All attempts exhausted
	task.Status = "failed"
	task.Metadata.UpdatedAt = time.Now()
	if err := store.UpdateTask(task); err != nil {
		return false, fmt.Errorf("failed to save failed task: %w", err)
	}

	return false, nil
}

// filterTasksAfterCheckpoint returns tasks that come after the checkpoint task ID
func filterTasksAfterCheckpoint(tasks []*storage.Task, checkpointID string) []*storage.Task {
	// Find the checkpoint task
	foundCheckpoint := false
	var filtered []*storage.Task

	for _, task := range tasks {
		if foundCheckpoint {
			filtered = append(filtered, task)
		}
		if task.ID == checkpointID {
			foundCheckpoint = true
		}
	}

	return filtered
}

// filterTasksForResume excludes tasks already successfully completed in this session.
// Used during resume to skip work already done while retrying failed/incomplete tasks.
func filterTasksForResume(tasks []*storage.Task, session *Session) []*storage.Task {
	completedSet := make(map[string]bool, len(session.TasksCompleted))
	for _, id := range session.TasksCompleted {
		completedSet[id] = true
	}
	var filtered []*storage.Task
	for _, task := range tasks {
		if !completedSet[task.ID] {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// printSummary displays execution summary
func printSummary(successCount, failureCount int, session *Session) error {
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("Execution Summary")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("Session: %s\n", session.ID[:8])
	fmt.Printf("Started: %s\n", session.StartedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("\n")

	if successCount > 0 {
		fmt.Println(ui.Success(fmt.Sprintf("Tasks completed: %d", successCount)))
	}
	if failureCount > 0 {
		fmt.Println(ui.Error(fmt.Sprintf("Tasks failed: %d", failureCount)))
	}

	total := successCount + failureCount
	if total > 0 {
		successRate := float64(successCount) / float64(total) * 100
		fmt.Printf("\nSuccess rate: %.1f%%\n", successRate)
	}

	fmt.Println("═══════════════════════════════════════════════════════════")

	if failureCount > 0 {
		return fmt.Errorf("execution completed with %d failure(s)", failureCount)
	}

	return nil
}

// coordinatorSubtaskDef is the shape of each object in coordinator-output.json.
type coordinatorSubtaskDef struct {
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Complexity         string   `json:"complexity"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	BlockedBy          []string `json:"blocked_by"`
	Tags               []string `json:"tags"`
}

// runCoordinator runs the coordinator agent for a task to decide whether to decompose it.
// It renders the coordinator prompt, executes the agent, and parses sentinel output.
//
// Returns (decomposed, subtasks, error):
//   - decomposed=false: the original task should proceed to executeTask as-is.
//   - decomposed=true: the task was split; subtasks have been saved to the store
//     and should be appended to the execution queue.
func runCoordinator(ctx context.Context, cfg *config.Config, store storage.TaskStore, runner agent.Runner, task *storage.Task, model string, notifier Notifier) (bool, []*storage.Task, error) {
	// Render coordinator prompt
	prompt, err := prompts.RenderCoordinatorPrompt(cfg, task)
	if err != nil {
		return false, nil, fmt.Errorf("failed to render coordinator prompt: %w", err)
	}

	// Build log path
	timestamp := time.Now().Format("20060102-150405")
	logPath := filepath.Join(cfg.Project.Path, ".devloop", "logs",
		fmt.Sprintf("coordinator-%s-%s.log", task.ID, timestamp))

	// Execute agent
	result, err := runner.RunWithOutput(model, prompt, logPath, notifier.AgentOutput)
	if err != nil {
		return false, nil, fmt.Errorf("coordinator agent execution failed: %w", err)
	}
	if !result.Success {
		return false, nil, fmt.Errorf("coordinator agent returned failure for task %s", task.ID)
	}

	output := result.Output

	if !strings.Contains(output, prompts.SentinelDecomposed) {
		// ##DEVLOOP:PROCEED## or no sentinel: execute the task as-is
		return false, nil, nil
	}

	// ##DEVLOOP:DECOMPOSED## — gather newly created subtasks
	var subtasks []*storage.Task

	if cfg.Storage.Backend == "beads" {
		// Re-query bd ready to surface subtasks the agent created via bd commands.
		// BeadsStore uses context-aware methods; convert to any first to allow assertion.
		type readyQuerier interface {
			QueryReadyTasks(ctx context.Context) ([]*storage.Task, error)
		}
		beadsStore, ok := any(store).(readyQuerier)
		if !ok {
			return false, nil, fmt.Errorf("storage backend is beads but store does not implement QueryReadyTasks(context.Context)")
		}
		subtasks, err = beadsStore.QueryReadyTasks(ctx)
		if err != nil {
			return false, nil, fmt.Errorf("failed to query beads ready tasks after decompose: %w", err)
		}
	} else {
		// JSONL backend: read coordinator-output.json and persist subtasks
		subtasks, err = loadAndSaveCoordinatorOutput(cfg, store, task)
		if err != nil {
			return false, nil, fmt.Errorf("failed to load coordinator output: %w", err)
		}
	}

	// Record decomposition on the parent task so session recovery can detect it.
	// The parent is marked in_progress with DecomposedInto set; this prevents the
	// coordinator from being re-invoked if execution is resumed.
	subtaskIDs := make([]string, len(subtasks))
	for i, st := range subtasks {
		subtaskIDs[i] = st.ID
	}
	task.Status = "in_progress"
	task.DecomposedInto = subtaskIDs
	if err := store.UpdateTask(task); err != nil {
		// Non-fatal: decomposition succeeded; log the warning but continue.
		notifier.Log(fmt.Sprintf("  ⚠ Warning: failed to record decomposition on parent task %s: %v", task.ID, err))
	}

	return true, subtasks, nil
}

// loadAndSaveCoordinatorOutput reads .devloop/coordinator-output.json, converts each entry
// into a storage.Task (with IDs derived from the parent), and saves them to the store.
func loadAndSaveCoordinatorOutput(cfg *config.Config, store storage.TaskStore, parentTask *storage.Task) ([]*storage.Task, error) {
	outputPath := filepath.Join(cfg.Project.Path, ".devloop", "coordinator-output.json")

	data, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read coordinator-output.json: %w", err)
	}

	var defs []coordinatorSubtaskDef
	if err := json.Unmarshal(data, &defs); err != nil {
		return nil, fmt.Errorf("failed to parse coordinator-output.json: %w", err)
	}

	now := time.Now()
	tasks := make([]*storage.Task, 0, len(defs))
	for i, def := range defs {
		subtask := &storage.Task{
			ID:                 fmt.Sprintf("%s.%d", parentTask.ID, i+1),
			Title:              def.Title,
			Description:        def.Description,
			Complexity:         def.Complexity,
			AcceptanceCriteria: def.AcceptanceCriteria,
			BlockedBy:          def.BlockedBy,
			Tags:               def.Tags,
			Status:             "pending",
			Metadata: storage.TaskMetadata{
				CreatedAt:   now,
				UpdatedAt:   now,
				SourceType:  "coordinator",
				MaxAttempts: cfg.Execution.MaxAttempts,
			},
			Execution: storage.TaskExecution{
				Attempts: []storage.Attempt{},
			},
		}
		if err := store.SaveTask(subtask); err != nil {
			return nil, fmt.Errorf("failed to save subtask %s: %w", subtask.ID, err)
		}
		tasks = append(tasks, subtask)
	}

	return tasks, nil
}

// printDryRunSummary prints a summary of tasks that would be executed
func printDryRunSummary(tasks []*storage.Task, agentConfig *config.AgentConfig) {
	fmt.Println("📋 Tasks that would be executed:")
	fmt.Println()

	for i, task := range tasks {
		fmt.Printf("%d. [%s] %s\n", i+1, task.ID, task.Title)
		model, _ := agentConfig.GetModel(task.Complexity)
		fmt.Printf("   Complexity: %s | Model: %s\n", task.Complexity, model)
		if len(task.BlockedBy) > 0 {
			fmt.Printf("   Dependencies: %v (all completed)\n", task.BlockedBy)
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d task(s) ready to execute\n", len(tasks))
}
