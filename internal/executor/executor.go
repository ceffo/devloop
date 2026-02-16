package executor

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ceffo/devloop/internal/agent"
	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/prompts"
	"github.com/ceffo/devloop/internal/storage"
	"github.com/ceffo/devloop/internal/ui"
)

// ExecuteDevLoop runs the main dev loop execution engine
// It executes tasks matching the filter criteria, handling retries, verification,
// auto-commits, and session checkpointing.
//
// Parameters:
//   - cfg: Configuration
//   - wave: Filter by wave number (0 = all waves)
//   - taskID: Filter by specific task ID (empty = all tasks)
//   - continueSession: Whether to resume from last checkpoint
func ExecuteDevLoop(cfg *config.Config, wave int, taskID string, continueSession bool) error {
	// Setup signal handling for graceful interrupts
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\n⚠️  Interrupt received, stopping after current task...")
		cancel()
	}()

	// Initialize storage
	store := storage.NewStorage(cfg)

	// Load or create session
	session := LoadSession(cfg)
	if !continueSession {
		// Start fresh session
		session = LoadSession(cfg)
		if session.LastCheckpoint != "" {
			// Reset session for new run
			session = &Session{
				ID:             session.ID,
				StartedAt:      time.Now(),
				LastCheckpoint: "",
				TasksCompleted: []string{},
				TasksFailed:    []string{},
			}
		}
	}

	// Save initial session state
	if err := SaveSession(cfg, session); err != nil {
		return fmt.Errorf("failed to save initial session: %w", err)
	}

	fmt.Printf("🚀 Starting dev loop execution (Session: %s)\n\n", session.ID[:8])

	// Build filter for task query
	filter := storage.Filter{
		Status:    "pending",
		BlockedBy: []string{}, // Only unblocked tasks
	}
	if wave > 0 {
		filter.Wave = wave
	}

	// Query executable tasks
	tasks, err := store.QueryTasks(filter)
	if err != nil {
		return fmt.Errorf("failed to query tasks: %w", err)
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

	// Handle resume from checkpoint
	if continueSession && session.LastCheckpoint != "" {
		tasks = filterTasksAfterCheckpoint(tasks, session.LastCheckpoint)
	}

	if len(tasks) == 0 {
		fmt.Println("✓ No pending tasks to execute")
		return nil
	}

	fmt.Printf("Found %d task(s) to execute\n\n", len(tasks))

	// Initialize agent runner
	agentRunner, err := agent.NewAgentRunner(cfg.CLI.Tool)
	if err != nil {
		return fmt.Errorf("failed to create agent runner: %w", err)
	}

	// Execute tasks
	successCount := 0
	failureCount := 0

	for i, task := range tasks {
		// Check for interrupts
		select {
		case <-ctx.Done():
			fmt.Println("\n" + ui.Warning("Execution interrupted by user"))
			return printSummary(successCount, failureCount, session)
		default:
		}

		fmt.Printf("═══════════════════════════════════════════════════════════\n")
		fmt.Printf("Task %d/%d: %s - %s\n", i+1, len(tasks), task.ID, task.Title)
		fmt.Printf("Complexity: %s | Model: %s\n", task.Complexity, task.Model)
		fmt.Printf("═══════════════════════════════════════════════════════════\n\n")

		// Execute task with retries
		success, err := executeTask(ctx, cfg, store, agentRunner, task)
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Task execution error: %v", err)))
			fmt.Println()
			failureCount++
			session.TasksFailed = append(session.TasksFailed, task.ID)

			// Save session state
			CheckpointSession(session, task.ID)
			if err := SaveSession(cfg, session); err != nil {
				fmt.Printf("Warning: failed to save session: %v\n", err)
			}

			if cfg.Execution.HaltOnFailure {
				fmt.Println("⚠️  Halting execution due to task failure (halt_on_failure=true)")
				return printSummary(successCount, failureCount, session)
			}
			continue
		}

		if success {
			fmt.Println(ui.Success("Task completed successfully"))
			fmt.Println()
			successCount++
			session.TasksCompleted = append(session.TasksCompleted, task.ID)
		} else {
			fmt.Println(ui.Error("Task failed after all attempts"))
			fmt.Println()
			failureCount++
			session.TasksFailed = append(session.TasksFailed, task.ID)

			if cfg.Execution.HaltOnFailure {
				fmt.Println("⚠️  Halting execution due to task failure (halt_on_failure=true)")
				// Save session state
				CheckpointSession(session, task.ID)
				if err := SaveSession(cfg, session); err != nil {
					fmt.Printf("Warning: failed to save session: %v\n", err)
				}
				return printSummary(successCount, failureCount, session)
			}
		}

		// Checkpoint after each task
		CheckpointSession(session, task.ID)
		if err := SaveSession(cfg, session); err != nil {
			fmt.Printf("Warning: failed to save session: %v\n", err)
		}
	}

	return printSummary(successCount, failureCount, session)
}

