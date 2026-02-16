package executor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yourusername/devloop/internal/config"
	"github.com/yourusername/devloop/internal/storage"
)

func TestGenerateCommitMessage(t *testing.T) {
	tests := []struct {
		name     string
		template string
		task     *storage.Task
		expected string
	}{
		{
			name:     "simple template with task_id and title",
			template: "task {task_id}: {title}",
			task: &storage.Task{
				ID:    "1.1",
				Title: "implement config",
			},
			expected: "task 1.1: implement config",
		},
		{
			name:     "template with all variables",
			template: "[{task_id}] {title} - {complexity}\n\n{description}",
			task: &storage.Task{
				ID:          "2.3",
				Title:       "add tests",
				Description: "Add unit tests for storage",
				Complexity:  "moderate",
			},
			expected: "[2.3] add tests - moderate\n\nAdd unit tests for storage",
		},
		{
			name:     "template without variables",
			template: "fixed bug",
			task: &storage.Task{
				ID:    "1.1",
				Title: "whatever",
			},
			expected: "fixed bug",
		},
		{
			name:     "template with repeated variables",
			template: "{task_id} - {task_id}: {title}",
			task: &storage.Task{
				ID:    "3.5",
				Title: "refactor code",
			},
			expected: "3.5 - 3.5: refactor code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateCommitMessage(tt.template, tt.task)
			if result != tt.expected {
				t.Errorf("generateCommitMessage() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractCommitHash(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		wantErr bool
		minLen  int // minimum expected hash length
	}{
		{
			name:    "standard commit output",
			output:  "[main 1234567] task 1.1: implement config",
			wantErr: false,
			minLen:  7,
		},
		{
			name:    "detached HEAD output",
			output:  "[detached HEAD abcdef1] fix bug",
			wantErr: false,
			minLen:  7,
		},
		{
			name:    "commit with longer hash",
			output:  "[main 1234567890abc] some commit",
			wantErr: false,
			minLen:  7,
		},
		{
			name:    "branch with dashes",
			output:  "[feature-branch 9876543] new feature",
			wantErr: false,
			minLen:  7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := extractCommitHash(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractCommitHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(hash) < tt.minLen {
					t.Errorf("extractCommitHash() hash length = %d, want >= %d", len(hash), tt.minLen)
				}
				// Verify hash is hexadecimal
				if !isHexString(hash) {
					t.Errorf("extractCommitHash() hash = %q is not hexadecimal", hash)
				}
			}
		})
	}
}

func TestAutoCommit(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}

	// Create temporary git repository for testing
	tempDir := t.TempDir()

	// Initialize git repo
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tempDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user (required for commits)
	configNameCmd := exec.Command("git", "config", "user.name", "Test User")
	configNameCmd.Dir = tempDir
	if err := configNameCmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user.name: %v", err)
	}

	configEmailCmd := exec.Command("git", "config", "user.email", "test@example.com")
	configEmailCmd.Dir = tempDir
	if err := configEmailCmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user.email: %v", err)
	}

	// Create initial commit (git requires at least one commit)
	testFile := filepath.Join(tempDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	addCmd := exec.Command("git", "add", "README.md")
	addCmd.Dir = tempDir
	if err := addCmd.Run(); err != nil {
		t.Fatalf("Failed to add initial file: %v", err)
	}

	initialCommitCmd := exec.Command("git", "commit", "-m", "initial commit")
	initialCommitCmd.Dir = tempDir
	if err := initialCommitCmd.Run(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	tests := []struct {
		name           string
		config         *config.Config
		task           *storage.Task
		createChanges  bool
		wantErr        bool
		errContains    string
	}{
		{
			name: "successful commit with changes",
			config: &config.Config{
				Project: config.ProjectConfig{
					Path: tempDir,
				},
				Execution: config.ExecutionConfig{
					AutoCommit:   true,
					CommitFormat: "task {task_id}: {title}",
				},
			},
			task: &storage.Task{
				ID:    "1.1",
				Title: "test commit",
			},
			createChanges: true,
			wantErr:       false,
		},
		{
			name: "auto-commit disabled",
			config: &config.Config{
				Project: config.ProjectConfig{
					Path: tempDir,
				},
				Execution: config.ExecutionConfig{
					AutoCommit:   false,
					CommitFormat: "task {task_id}: {title}",
				},
			},
			task: &storage.Task{
				ID:    "1.2",
				Title: "should not commit",
			},
			createChanges: true,
			wantErr:       true,
			errContains:   "auto-commit is not enabled",
		},
		{
			name: "commit with custom format",
			config: &config.Config{
				Project: config.ProjectConfig{
					Path: tempDir,
				},
				Execution: config.ExecutionConfig{
					AutoCommit:   true,
					CommitFormat: "[{task_id}] {title} ({complexity})",
				},
			},
			task: &storage.Task{
				ID:         "2.1",
				Title:      "custom format test",
				Complexity: "simple",
			},
			createChanges: true,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create changes if needed
			if tt.createChanges {
				changeFile := filepath.Join(tempDir, "change-"+tt.task.ID+".txt")
				if err := os.WriteFile(changeFile, []byte("some changes"), 0644); err != nil {
					t.Fatalf("Failed to create change file: %v", err)
				}
			}

			// Execute AutoCommit
			hash, err := AutoCommit(tt.config, tt.task)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("AutoCommit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("AutoCommit() error = %v, should contain %q", err, tt.errContains)
				}
				return
			}

			// Verify commit hash
			if hash == "" {
				t.Error("AutoCommit() returned empty hash")
			}
			if len(hash) < 7 {
				t.Errorf("AutoCommit() hash length = %d, want >= 7", len(hash))
			}
			if !isHexString(hash) {
				t.Errorf("AutoCommit() hash = %q is not hexadecimal", hash)
			}

			// Verify commit exists in git log
			logCmd := exec.Command("git", "log", "--oneline", "-1")
			logCmd.Dir = tempDir
			output, err := logCmd.Output()
			if err != nil {
				t.Fatalf("Failed to get git log: %v", err)
			}

			logStr := string(output)
			if !strings.Contains(logStr, hash) {
				t.Errorf("Git log does not contain commit hash %q: %s", hash, logStr)
			}
		})
	}
}

