package config

import (
	"errors"
	"fmt"
	"os"
)

// modelsFetcher is the function used to discover valid models for a binary.
// It is a package-level variable so tests can substitute a mock implementation.
var modelsFetcher = FetchModels

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

	// Validate agents configuration
	if len(c.CLI.Agents) > 0 {
		// New format: agents map
		// Validate each agent
		for agentName, agent := range c.CLI.Agents {
			// Check tool is valid
			if agent.Tool != "claude" && agent.Tool != "copilot" {
				return fmt.Errorf("cli.agents.%s.tool must be 'claude' or 'copilot', got: %s", agentName, agent.Tool)
			}

			// Check models is not empty
			if len(agent.Models) == 0 {
				return fmt.Errorf("cli.agents.%s.models cannot be empty", agentName)
			}

			// Check model names are valid
			validModels, err := modelsFetcher(agent.Tool)
			if err != nil {
				return fmt.Errorf("cli.agents.%s: failed to query models for tool %q: %w", agentName, agent.Tool, err)
			}
			if validModels != nil {
				for complexity, model := range agent.Models {
					if !validModels[model] {
						return fmt.Errorf("cli.agents.%s.models.%s has invalid model name: %s", agentName, complexity, model)
					}
				}
			}
			// validModels == nil means the binary was not found or does not list
			// choices — model names are accepted as-is.
		}
	} else {
		return errors.New("cli.agents must be configured")
	}

	// Check max_attempts > 0
	if c.Execution.MaxAttempts <= 0 {
		return fmt.Errorf("execution.max_attempts must be greater than 0, got: %d", c.Execution.MaxAttempts)
	}

	return nil
}