// executeTask executes a single task with retry logic
// Returns (success, error)
func executeTask(ctx context.Context, cfg *config.Config, store *storage.Storage, runner agent.AgentRunner, task *storage.Task) (bool, error) {
	// Mark task as in progress
	task.Status = "in_progress"
	task.Metadata.UpdatedAt = time.Now()
	if err := store.UpdateTask(task); err != nil {
		return false, fmt.Errorf("failed to update task status: %w", err)
	}

	// Track previous error for retry prompts
	var previousError string

	// Attempt execution up to MaxAttempts times
	for attemptNum := 1; attemptNum <= task.Metadata.MaxAttempts; attemptNum++ {
		// Check for interrupts
		select {
		case <-ctx.Done():
			return false, nil
		default:
		}

		fmt.Printf("  Attempt %d/%d...\n", attemptNum, task.Metadata.MaxAttempts)

		// Generate prompt with context
		prompt, err := prompts.RenderTaskPrompt(cfg, task, attemptNum, previousError)
		if err != nil {
			return false, fmt.Errorf("failed to generate prompt: %w", err)
		}

		// Create log path
		timestamp := time.Now().Format("20060102-150405")
		logPath := filepath.Join(cfg.Project.Path, ".devloop", "logs",
			fmt.Sprintf("agent-%s-attempt%d-%s.log", task.ID, attemptNum, timestamp))

		// Execute agent
		startTime := time.Now()
		fmt.Printf("  → Running AI agent (%s)...\n", task.Model)

		agentResult, err := runner.Run(task.Model, prompt, logPath)
		duration := int(time.Since(startTime).Seconds())

		if err != nil {
			return false, fmt.Errorf("agent execution failed: %w", err)
		}

		// Create attempt record
		attempt := storage.Attempt{
			AttemptNumber: attemptNum,
			StartedAt:     startTime,
			CompletedAt:   time.Now(),
			Duration:      duration,
			Model:         task.Model,
			Success:       agentResult.Success,
			LogPath:       logPath,
		}

		if !agentResult.Success {
			// Agent execution failed
			attempt.Result = "error"
			attempt.Error = fmt.Sprintf("Agent failed: %v", agentResult.Error)
			task.AddAttempt(attempt)
			previousError = attempt.Error

			fmt.Printf("  ✗ Agent execution failed: %v\n", agentResult.Error)
			fmt.Printf("  Log: %s\n\n", logPath)

			// Save task state
			if err := store.UpdateTask(task); err != nil {
				return false, fmt.Errorf("failed to save task: %w", err)
			}

			// Continue to next attempt
			continue
		}

		fmt.Printf("  ✓ Agent execution completed (%ds)\n", duration)

		// Run verification
		fmt.Printf("  → Running verification...\n")
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

			fmt.Printf("  ✗ Verification failed (%ds)\n", verifyResult.Duration)
			fmt.Printf("  Log: %s\n\n", verifyResult.LogPath)

			// Save task state
			if err := store.UpdateTask(task); err != nil {
				return false, fmt.Errorf("failed to save task: %w", err)
			}

			// Continue to next attempt
			continue
		}

		fmt.Printf("  ✓ Verification passed (%ds)\n", verifyResult.Duration)

		// Success! Mark attempt as successful
		attempt.Result = "passed"
		attempt.Success = true
		task.AddAttempt(attempt)

		// Auto-commit if enabled
		var commitHash string
		if cfg.Execution.AutoCommit {
			fmt.Printf("  → Creating git commit...\n")
			hash, err := AutoCommit(cfg, task)
			if err != nil {
				// Commit failure shouldn't fail the task, just warn
				fmt.Printf("  ⚠ Auto-commit failed: %v\n", err)
			} else {
				commitHash = hash
				fmt.Printf("  ✓ Committed as %s\n", commitHash)
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
