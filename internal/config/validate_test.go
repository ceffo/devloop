package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidate_ValidConfig(t *testing.T) {
	// Create a temporary directory to use as project path
	tmpDir := t.TempDir()

	cfg := &Config{
		Version: "1.0",
		Project: ProjectConfig{
			Name:       "test-project",
			Path:       tmpDir,
			TechStack:  "Go",
			MainBranch: "main",
		},
		Verification: VerificationConfig{
			Command:        "go test",
			TimeoutSeconds: 300,
		},
		CLI: CLIConfig{
			Tool: "claude",
			Models: map[string]string{
				"simple":   "claude-haiku-4-5-20251001",
				"moderate": "claude-sonnet-4-5-20250929",
				"complex":  "claude-opus-4-6",
			},
		},
		Execution: ExecutionConfig{
			MaxAttempts:   2,
			HaltOnFailure: true,
			AutoCommit:    true,
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() failed for valid config: %v", err)
	}
}

func TestValidate_EmptyProjectPath(t *testing.T) {
	cfg := getDefaultConfig()
	cfg.Project.Path = ""
	cfg.Verification.Command = "go test"

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail when project.path is empty")
	}

	if !strings.Contains(err.Error(), "project.path is required") {
		t.Errorf("Expected error about project.path, got: %v", err)
	}
}

func TestValidate_NonExistentProjectPath(t *testing.T) {
	cfg := getDefaultConfig()
	cfg.Project.Path = "/path/that/does/not/exist/xyz123"
	cfg.Verification.Command = "go test"

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail when project.path does not exist")
	}

	if !strings.Contains(err.Error(), "project.path does not exist") {
		t.Errorf("Expected error about non-existent path, got: %v", err)
	}
}

func TestValidate_EmptyVerificationCommand(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := getDefaultConfig()
	cfg.Project.Path = tmpDir
	cfg.Verification.Command = ""

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail when verification.command is empty")
	}

	if !strings.Contains(err.Error(), "verification.command is required") {
		t.Errorf("Expected error about verification.command, got: %v", err)
	}
}

func TestValidate_ZeroTimeout(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := getDefaultConfig()
	cfg.Project.Path = tmpDir
	cfg.Verification.Command = "go test"
	cfg.Verification.TimeoutSeconds = 0

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail when timeout is 0")
	}

	if !strings.Contains(err.Error(), "timeout_seconds must be greater than 0") {
		t.Errorf("Expected error about timeout, got: %v", err)
	}
}

func TestValidate_NegativeTimeout(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := getDefaultConfig()
	cfg.Project.Path = tmpDir
	cfg.Verification.Command = "go test"
	cfg.Verification.TimeoutSeconds = -100

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail when timeout is negative")
	}

	if !strings.Contains(err.Error(), "timeout_seconds must be greater than 0") {
		t.Errorf("Expected error about timeout, got: %v", err)
	}
}

func TestValidate_InvalidModelName(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := getDefaultConfig()
	cfg.Project.Path = tmpDir
	cfg.Verification.Command = "go test"
	cfg.CLI.Models = map[string]string{
		"simple":   "invalid-model-name",
		"moderate": "claude-sonnet-4-5-20250929",
		"complex":  "claude-opus-4-6",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail when model name is invalid")
	}

	if !strings.Contains(err.Error(), "invalid model name") {
		t.Errorf("Expected error about invalid model, got: %v", err)
	}
}

func TestValidate_ZeroMaxAttempts(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := getDefaultConfig()
	cfg.Project.Path = tmpDir
	cfg.Verification.Command = "go test"
	cfg.Execution.MaxAttempts = 0

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail when max_attempts is 0")
	}

	if !strings.Contains(err.Error(), "max_attempts must be greater than 0") {
		t.Errorf("Expected error about max_attempts, got: %v", err)
	}
}

func TestValidate_NegativeMaxAttempts(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := getDefaultConfig()
	cfg.Project.Path = tmpDir
	cfg.Verification.Command = "go test"
	cfg.Execution.MaxAttempts = -5

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail when max_attempts is negative")
	}

	if !strings.Contains(err.Error(), "max_attempts must be greater than 0") {
		t.Errorf("Expected error about max_attempts, got: %v", err)
	}
}

func TestValidate_AllModelsInvalid(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := getDefaultConfig()
	cfg.Project.Path = tmpDir
	cfg.Verification.Command = "go test"
	cfg.CLI.Models = map[string]string{
		"simple":   "gpt-4",
		"moderate": "gpt-3.5-turbo",
		"complex":  "gemini-pro",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail when all models are invalid")
	}

	if !strings.Contains(err.Error(), "invalid model name") {
		t.Errorf("Expected error about invalid model, got: %v", err)
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	// Test that validation catches the first error and returns it
	cfg := &Config{
		Project: ProjectConfig{
			Path: "", // Invalid: empty path
		},
		Verification: VerificationConfig{
			Command:        "", // Invalid: empty command
			TimeoutSeconds: 0,  // Invalid: zero timeout
		},
		Execution: ExecutionConfig{
			MaxAttempts: 0, // Invalid: zero attempts
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail when multiple fields are invalid")
	}

	// Should catch the first error (empty project path)
	if !strings.Contains(err.Error(), "project.path") {
		t.Errorf("Expected error about project.path first, got: %v", err)
	}
}

func TestValidate_RealProjectDirectory(t *testing.T) {
	// Test with the actual current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	cfg := &Config{
		Project: ProjectConfig{
			Name: "devloop",
			Path: filepath.Dir(filepath.Dir(cwd)), // Go up to project root
		},
		Verification: VerificationConfig{
			Command:        "go test ./...",
			TimeoutSeconds: 600,
		},
		CLI: CLIConfig{
			Tool: "claude",
			Models: map[string]string{
				"simple":   "claude-haiku-4-5-20251001",
				"moderate": "claude-sonnet-4-5-20250929",
				"complex":  "claude-opus-4-6",
			},
		},
		Execution: ExecutionConfig{
			MaxAttempts: 3,
		},
	}

	err = cfg.Validate()
	if err != nil {
		t.Errorf("Validate() failed for config with real project directory: %v", err)
	}
}
