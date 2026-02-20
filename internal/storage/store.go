package storage

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