func TestAutoCommit_NoChanges(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}

	// Create temporary git repository
	tempDir := t.TempDir()

	// Initialize git repo with initial commit
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tempDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	configNameCmd := exec.Command("git", "config", "user.name", "Test User")
	configNameCmd.Dir = tempDir
	if err := configNameCmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user.name: %v", err)
	}

	configEmailCmd := exec.Command("git", "config", "user.email", "test@example.com")
	configEmailCmd.Dir = tempDir
	if err := configEmailCmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user.email: %v", err)
	}

	testFile := filepath.Join(tempDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	addCmd := exec.Command("git", "add", "README.md")
	addCmd.Dir = tempDir
	if err := addCmd.Run(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	commitCmd := exec.Command("git", "commit", "-m", "initial")
	commitCmd.Dir = tempDir
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: tempDir,
		},
		Execution: config.ExecutionConfig{
			AutoCommit:   true,
			CommitFormat: "task {task_id}: {title}",
		},
	}

	task := &storage.Task{
		ID:    "1.1",
		Title: "no changes",
	}

	// Try to commit with no changes - should fail
	_, err := AutoCommit(cfg, task)
	if err == nil {
		t.Error("AutoCommit() should fail when there are no changes")
	}
	if !strings.Contains(err.Error(), "git commit failed") {
		t.Errorf("AutoCommit() error should mention git commit failure, got: %v", err)
	}
}

func TestAutoCommit_InvalidGitRepo(t *testing.T) {
	// Create temp directory that is NOT a git repo
	tempDir := t.TempDir()

	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: tempDir,
		},
		Execution: config.ExecutionConfig{
			AutoCommit:   true,
			CommitFormat: "task {task_id}: {title}",
		},
	}

	task := &storage.Task{
		ID:    "1.1",
		Title: "test",
	}

	// Create a file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Should fail because directory is not a git repo
	_, err := AutoCommit(cfg, task)
	if err == nil {
		t.Error("AutoCommit() should fail when directory is not a git repo")
	}
}

func TestGenerateCommitMessage_EmptyFields(t *testing.T) {
	template := "task {task_id}: {title} - {description}"
	task := &storage.Task{
		ID:          "1.1",
		Title:       "test",
		Description: "", // empty description
	}

	result := generateCommitMessage(template, task)
	expected := "task 1.1: test - "

	if result != expected {
		t.Errorf("generateCommitMessage() = %q, want %q", result, expected)
	}
}

// Helper function to check if a string is hexadecimal
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// Example usage of AutoCommit
func ExampleAutoCommit() {
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: "/path/to/project",
		},
		Execution: config.ExecutionConfig{
			AutoCommit:   true,
			CommitFormat: "task {task_id}: {title}",
		},
	}

	task := &storage.Task{
		ID:          "1.1",
		Title:       "implement feature",
		Description: "Add new feature to the system",
		Complexity:  "moderate",
		Metadata: storage.TaskMetadata{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	hash, err := AutoCommit(cfg, task)
	if err != nil {
		// Handle error
		return
	}

	// hash contains the commit hash (e.g., "1234567")
	_ = hash
}
