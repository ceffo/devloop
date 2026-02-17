package executor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ceffo/devloop/internal/config"
)

func TestLoadSession_NewSession(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a config pointing to the temp directory
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: tempDir,
		},
	}

	// Load session (should create new since file doesn't exist)
	session := LoadSession(cfg)

	// Verify session is created
	if session == nil {
		t.Fatal("Expected session to be created, got nil")
	}

	// Verify session has UUID
	if session.ID == "" {
		t.Error("Expected session to have a UUID, got empty string")
	}

	// Verify StartedAt is set
	if session.StartedAt.IsZero() {
		t.Error("Expected StartedAt to be set, got zero time")
	}

	// Verify LastCheckpoint is empty
	if session.LastCheckpoint != "" {
		t.Errorf("Expected LastCheckpoint to be empty, got %q", session.LastCheckpoint)
	}

	// Verify TasksCompleted is initialized
	if session.TasksCompleted == nil {
		t.Error("Expected TasksCompleted to be initialized, got nil")
	}

	// Verify TasksFailed is initialized
	if session.TasksFailed == nil {
		t.Error("Expected TasksFailed to be initialized, got nil")
	}
}

func TestLoadSession_ExistingSession(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a config pointing to the temp directory
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: tempDir,
		},
	}

	// Create a session manually
	existingSession := &Session{
		ID:             "test-uuid-123",
		StartedAt:      time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		LastCheckpoint: "1.2",
		TasksCompleted: []string{"1.1"},
		TasksFailed:    []string{},
	}

	// Save the session
	err := SaveSession(cfg, existingSession)
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Load the session
	loadedSession := LoadSession(cfg)

	// Verify loaded session matches
	if loadedSession.ID != existingSession.ID {
		t.Errorf("Expected ID %q, got %q", existingSession.ID, loadedSession.ID)
	}

	if !loadedSession.StartedAt.Equal(existingSession.StartedAt) {
		t.Errorf("Expected StartedAt %v, got %v", existingSession.StartedAt, loadedSession.StartedAt)
	}

	if loadedSession.LastCheckpoint != existingSession.LastCheckpoint {
		t.Errorf("Expected LastCheckpoint %q, got %q", existingSession.LastCheckpoint, loadedSession.LastCheckpoint)
	}

	if len(loadedSession.TasksCompleted) != len(existingSession.TasksCompleted) {
		t.Errorf("Expected %d completed tasks, got %d", len(existingSession.TasksCompleted), len(loadedSession.TasksCompleted))
	}

	if len(loadedSession.TasksFailed) != len(existingSession.TasksFailed) {
		t.Errorf("Expected %d failed tasks, got %d", len(existingSession.TasksFailed), len(loadedSession.TasksFailed))
	}
}

func TestLoadSession_CorruptedFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a config pointing to the temp directory
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: tempDir,
		},
	}

	// Create state directory
	stateDir := filepath.Join(tempDir, ".devloop", "state")
	err := os.MkdirAll(stateDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create state directory: %v", err)
	}

	// Write corrupted JSON to session file
	sessionPath := filepath.Join(stateDir, "session.json")
	err = os.WriteFile(sessionPath, []byte("invalid json {{{"), 0644)
	if err != nil {
		t.Fatalf("Failed to write corrupted session file: %v", err)
	}

	// Load session (should create new since file is corrupted)
	session := LoadSession(cfg)

	// Verify a new session is created
	if session == nil {
		t.Fatal("Expected session to be created, got nil")
	}

	if session.ID == "" {
		t.Error("Expected new session to have a UUID")
	}
}

