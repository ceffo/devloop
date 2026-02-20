package storage

import (
	"fmt"

	"github.com/ceffo/devloop/internal/config"
)

// Syncer is implemented by storage backends that support explicit synchronization.
// Sync is backend-specific and not part of the TaskStore interface.
type Syncer interface {
	Sync() error
}

// TaskStore defines the contract that all storage backends must implement
// for persisting and querying tasks.
type TaskStore interface {
	// LoadTasks reads all tasks from storage
	LoadTasks() ([]*Task, error)

	// SaveTask appends a new task to storage
	SaveTask(task *Task) error

	// UpdateTask updates an existing task in storage
	UpdateTask(task *Task) error

	// GetTask finds and returns a task by ID
	GetTask(id string) (*Task, error)

	// QueryTasks filters tasks based on the provided filter criteria
	// Returns results sorted by task ID
	QueryTasks(filter Filter) ([]*Task, error)

	// QueryReadyTasks returns all tasks that are ready to execute
	// (pending status and not blocked by any dependencies)
	QueryReadyTasks() ([]*Task, error)
}

// NewTaskStore creates a TaskStore instance based on the configured backend
// Returns *JSONLStore for 'jsonl' or empty backend
// Returns an error for unknown backends
func NewTaskStore(cfg *config.Config) (TaskStore, error) {
	backend := cfg.Storage.Backend

	// Default to jsonl if backend is empty
	if backend == "" {
		return NewJSONLStore(cfg), nil
	}

	switch backend {
	case "jsonl":
		return NewJSONLStore(cfg), nil
	case "beads":
		return nil, fmt.Errorf("beads backend is not yet implemented")
	default:
		return nil, fmt.Errorf("unknown storage backend: %q", backend)
	}
}
