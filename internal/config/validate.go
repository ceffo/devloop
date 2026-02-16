package config

import (
	"errors"
	"fmt"
	"os"
)

// knownModels is the list of valid model identifiers for both Claude and Copilot
var knownModels = map[string]bool{
	// Claude CLI models
	"claude-haiku-4-5-20251001":  true,
	"claude-sonnet-4-5-20250929": true,
	"claude-opus-4-6":            true,
	// GitHub Copilot CLI models
	"claude-sonnet-4.5":    true,
	"claude-haiku-4.5":     true,
	"claude-opus-4.6":      true,
	"claude-opus-4.6-fast": true,
	"claude-opus-4.5":      true,
	"claude-sonnet-4":      true,
	"gemini-3-pro-preview": true,
	"gpt-5.3-codex":        true,
	"gpt-5.2-codex":        true,
	"gpt-5.2":              true,
	"gpt-5.1-codex-max":    true,
	"gpt-5.1-codex":        true,
	"gpt-5.1":              true,
	"gpt-5":                true,
	"gpt-5.1-codex-mini":   true,
	"gpt-5-mini":           true,
	"gpt-4.1":              true,
}

// Validate checks if the configuration is valid
// Returns an error if any required fields are missing or invalid
func (c *Config) Validate() error {
	// Check project path exists
	if c.Project.Path == "" {
		return errors.New("project.path is required")
	}

	if _, err := os.Stat(c.Project.Path); os.IsNotExist(err) {
		return fmt.Errorf("project.path does not exist: %s", c.Project.Path)
	}

	// Check verification command is not empty
	if c.Verification.Command == "" {
		return errors.New("verification.command is required")
	}

	// Check timeout > 0
	if c.Verification.TimeoutSeconds <= 0 {
		return fmt.Errorf("verification.timeout_seconds must be greater than 0, got: %d", c.Verification.TimeoutSeconds)
	}

	// Check model names are valid
	for complexity, model := range c.CLI.Models {
		if !knownModels[model] {
			return fmt.Errorf("cli.models.%s has invalid model name: %s (valid models: claude-haiku-4-5-20251001, claude-sonnet-4-5-20250929, claude-opus-4-6)", complexity, model)
		}
	}

	// Check max_attempts > 0
	if c.Execution.MaxAttempts <= 0 {
		return fmt.Errorf("execution.max_attempts must be greater than 0, got: %d", c.Execution.MaxAttempts)
	}

	return nil
}
