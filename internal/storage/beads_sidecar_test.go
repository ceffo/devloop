package storage

import (
	"testing"
	"time"

	"github.com/ceffo/devloop/internal/config"
)

// newSidecarTestStore returns a BeadsStore rooted at a temporary directory.
func newSidecarTestStore(t *testing.T) *BeadsStore {
	t.Helper()
	return &BeadsStore{
		cfg: &config.Config{
			Project: config.ProjectConfig{
				Path: t.TempDir(),
			},
		},
		bdPath: "/usr/local/bin/bd",
	}
}

func TestWriteReadSidecar_RoundTrip(t *testing.T) {
	store := newSidecarTestStore(t)

	want := &TaskSidecar{
		DevloopID:          "DEV-5",
		Complexity:         "moderate",
		AcceptanceCriteria: []string{"criterion 1", "criterion 2"},
		Tags:               []string{"backend", "api"},
		MaxAttempts:        3,
		Execution: TaskExecution{
			TotalDuration: 120,
			Attempts: []Attempt{
				{
					AttemptNumber: 1,
					StartedAt:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					CompletedAt:   time.Date(2024, 1, 1, 0, 2, 0, 0, time.UTC),
					Duration:      120,
					Model:         "claude-sonnet",
					Success:       true,
					Result:        "passed",
				},
			},
		},
		Results: &TaskResults{
			VerificationOutput: "all tests passed",
			CommitHash:         "abc123",
			CompletedAt:        time.Date(2024, 1, 1, 0, 2, 0, 0, time.UTC),
		},
	}

	beadsID := "bd-x7f3"

	if err := store.writeSidecar(beadsID, want); err != nil {
		t.Fatalf("writeSidecar failed: %v", err)
	}

	got, err := store.readSidecar(beadsID)
	if err != nil {
		t.Fatalf("readSidecar failed: %v", err)
	}
	if got == nil {
		t.Fatal("readSidecar returned nil for existing sidecar")
	}

	if got.DevloopID != want.DevloopID {
		t.Errorf("DevloopID: expected %q, got %q", want.DevloopID, got.DevloopID)
	}
	if got.Complexity != want.Complexity {
		t.Errorf("Complexity: expected %q, got %q", want.Complexity, got.Complexity)
	}
	if len(got.AcceptanceCriteria) != len(want.AcceptanceCriteria) {
		t.Errorf("AcceptanceCriteria: expected %d items, got %d", len(want.AcceptanceCriteria), len(got.AcceptanceCriteria))
	} else {
		for i, c := range want.AcceptanceCriteria {
			if got.AcceptanceCriteria[i] != c {
				t.Errorf("AcceptanceCriteria[%d]: expected %q, got %q", i, c, got.AcceptanceCriteria[i])
			}
		}
	}
	if len(got.Tags) != len(want.Tags) {
		t.Errorf("Tags: expected %d items, got %d", len(want.Tags), len(got.Tags))
	}
	if got.MaxAttempts != want.MaxAttempts {
		t.Errorf("MaxAttempts: expected %d, got %d", want.MaxAttempts, got.MaxAttempts)
	}
	if got.Execution.TotalDuration != want.Execution.TotalDuration {
		t.Errorf("TotalDuration: expected %d, got %d", want.Execution.TotalDuration, got.Execution.TotalDuration)
	}
	if len(got.Execution.Attempts) != len(want.Execution.Attempts) {
		t.Errorf("Attempts: expected %d, got %d", len(want.Execution.Attempts), len(got.Execution.Attempts))
	} else {
		a := got.Execution.Attempts[0]
		w := want.Execution.Attempts[0]
		if a.AttemptNumber != w.AttemptNumber {
			t.Errorf("AttemptNumber: expected %d, got %d", w.AttemptNumber, a.AttemptNumber)
		}
		if a.Model != w.Model {
			t.Errorf("Model: expected %q, got %q", w.Model, a.Model)
		}
		if a.Success != w.Success {
			t.Errorf("Success: expected %v, got %v", w.Success, a.Success)
		}
	}
	if got.Results == nil {
		t.Fatal("Results should not be nil")
	}
	if got.Results.CommitHash != want.Results.CommitHash {
		t.Errorf("CommitHash: expected %q, got %q", want.Results.CommitHash, got.Results.CommitHash)
	}
	if got.Results.VerificationOutput != want.Results.VerificationOutput {
		t.Errorf("VerificationOutput: expected %q, got %q", want.Results.VerificationOutput, got.Results.VerificationOutput)
	}
}