func TestSaveSession(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a config pointing to the temp directory
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: tempDir,
		},
	}

	// Create a session
	session := &Session{
		ID:             "test-uuid-456",
		StartedAt:      time.Date(2024, 2, 15, 10, 30, 0, 0, time.UTC),
		LastCheckpoint: "2.3",
		TasksCompleted: []string{"1.1", "1.2", "2.1", "2.2"},
		TasksFailed:    []string{"1.3"},
	}

	// Save the session
	err := SaveSession(cfg, session)
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Verify file exists
	sessionPath := filepath.Join(tempDir, ".devloop", "state", "session.json")
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		t.Fatalf("Expected session file to exist at %s", sessionPath)
	}

	// Read and verify file contents
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		t.Fatalf("Failed to read session file: %v", err)
	}

	// Parse JSON
	var savedSession Session
	err = json.Unmarshal(data, &savedSession)
	if err != nil {
		t.Fatalf("Failed to parse session JSON: %v", err)
	}

	// Verify contents match
	if savedSession.ID != session.ID {
		t.Errorf("Expected ID %q, got %q", session.ID, savedSession.ID)
	}

	if savedSession.LastCheckpoint != session.LastCheckpoint {
		t.Errorf("Expected LastCheckpoint %q, got %q", session.LastCheckpoint, savedSession.LastCheckpoint)
	}

	if len(savedSession.TasksCompleted) != len(session.TasksCompleted) {
		t.Errorf("Expected %d completed tasks, got %d", len(session.TasksCompleted), len(savedSession.TasksCompleted))
	}
}

func TestSaveSession_CreatesDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a config pointing to the temp directory
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: tempDir,
		},
	}

	// Create a session
	session := &Session{
		ID:             "test-uuid-789",
		StartedAt:      time.Now(),
		LastCheckpoint: "",
		TasksCompleted: []string{},
		TasksFailed:    []string{},
	}

	// Save the session (should create state directory)
	err := SaveSession(cfg, session)
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Verify state directory exists
	stateDir := filepath.Join(tempDir, ".devloop", "state")
	if _, err := os.Stat(stateDir); os.IsNotExist(err) {
		t.Fatalf("Expected state directory to be created at %s", stateDir)
	}

	// Verify directory permissions
	info, err := os.Stat(stateDir)
	if err != nil {
		t.Fatalf("Failed to stat state directory: %v", err)
	}

	if !info.IsDir() {
		t.Error("Expected state path to be a directory")
	}

	// Check permissions (0755)
	perm := info.Mode().Perm()
	if perm != 0755 {
		t.Errorf("Expected directory permissions 0755, got %o", perm)
	}
}

func TestCheckpointSession(t *testing.T) {
	// Create a session
	session := &Session{
		ID:             "test-uuid",
		StartedAt:      time.Now(),
		LastCheckpoint: "",
		TasksCompleted: []string{},
		TasksFailed:    []string{},
	}

	// Initially, LastCheckpoint should be empty
	if session.LastCheckpoint != "" {
		t.Errorf("Expected initial LastCheckpoint to be empty, got %q", session.LastCheckpoint)
	}

	// Update checkpoint
	CheckpointSession(session, "1.1")
	if session.LastCheckpoint != "1.1" {
		t.Errorf("Expected LastCheckpoint to be %q, got %q", "1.1", session.LastCheckpoint)
	}

	// Update checkpoint again
	CheckpointSession(session, "2.5")
	if session.LastCheckpoint != "2.5" {
		t.Errorf("Expected LastCheckpoint to be %q, got %q", "2.5", session.LastCheckpoint)
	}
}

func TestRoundTripSession(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a config pointing to the temp directory
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: tempDir,
		},
	}

	// Create a new session
	session1 := LoadSession(cfg)
	session1.LastCheckpoint = "3.4"
	session1.TasksCompleted = []string{"1.1", "2.2", "3.3"}
	session1.TasksFailed = []string{"1.2"}

	// Save it
	err := SaveSession(cfg, session1)
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Load it again
	session2 := LoadSession(cfg)

	// Verify it matches
	if session2.ID != session1.ID {
		t.Errorf("Expected ID %q, got %q", session1.ID, session2.ID)
	}

	if session2.LastCheckpoint != session1.LastCheckpoint {
		t.Errorf("Expected LastCheckpoint %q, got %q", session1.LastCheckpoint, session2.LastCheckpoint)
	}

	if len(session2.TasksCompleted) != len(session1.TasksCompleted) {
		t.Errorf("Expected %d completed tasks, got %d", len(session1.TasksCompleted), len(session2.TasksCompleted))
	}

	if len(session2.TasksFailed) != len(session1.TasksFailed) {
		t.Errorf("Expected %d failed tasks, got %d", len(session1.TasksFailed), len(session2.TasksFailed))
	}

	// Verify task arrays match
	for i, task := range session1.TasksCompleted {
		if session2.TasksCompleted[i] != task {
			t.Errorf("Expected completed task %d to be %q, got %q", i, task, session2.TasksCompleted[i])
		}
	}

	for i, task := range session1.TasksFailed {
		if session2.TasksFailed[i] != task {
			t.Errorf("Expected failed task %d to be %q, got %q", i, task, session2.TasksFailed[i])
		}
	}
}

