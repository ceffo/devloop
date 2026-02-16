package storage

import (
	"encoding/json"
	"testing"
	"time"
)

// TestTaskMarshalJSON tests that Task can be marshaled to JSON
func TestTaskMarshalJSON(t *testing.T) {
	now := time.Now()
	task := &Task{
		ID:          "1.1",
		Title:       "Test task",
		Wave:        1,
		Status:      "pending",
		Complexity:  "simple",
		Description: "A test task",
		AcceptanceCriteria: []string{
			"Criterion 1",
			"Criterion 2",
		},
		BlockedBy: []string{},
		Tags:      []string{"test", "sample"},
		Metadata: TaskMetadata{
			CreatedAt:      now,
			UpdatedAt:      now,
			SourceType:     "todo",
			SourceTodoItem: "item-1",
			MaxAttempts:    2,
		},
		Execution: TaskExecution{
			Attempts:      []Attempt{},
			TotalDuration: 0,
		},
		Results: nil,
	}

	// Marshal to JSON
	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal task to JSON: %v", err)
	}

	// Verify it's valid JSON
	if len(data) == 0 {
		t.Fatal("Marshaled JSON is empty")
	}
}

// TestTaskUnmarshalJSON tests that Task can be unmarshaled from JSON
func TestTaskUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"id": "1.1",
		"title": "Test task",
		"wave": 1,
		"status": "pending",
		"complexity": "simple",
		"model": "claude-haiku-4-5-20251001",
		"description": "A test task",
		"acceptance_criteria": ["Criterion 1", "Criterion 2"],
		"blocked_by": [],
		"tags": ["test", "sample"],
		"metadata": {
			"created_at": "2026-02-15T10:00:00Z",
			"updated_at": "2026-02-15T10:00:00Z",
			"source_type": "todo",
			"source_todo_item": "item-1",
			"max_attempts": 2
		},
		"execution": {
			"attempts": [],
			"total_duration": 0
		}
	}`

	var task Task
	if err := json.Unmarshal([]byte(jsonData), &task); err != nil {
		t.Fatalf("Failed to unmarshal task from JSON: %v", err)
	}

	// Verify fields
	if task.ID != "1.1" {
		t.Errorf("Expected ID '1.1', got '%s'", task.ID)
	}
	if task.Title != "Test task" {
		t.Errorf("Expected Title 'Test task', got '%s'", task.Title)
	}
	if task.Wave != 1 {
		t.Errorf("Expected Wave 1, got %d", task.Wave)
	}
	if task.Status != "pending" {
		t.Errorf("Expected Status 'pending', got '%s'", task.Status)
	}
	if len(task.AcceptanceCriteria) != 2 {
		t.Errorf("Expected 2 acceptance criteria, got %d", len(task.AcceptanceCriteria))
	}
}

// TestTaskRoundTrip tests that Task can be marshaled and unmarshaled without data loss
func TestTaskRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second) // Truncate to second precision for comparison

	original := &Task{
		ID:          "2.3",
		Title:       "Complex task",
		Wave:        2,
		Status:      "in_progress",
		Complexity:  "complex",
		Description: "A complex test task",
		AcceptanceCriteria: []string{
			"Must pass all tests",
			"Must compile without errors",
		},
		BlockedBy: []string{"2.1", "2.2"},
		Tags:      []string{"important"},
		Metadata: TaskMetadata{
			CreatedAt:      now,
			UpdatedAt:      now,
			SourceType:     "manual",
			SourceTodoItem: "",
			MaxAttempts:    3,
		},
		Execution: TaskExecution{
			Attempts: []Attempt{
				{
					AttemptNumber: 1,
					StartedAt:     now,
					CompletedAt:   now.Add(5 * time.Minute),
					Duration:      300,
					Success:       false,
					Result:        "failed",
					LogPath:       "/path/to/log",
					Error:         "Verification failed",
				},
			},
			TotalDuration: 300,
		},
		Results: nil,
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal task: %v", err)
	}

	// Unmarshal
	var restored Task
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal task: %v", err)
	}

	// Compare key fields
	if restored.ID != original.ID {
		t.Errorf("ID mismatch: expected '%s', got '%s'", original.ID, restored.ID)
	}
	if restored.Status != original.Status {
		t.Errorf("Status mismatch: expected '%s', got '%s'", original.Status, restored.Status)
	}
	if len(restored.BlockedBy) != len(original.BlockedBy) {
		t.Errorf("BlockedBy length mismatch: expected %d, got %d", len(original.BlockedBy), len(restored.BlockedBy))
	}
	if len(restored.Execution.Attempts) != len(original.Execution.Attempts) {
		t.Errorf("Attempts length mismatch: expected %d, got %d", len(original.Execution.Attempts), len(restored.Execution.Attempts))
	}
	if restored.Execution.TotalDuration != original.Execution.TotalDuration {
		t.Errorf("TotalDuration mismatch: expected %d, got %d", original.Execution.TotalDuration, restored.Execution.TotalDuration)
	}
}

// TestIsBlocked tests the IsBlocked helper method
func TestIsBlocked(t *testing.T) {
	tests := []struct {
		name      string
		blockedBy []string
		want      bool
	}{
		{
			name:      "No blockers",
			blockedBy: []string{},
			want:      false,
		},
		{
			name:      "Nil blockers",
			blockedBy: nil,
			want:      false,
		},
		{
			name:      "One blocker",
			blockedBy: []string{"1.1"},
			want:      true,
		},
		{
			name:      "Multiple blockers",
			blockedBy: []string{"1.1", "1.2", "1.3"},
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{
				BlockedBy: tt.blockedBy,
			}
			if got := task.IsBlocked(); got != tt.want {
				t.Errorf("IsBlocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCanStart tests the CanStart helper method
func TestCanStart(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		blockedBy []string
		want      bool
	}{
		{
			name:      "Pending and not blocked",
			status:    "pending",
			blockedBy: []string{},
			want:      true,
		},
		{
			name:      "Pending but blocked",
			status:    "pending",
			blockedBy: []string{"1.1"},
			want:      false,
		},
		{
			name:      "In progress",
			status:    "in_progress",
			blockedBy: []string{},
			want:      false,
		},
		{
			name:      "Completed",
			status:    "completed",
			blockedBy: []string{},
			want:      false,
		},
		{
			name:      "Failed",
			status:    "failed",
			blockedBy: []string{},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{
				Status:    tt.status,
				BlockedBy: tt.blockedBy,
			}
			if got := task.CanStart(); got != tt.want {
				t.Errorf("CanStart() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAddAttempt tests the AddAttempt helper method
func TestAddAttempt(t *testing.T) {
	now := time.Now()
	task := &Task{
		ID:     "1.1",
		Status: "in_progress",
		Metadata: TaskMetadata{
			CreatedAt: now,
			UpdatedAt: now,
		},
		Execution: TaskExecution{
			Attempts:      []Attempt{},
			TotalDuration: 0,
		},
	}

	// Add first attempt
	attempt1 := Attempt{
		AttemptNumber: 1,
		StartedAt:     now,
		CompletedAt:   now.Add(2 * time.Minute),
		Duration:      120,
		Success:       false,
		Result:        "failed",
		LogPath:       "/path/to/log1",
		Error:         "Test failed",
	}
	task.AddAttempt(attempt1)

	// Verify first attempt was added
	if len(task.Execution.Attempts) != 1 {
		t.Errorf("Expected 1 attempt, got %d", len(task.Execution.Attempts))
	}
	if task.Execution.TotalDuration != 120 {
		t.Errorf("Expected TotalDuration 120, got %d", task.Execution.TotalDuration)
	}

	// Add second attempt
	attempt2 := Attempt{
		AttemptNumber: 2,
		StartedAt:     now.Add(3 * time.Minute),
		CompletedAt:   now.Add(5 * time.Minute),
		Duration:      120,
		Success:       true,
		Result:        "passed",
		LogPath:       "/path/to/log2",
	}
	task.AddAttempt(attempt2)

	// Verify second attempt was added
	if len(task.Execution.Attempts) != 2 {
		t.Errorf("Expected 2 attempts, got %d", len(task.Execution.Attempts))
	}
	if task.Execution.TotalDuration != 240 {
		t.Errorf("Expected TotalDuration 240, got %d", task.Execution.TotalDuration)
	}

	// Verify UpdatedAt was updated
	if task.Metadata.UpdatedAt.Before(now) {
		t.Error("Expected UpdatedAt to be updated after AddAttempt")
	}
}

// TestTaskWithResults tests Task with completed results
func TestTaskWithResults(t *testing.T) {
	now := time.Now()
	task := &Task{
		ID:     "1.1",
		Status: "completed",
		Results: &TaskResults{
			VerificationOutput: "All tests passed",
			CommitHash:         "abc123def456",
			CompletedAt:        now,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal task with results: %v", err)
	}

	// Unmarshal
	var restored Task
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal task with results: %v", err)
	}

	// Verify results
	if restored.Results == nil {
		t.Fatal("Expected Results to be non-nil")
	}
	if restored.Results.CommitHash != "abc123def456" {
		t.Errorf("Expected CommitHash 'abc123def456', got '%s'", restored.Results.CommitHash)
	}
}
