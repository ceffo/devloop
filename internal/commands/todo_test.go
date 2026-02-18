package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ceffo/devloop/internal/config"
)

func TestProcessCmd(t *testing.T) {
	// Create a temporary directory for test
	tmpDir, err := os.MkdirTemp("", "devloop-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a valid config
	cfg := &config.Config{
		Version: "1.0",
		Project: config.ProjectConfig{
			Name:       "test-project",
			Path:       tmpDir,
			TechStack:  "Go",
			MainBranch: "main",
		},
		Verification: config.VerificationConfig{
			Command:        "go test ./...",
			TimeoutSeconds: 300,
		},
		CLI: config.CLIConfig{
			Tool: "claude",
			Models: map[string]string{
				"simple":   "claude-haiku-4-5-20251001",
				"moderate": "claude-sonnet-4-5-20250929",
				"complex":  "claude-opus-4-6",
			},
		},
		Execution: config.ExecutionConfig{
			MaxAttempts:   2,
			HaltOnFailure: true,
			AutoCommit:    true,
		},
	}

	// Save config
	configPath := filepath.Join(tmpDir, "config.json")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create .devloop directory
	devloopDir := filepath.Join(tmpDir, ".devloop")
	if err := os.MkdirAll(devloopDir, 0755); err != nil {
		t.Fatalf("failed to create .devloop directory: %v", err)
	}

	// Test that ProcessCmd returns a valid command
	cmd := ProcessCmd()
	if cmd == nil {
		t.Error("ProcessCmd returned nil")
	}

	if cmd.Use != "process" {
		t.Errorf("expected Use to be 'process', got '%s'", cmd.Use)
	}

	// Check that todo and prd subcommands exist
	subCmds := cmd.Commands()
	if len(subCmds) == 0 {
		t.Error("expected ProcessCmd to have subcommands")
	}

	foundTodo := false
	foundPrd := false
	for _, subCmd := range subCmds {
		if subCmd.Use == "todo FILE" {
			foundTodo = true
		}
		if subCmd.Use == "prd FILE" {
			foundPrd = true
		}
	}

	if !foundTodo {
		t.Error("expected 'todo' subcommand to be present")
	}
	if !foundPrd {
		t.Error("expected 'prd' subcommand to be present")
	}
}

func TestProcessTodoCmd_MissingFile(t *testing.T) {
	// Create a temporary directory for test
	tmpDir, err := os.MkdirTemp("", "devloop-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a valid config
	cfg := &config.Config{
		Version: "1.0",
		Project: config.ProjectConfig{
			Name:       "test-project",
			Path:       tmpDir,
			TechStack:  "Go",
			MainBranch: "main",
		},
		Verification: config.VerificationConfig{
			Command:        "go test ./...",
			TimeoutSeconds: 300,
		},
		CLI: config.CLIConfig{
			Tool: "claude",
			Models: map[string]string{
				"simple":   "claude-haiku-4-5-20251001",
				"moderate": "claude-sonnet-4-5-20250929",
				"complex":  "claude-opus-4-6",
			},
		},
		Execution: config.ExecutionConfig{
			MaxAttempts:   2,
			HaltOnFailure: true,
			AutoCommit:    true,
		},
	}

	// Save config
	configPath := filepath.Join(tmpDir, "config.json")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create .devloop directory
	devloopDir := filepath.Join(tmpDir, ".devloop")
	if err := os.MkdirAll(devloopDir, 0755); err != nil {
		t.Fatalf("failed to create .devloop directory: %v", err)
	}

	// Create command
	cmd := processTodoCmd()
	cmd.Flags().String("config", configPath, "")
	cmd.SetArgs([]string{"/nonexistent/file.md"})

	// Execute command - should fail because file doesn't exist
	err = cmd.Execute()
	if err == nil {
		t.Error("expected error when processing non-existent file")
	}
}

func TestProcessTodoCmd_EmptyFile(t *testing.T) {
	// Create a temporary directory for test
	tmpDir, err := os.MkdirTemp("", "devloop-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a valid config
	cfg := &config.Config{
		Version: "1.0",
		Project: config.ProjectConfig{
			Name:       "test-project",
			Path:       tmpDir,
			TechStack:  "Go",
			MainBranch: "main",
		},
		Verification: config.VerificationConfig{
			Command:        "go test ./...",
			TimeoutSeconds: 300,
		},
		CLI: config.CLIConfig{
			Tool: "claude",
			Models: map[string]string{
				"simple":   "claude-haiku-4-5-20251001",
				"moderate": "claude-sonnet-4-5-20250929",
				"complex":  "claude-opus-4-6",
			},
		},
		Execution: config.ExecutionConfig{
			MaxAttempts:   2,
			HaltOnFailure: true,
			AutoCommit:    true,
		},
	}

	// Save config
	configPath := filepath.Join(tmpDir, "config.json")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create .devloop directory
	devloopDir := filepath.Join(tmpDir, ".devloop")
	if err := os.MkdirAll(devloopDir, 0755); err != nil {
		t.Fatalf("failed to create .devloop directory: %v", err)
	}

	// Create an empty TODO file
	todoPath := filepath.Join(tmpDir, "TODO.md")
	if err := os.WriteFile(todoPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create TODO file: %v", err)
	}

	// Create command
	cmd := processTodoCmd()
	cmd.Flags().String("config", configPath, "")
	cmd.SetArgs([]string{todoPath})

	// Execute command - should succeed but find no TODO items
	err = cmd.Execute()
	if err != nil {
		t.Errorf("unexpected error with empty TODO file: %v", err)
	}
}
