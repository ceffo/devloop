package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/ceffo/devloop/internal/config"
)

// Storage handles JSONL read/write operations for tasks
type Storage struct {
	cfg      *config.Config
	tasksDir string
}

// NewStorage creates a new Storage instance
func NewStorage(cfg *config.Config) *Storage {
	// Construct the tasks directory path: <project_path>/.devloop
	tasksDir := filepath.Join(cfg.Project.Path, ".devloop")

	return &Storage{
		cfg:      cfg,
		tasksDir: tasksDir,
	}
}

// getTasksFilePath returns the full path to the tasks.jsonl file
func (s *Storage) getTasksFilePath() string {
	return filepath.Join(s.tasksDir, "tasks.jsonl")
}

// ConfigPath returns the full path to the config.json file
func (s *Storage) ConfigPath() string {
	return filepath.Join(s.tasksDir, "config.json")
}

// LoadTasks reads all tasks from the JSONL file
func (s *Storage) LoadTasks() ([]*Task, error) {
	path := s.getTasksFilePath()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Return empty slice if file doesn't exist
		return []*Task{}, nil
	}

	// Open file for reading
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open tasks file: %w", err)
	}
	defer file.Close()

	var tasks []*Task
	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Read line by line
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		// Parse JSON line
		var task Task
		if err := json.Unmarshal([]byte(line), &task); err != nil {
			return nil, fmt.Errorf("failed to parse task at line %d: %w", lineNum, err)
		}

		tasks = append(tasks, &task)
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading tasks file: %w", err)
	}

	return tasks, nil
}

// SaveTask appends a new task to the JSONL file
func (s *Storage) SaveTask(task *Task) error {
	path := s.getTasksFilePath()

	// Create parent directory if missing
	if err := os.MkdirAll(s.tasksDir, 0755); err != nil {
		return fmt.Errorf("failed to create tasks directory: %w", err)
	}

	// Open file for appending (create if doesn't exist)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open tasks file: %w", err)
	}
	defer file.Close()

	// Encode task to JSON
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(task); err != nil {
		return fmt.Errorf("failed to encode task: %w", err)
	}

	return nil
}

// UpdateTask updates an existing task by rewriting the entire file
func (s *Storage) UpdateTask(task *Task) error {
	// Load all tasks
	tasks, err := s.LoadTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Find and replace the task
	found := false
	for i, t := range tasks {
		if t.ID == task.ID {
			tasks[i] = task
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("task with ID %s not found", task.ID)
	}

	// Rewrite the file
	path := s.getTasksFilePath()

	// Create parent directory if missing
	if err := os.MkdirAll(s.tasksDir, 0755); err != nil {
		return fmt.Errorf("failed to create tasks directory: %w", err)
	}

	// Write to temporary file first
	tmpPath := path + ".tmp"
	tmpFile, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	encoder := json.NewEncoder(tmpFile)
	for _, t := range tasks {
		if err := encoder.Encode(t); err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("failed to encode task: %w", err)
		}
	}

	tmpFile.Close()

	// Atomically replace the original file
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to replace tasks file: %w", err)
	}

	return nil
}

// DeleteAllTasks removes all tasks from storage
// This is used during migration to rewrite all task IDs
func (s *Storage) DeleteAllTasks() error {
	path := s.getTasksFilePath()

	// Create parent directory if missing
	if err := os.MkdirAll(s.tasksDir, 0755); err != nil {
		return fmt.Errorf("failed to create tasks directory: %w", err)
	}

	// Create empty file (truncate if exists)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create empty tasks file: %w", err)
	}
	defer file.Close()

	return nil
}

// GetTask finds and returns a task by ID
func (s *Storage) GetTask(id string) (*Task, error) {
	// Load all tasks
	tasks, err := s.LoadTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	// Search for task by ID
	for _, task := range tasks {
		if task.ID == id {
			return task, nil
		}
	}

	return nil, fmt.Errorf("task with ID %s not found", id)
}

// QueryTasks filters tasks based on the provided filter criteria and returns them sorted by ID
func (s *Storage) QueryTasks(filter Filter) ([]*Task, error) {
	// Load all tasks
	tasks, err := s.LoadTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	// Filter tasks
	var filtered []*Task
	for _, task := range tasks {
		if task.matchesFilter(filter) {
			filtered = append(filtered, task)
		}
	}

	// Sort by ID (ascending)
	sort.Slice(filtered, func(i, j int) bool {
		return compareTaskIDs(filtered[i].ID, filtered[j].ID)
	})

	return filtered, nil
}
