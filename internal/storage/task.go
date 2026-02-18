package storage

import "time"

// Task represents an atomic unit of work in the devloop system
type Task struct {
	// Core identification
	ID    string `json:"id"`    // Task ID (e.g., "DEV-1")
	Title string `json:"title"` // Brief task title

	// Status and state
	Status     string `json:"status"`     // pending, in_progress, completed, failed, blocked, archived
	Complexity string `json:"complexity"` // simple, moderate, complex (maps to AI model)

	// Task content
	Description        string   `json:"description"`                   // Detailed task description
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"` // Bulleted list of criteria

	// Dependency tracking
	BlockedBy []string `json:"blocked_by,omitempty"` // Task IDs that must complete first
	Tags      []string `json:"tags,omitempty"`       // Optional tags for categorization

	// Metadata
	Metadata TaskMetadata `json:"metadata"` // Creation and update tracking

	// Execution tracking
	Execution TaskExecution `json:"execution"` // Attempts and duration

	// Results (when completed)
	Results *TaskResults `json:"results,omitempty"` // Verification output, commit hash
}

// TaskMetadata tracks task creation and updates
type TaskMetadata struct {
	CreatedAt      time.Time `json:"created_at"`                 // When task was created
	UpdatedAt      time.Time `json:"updated_at"`                 // Last update timestamp
	SourceType     string    `json:"source_type"`                // e.g., "todo", "manual"
	SourceTodoItem string    `json:"source_todo_item,omitempty"` // Reference to original TODO item
	MaxAttempts    int       `json:"max_attempts"`               // Max retry attempts from config
}

// TaskExecution tracks execution history and duration
type TaskExecution struct {
	Attempts      []Attempt `json:"attempts,omitempty"` // History of execution attempts
	TotalDuration int       `json:"total_duration"`     // Total execution time in seconds
}

// Attempt represents a single execution attempt
type Attempt struct {
	AttemptNumber int       `json:"attempt_number"`            // 1-indexed attempt number
	StartedAt     time.Time `json:"started_at"`                // When attempt started
	CompletedAt   time.Time `json:"completed_at"`              // When attempt finished
	Duration      int       `json:"duration"`                  // Duration in seconds
	Model         string    `json:"model"`                     // AI model used
	Success       bool      `json:"success"`                   // Whether attempt succeeded
	Result        string    `json:"result"`                    // "passed", "failed", "error"
	LogPath       string    `json:"log_path"`                  // Path to agent execution log
	VerifyLogPath string    `json:"verify_log_path,omitempty"` // Path to verification log
	Error         string    `json:"error,omitempty"`           // Error message if failed
}

// TaskResults holds final task results when completed
type TaskResults struct {
	VerificationOutput string    `json:"verification_output,omitempty"` // Output from verification command
	CommitHash         string    `json:"commit_hash,omitempty"`         // Git commit hash if auto-committed
	CompletedAt        time.Time `json:"completed_at"`                  // Completion timestamp
}

// IsBlocked returns true if the task has unresolved blockedBy dependencies
func (t *Task) IsBlocked() bool {
	return len(t.BlockedBy) > 0
}

// CanStart returns true if the task is pending and not blocked
func (t *Task) CanStart() bool {
	return t.Status == "pending" && !t.IsBlocked()
}

// AddAttempt appends a new attempt to the task's execution history
func (t *Task) AddAttempt(attempt Attempt) {
	t.Execution.Attempts = append(t.Execution.Attempts, attempt)
	t.Execution.TotalDuration += attempt.Duration
	t.Metadata.UpdatedAt = time.Now()
}