func TestReadSidecar_NotFound(t *testing.T) {
	store := newSidecarTestStore(t)

	got, err := store.readSidecar("bd-notfound")
	if err != nil {
		t.Fatalf("readSidecar should not error for missing file: %v", err)
	}
	if got != nil {
		t.Error("readSidecar should return nil for missing sidecar")
	}
}

func TestWriteSidecar_CreatesDirectory(t *testing.T) {
	store := newSidecarTestStore(t)

	sidecar := &TaskSidecar{DevloopID: "DEV-1", Complexity: "simple"}
	if err := store.writeSidecar("bd-abc1", sidecar); err != nil {
		t.Fatalf("writeSidecar failed: %v", err)
	}

	// Reading back should succeed, confirming the directory was created
	got, err := store.readSidecar("bd-abc1")
	if err != nil {
		t.Fatalf("readSidecar failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil sidecar after write")
	}
}

func TestMergeSidecar_WithSidecar(t *testing.T) {
	bd := beadsTaskData{
		ID:          "bd-x7f3",
		Title:       "My Task",
		Description: "Do something",
		Status:      "pending",
	}
	sidecar := &TaskSidecar{
		DevloopID:          "DEV-5",
		Complexity:         "moderate",
		AcceptanceCriteria: []string{"criterion 1"},
		Tags:               []string{"backend"},
		MaxAttempts:        2,
	}

	task := mergeSidecar(bd, sidecar)

	if task.ID != "bd-x7f3" {
		t.Errorf("ID: expected %q, got %q", "bd-x7f3", task.ID)
	}
	if task.Title != "My Task" {
		t.Errorf("Title: expected %q, got %q", "My Task", task.Title)
	}
	if task.Description != "Do something" {
		t.Errorf("Description: expected %q, got %q", "Do something", task.Description)
	}
	if task.Status != "pending" {
		t.Errorf("Status: expected %q, got %q", "pending", task.Status)
	}
	if task.Complexity != "moderate" {
		t.Errorf("Complexity: expected %q, got %q", "moderate", task.Complexity)
	}
	if len(task.AcceptanceCriteria) != 1 || task.AcceptanceCriteria[0] != "criterion 1" {
		t.Errorf("AcceptanceCriteria: expected [criterion 1], got %v", task.AcceptanceCriteria)
	}
	if len(task.Tags) != 1 || task.Tags[0] != "backend" {
		t.Errorf("Tags: expected [backend], got %v", task.Tags)
	}
	if task.Metadata.MaxAttempts != 2 {
		t.Errorf("MaxAttempts: expected 2, got %d", task.Metadata.MaxAttempts)
	}
}

func TestMergeSidecar_NilSidecar(t *testing.T) {
	bd := beadsTaskData{
		ID:     "bd-x7f3",
		Title:  "My Task",
		Status: "pending",
	}

	task := mergeSidecar(bd, nil)

	if task.ID != "bd-x7f3" {
		t.Errorf("ID: expected %q, got %q", "bd-x7f3", task.ID)
	}
	if task.Title != "My Task" {
		t.Errorf("Title: expected %q, got %q", "My Task", task.Title)
	}
	if task.Complexity != "" {
		t.Errorf("Complexity should be empty for nil sidecar, got %q", task.Complexity)
	}
	if len(task.AcceptanceCriteria) != 0 {
		t.Errorf("AcceptanceCriteria should be nil/empty for nil sidecar, got %v", task.AcceptanceCriteria)
	}
	if task.Metadata.MaxAttempts != 0 {
		t.Errorf("MaxAttempts should be 0 for nil sidecar, got %d", task.Metadata.MaxAttempts)
	}
}
