package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ceffo/devloop/internal/config"
	"github.com/google/uuid"
)

// Session represents a dev loop execution session for crash recovery
type Session struct {
	ID             string    `json:"id"`
	StartedAt      time.Time `json:"started_at"`
	LastCheckpoint string    `json:"last_checkpoint"`
	TasksCompleted []string  `json:"tasks_completed"`
	TasksFailed    []string  `json:"tasks_failed"`
}

// LoadSession loads an existing session from disk or creates a new one
// Returns a new session if no session file exists
func LoadSession(cfg *config.Config) *Session {
	sessionPath := getSessionPath(cfg)

	// Try to load existing session
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		// If file doesn't exist or can't be read, create new session
		return newSession()
	}

	// Parse existing session
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		// If parsing fails, create new session
		return newSession()
	}

	return &session
}

// SaveSession writes the session state to disk
func SaveSession(cfg *config.Config, session *Session) error {
	sessionPath := getSessionPath(cfg)

	// Create state directory if it doesn't exist
	stateDir := filepath.Dir(sessionPath)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Marshal session to JSON
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session to JSON: %w", err)
	}

	// Write to file with 0644 permissions
	if err := os.WriteFile(sessionPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// CheckpointSession updates the session's last checkpoint with the given task ID
func CheckpointSession(session *Session, taskID string) {
	session.LastCheckpoint = taskID
}

// newSession creates a new session with a UUID and current timestamp
func newSession() *Session {
	return &Session{
		ID:             uuid.New().String(),
		StartedAt:      time.Now(),
		LastCheckpoint: "",
		TasksCompleted: []string{},
		TasksFailed:    []string{},
	}
}

// getSessionPath returns the path to the session.json file based on config
func getSessionPath(cfg *config.Config) string {
	// Construct path: <project_path>/.devloop/state/session.json
	return filepath.Join(cfg.Project.Path, ".devloop", "state", "session.json")
}
