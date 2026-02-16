package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the complete devloop configuration
type Config struct {
	Version      string             `json:"version"`
	Project      ProjectConfig      `json:"project"`
	Verification VerificationConfig `json:"verification"`
	CLI          CLIConfig          `json:"cli"`
	Execution    ExecutionConfig    `json:"execution"`
	Files        FilesConfig        `json:"files"`
	Archival     ArchivalConfig     `json:"archival,omitempty"`
	Prompts      PromptsConfig      `json:"prompts,omitempty"`
}

// ProjectConfig holds project-specific metadata
type ProjectConfig struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	TechStack  string `json:"tech_stack"`
	MainBranch string `json:"main_branch"`
}

// VerificationConfig defines how to verify task completion
type VerificationConfig struct {
	Command        string `json:"command"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// CLIConfig specifies which AI CLI tool and models to use
type CLIConfig struct {
	Tool   string            `json:"tool"`
	Models map[string]string `json:"models"`
}

// ExecutionConfig controls task execution behavior
type ExecutionConfig struct {
	MaxAttempts   int    `json:"max_attempts"`
	HaltOnFailure bool   `json:"halt_on_failure"`
	AutoCommit    bool   `json:"auto_commit"`
	CommitFormat  string `json:"commit_format,omitempty"`
}

// FilesConfig maps to project artifact locations
type FilesConfig struct {
	PRD   string `json:"prd,omitempty"`
	Tasks string `json:"tasks,omitempty"`
	Todo  string `json:"todo,omitempty"`
}

// ArchivalConfig controls archival behavior
type ArchivalConfig struct {
	AutoArchive bool `json:"auto_archive,omitempty"`
}

// PromptsConfig allows custom prompt templates
type PromptsConfig struct {
	CustomInstructions string `json:"custom_instructions,omitempty"`
}

// LoadConfig loads configuration from a JSON file
// Returns a Config with sensible defaults if the file doesn't exist
func LoadConfig(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Return config with defaults
		return getDefaultConfig(), nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Apply defaults for missing fields
	applyDefaults(&cfg)

	return &cfg, nil
}

// SaveConfig writes configuration to a JSON file
func SaveConfig(path string, cfg *Config) error {
	// Create parent directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to pretty JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config to JSON: %w", err)
	}

	// Write to file with 0644 permissions
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getDefaultConfig returns a Config with sensible defaults
func getDefaultConfig() *Config {
	return &Config{
		Version: "1.0",
		Project: ProjectConfig{
			Name:       "",
			Path:       "",
			TechStack:  "",
			MainBranch: "main",
		},
		Verification: VerificationConfig{
			Command:        "",
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
			CommitFormat:  "task {task_id}: {title}",
		},
		Files: FilesConfig{
			PRD:   "docs/PRD.md",
			Tasks: "docs/TASKS.md",
			Todo:  ".todo/TODO.md",
		},
		Archival: ArchivalConfig{
			AutoArchive: false,
		},
		Prompts: PromptsConfig{
			CustomInstructions: "",
		},
	}
}

// applyDefaults fills in missing fields with default values
func applyDefaults(cfg *Config) {
	defaults := getDefaultConfig()

	if cfg.Version == "" {
		cfg.Version = defaults.Version
	}

	if cfg.Project.MainBranch == "" {
		cfg.Project.MainBranch = defaults.Project.MainBranch
	}

	if cfg.Verification.TimeoutSeconds <= 0 {
		cfg.Verification.TimeoutSeconds = defaults.Verification.TimeoutSeconds
	}

	if cfg.CLI.Tool == "" {
		cfg.CLI.Tool = defaults.CLI.Tool
	}

	if cfg.CLI.Models == nil {
		cfg.CLI.Models = defaults.CLI.Models
	}

	if cfg.Execution.MaxAttempts <= 0 {
		cfg.Execution.MaxAttempts = defaults.Execution.MaxAttempts
	}

	if cfg.Execution.CommitFormat == "" {
		cfg.Execution.CommitFormat = defaults.Execution.CommitFormat
	}

	if cfg.Files.PRD == "" {
		cfg.Files.PRD = defaults.Files.PRD
	}

	if cfg.Files.Tasks == "" {
		cfg.Files.Tasks = defaults.Files.Tasks
	}

	if cfg.Files.Todo == "" {
		cfg.Files.Todo = defaults.Files.Todo
	}
}