func TestLoadSessionForResume_NoSession(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		Project: config.ProjectConfig{Path: tempDir},
	}

	session, exists := LoadSessionForResume(cfg)
	if exists {
		t.Error("Expected no session to exist, but got one")
	}
	if session != nil {
		t.Error("Expected nil session, got non-nil")
	}
}

func TestLoadSessionForResume_ExistingSession(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		Project: config.ProjectConfig{Path: tempDir},
	}

	// Save a session first
	existing := &Session{
		ID:             "resume-test-uuid",
		StartedAt:      time.Date(2024, 3, 1, 9, 0, 0, 0, time.UTC),
		LastCheckpoint: "DEV-5",
		TasksCompleted: []string{"DEV-1", "DEV-2"},
		TasksFailed:    []string{},
		AgentName:      "claude",
		ModelMap:       map[string]string{"simple": "haiku", "moderate": "sonnet"},
	}
	if err := SaveSession(cfg, existing); err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	session, exists := LoadSessionForResume(cfg)
	if !exists {
		t.Error("Expected session to exist, but got none")
	}
	if session == nil {
		t.Fatal("Expected non-nil session")
	}
	if session.ID != existing.ID {
		t.Errorf("Expected ID %q, got %q", existing.ID, session.ID)
	}
	if session.LastCheckpoint != existing.LastCheckpoint {
		t.Errorf("Expected LastCheckpoint %q, got %q", existing.LastCheckpoint, session.LastCheckpoint)
	}
	if session.AgentName != existing.AgentName {
		t.Errorf("Expected AgentName %q, got %q", existing.AgentName, session.AgentName)
	}
	if len(session.ModelMap) != len(existing.ModelMap) {
		t.Errorf("Expected %d model map entries, got %d", len(existing.ModelMap), len(session.ModelMap))
	}
}

func TestSessionMetadata(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		Project: config.ProjectConfig{Path: tempDir},
	}

	session := &Session{
		ID:        "meta-test",
		StartedAt: time.Now(),
		AgentName: "claude",
		ModelMap: map[string]string{
			"simple":   "claude-haiku",
			"moderate": "claude-sonnet",
			"complex":  "claude-opus",
		},
		TaskSnapshot: []TaskSnapshot{
			{ID: "DEV-1", Title: "First task", Status: "pending", Complexity: "simple", Wave: 1},
			{ID: "DEV-2", Title: "Second task", Status: "pending", Complexity: "moderate", Wave: 1},
		},
		TasksCompleted: []string{},
		TasksFailed:    []string{},
	}

	if err := SaveSession(cfg, session); err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	loaded := LoadSession(cfg)
	if loaded.AgentName != "claude" {
		t.Errorf("Expected AgentName %q, got %q", "claude", loaded.AgentName)
	}
	if len(loaded.ModelMap) != 3 {
		t.Errorf("Expected 3 model map entries, got %d", len(loaded.ModelMap))
	}
	if loaded.ModelMap["simple"] != "claude-haiku" {
		t.Errorf("Expected simple model %q, got %q", "claude-haiku", loaded.ModelMap["simple"])
	}
	if len(loaded.TaskSnapshot) != 2 {
		t.Errorf("Expected 2 task snapshots, got %d", len(loaded.TaskSnapshot))
	}
	if loaded.TaskSnapshot[0].ID != "DEV-1" {
		t.Errorf("Expected first snapshot ID %q, got %q", "DEV-1", loaded.TaskSnapshot[0].ID)
	}
}

func TestFilePermissions(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a config pointing to the temp directory
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: tempDir,
		},
	}

	// Create and save a session
	session := LoadSession(cfg)
	err := SaveSession(cfg, session)
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Check file permissions
	sessionPath := filepath.Join(tempDir, ".devloop", "state", "session.json")
	info, err := os.Stat(sessionPath)
	if err != nil {
		t.Fatalf("Failed to stat session file: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0644 {
		t.Errorf("Expected file permissions 0644, got %o", perm)
	}
}
