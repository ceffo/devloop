package storage

import (
	"context"
	"fmt"

	"github.com/ceffo/devloop/internal/config"
)

// Syncer is implemented by storage backends that support explicit synchronization.
// Sync is backend-specific and not part of the TaskStore interface.
type Syncer interface {
	Sync() error
}

// Compactor is implemented by storage backends that support memory compaction.
// Compact is backend-specific and not part of the TaskStore interface.
type Compactor interface {
	Compact() error
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
		bs, err := NewBeadsStore(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize beads backend: %w", err)
		}
		return &beadsTaskStoreAdapter{store: bs}, nil
	default:
		return nil, fmt.Errorf("unknown storage backend: %q", backend)
	}
}

// beadsTaskStoreAdapter wraps *BeadsStore and implements TaskStore using context.Background().
// It also implements Syncer and Compactor by delegating to the underlying BeadsStore.
type beadsTaskStoreAdapter struct {
	store *BeadsStore
}

func (a *beadsTaskStoreAdapter) LoadTasks() ([]*Task, error) {
	return a.store.LoadTasks(context.Background())
}

func (a *beadsTaskStoreAdapter) SaveTask(task *Task) error {
	return a.store.SaveTask(context.Background(), task)
}

func (a *beadsTaskStoreAdapter) UpdateTask(task *Task) error {
	return a.store.UpdateTask(context.Background(), task)
}

func (a *beadsTaskStoreAdapter) GetTask(id string) (*Task, error) {
	return a.store.GetTask(context.Background(), id)
}

func (a *beadsTaskStoreAdapter) QueryTasks(filter Filter) ([]*Task, error) {
	return a.store.QueryTasks(context.Background(), filter)
}

func (a *beadsTaskStoreAdapter) QueryReadyTasks() ([]*Task, error) {
	return a.store.QueryReadyTasks(context.Background())
}

func (a *beadsTaskStoreAdapter) Sync() error {
	return a.store.Sync()
}

func (a *beadsTaskStoreAdapter) Compact() error {
	return a.store.Compact()
}
