package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// TaskSidecar holds devloop-internal data stored alongside a Beads task.
// It is stored at .devloop/tasks/<beads-id>.json and is separate from
// the Beads task itself. Dependencies are not stored here.
type TaskSidecar struct {
	DevloopID          string        `json:"devloop_id"`
	Complexity         string        `json:"complexity"`
	AcceptanceCriteria []string      `json:"acceptance_criteria,omitempty"`
	Tags               []string      `json:"tags,omitempty"`
	MaxAttempts        int           `json:"max_attempts"`
	Execution          TaskExecution `json:"execution"`
	Results            *TaskResults  `json:"results,omitempty"`
}

// beadsTaskData holds task fields sourced from the Beads system.
type beadsTaskData struct {
	ID          string
	Title       string
	Description string
	Status      string
}

// sidecarPath returns the path to the sidecar file for the given Beads ID.
func (s *BeadsStore) sidecarPath(beadsID string) string {
	return filepath.Join(s.cfg.Project.Path, ".devloop", "tasks", beadsID+".json")
}

// writeSidecar creates .devloop/tasks/ if needed and writes the sidecar as JSON.
func (s *BeadsStore) writeSidecar(beadsID string, sidecar *TaskSidecar) error {
	path := s.sidecarPath(beadsID)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create sidecar directory: %w", err)
	}

	data, err := json.MarshalIndent(sidecar, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sidecar: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write sidecar: %w", err)
	}

	return nil
}

// readSidecar reads the sidecar file for the given Beads ID.
// Returns nil if the file does not exist.
func (s *BeadsStore) readSidecar(beadsID string) (*TaskSidecar, error) {
	path := s.sidecarPath(beadsID)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read sidecar: %w", err)
	}

	var sidecar TaskSidecar
	if err := json.Unmarshal(data, &sidecar); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sidecar: %w", err)
	}

	return &sidecar, nil
}

// mergeSidecar combines Beads task data with sidecar data into a *Task.
// The Beads data provides ID, Title, Description, and Status;
// the sidecar provides devloop-internal fields.
func mergeSidecar(bd beadsTaskData, sidecar *TaskSidecar) *Task {
	task := &Task{
		ID:          bd.ID,
		Title:       bd.Title,
		Description: bd.Description,
		Status:      bd.Status,
	}

	if sidecar != nil {
		task.Complexity = sidecar.Complexity
		task.AcceptanceCriteria = sidecar.AcceptanceCriteria
		task.Tags = sidecar.Tags
		task.Metadata.MaxAttempts = sidecar.MaxAttempts
		task.Execution = sidecar.Execution
		task.Results = sidecar.Results
	}

	return task
}
