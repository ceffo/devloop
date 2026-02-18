package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Config represents the complete devloop configuration
type Config struct {
	Version      string             `json:"version"`
	Project      ProjectConfig      `json:"project"`
	Verification VerificationConfig `json:"verification"`
	CLI          CLIConfig          `json:"cli"`
	Execution    ExecutionConfig    `json:"execution"`
	Archival     ArchivalConfig     `json:"archival,omitempty"`
	Prompts      PromptsConfig      `json:"prompts,omitempty"`
}

// ProjectConfig holds project-specific metadata
type ProjectConfig struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	TechStack     string `json:"tech_stack"`
	MainBranch    string `json:"main_branch"`
	TaskIDPrefix  string `json:"task_id_prefix,omitempty"`  // e.g., "DEV", "WORK"
	TaskIDFormat  string `json:"task_id_format,omitempty"`  // "hierarchical" or "jira"
}

// VerificationConfig defines how to verify task completion
type VerificationConfig struct {
	Command        string `json:"command"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// AgentConfig defines a single agentic CLI with its tool and models
type AgentConfig struct {
	Tool   string            `json:"tool"`
	Models map[string]string `json:"models"`
}

// GetModel returns the model name for the given complexity level
// Returns an error if the complexity level is not found
func (a *AgentConfig) GetModel(complexity string) (string, error) {
	model, exists := a.Models[complexity]
	if !exists {
		available := make([]string, 0, len(a.Models))
		for k := range a.Models {
			available = append(available, k)
		}
		sort.Strings(available)
		return "", fmt.Errorf("complexity %q not found in agent models (available: %v)", complexity, available)
	}
	return model, nil
}

// CLIConfig specifies which AI CLI tool and models to use
// Supports multiple agents with optional default agent selection
type CLIConfig struct {
	DefaultAgent string                  `json:"default_agent,omitempty"`
	Agents       map[string]*AgentConfig `json:"agents,omitempty"`

	// Deprecated fields for backwards compatibility
	// These are converted to the agents map on load
	Tool   string            `json:"tool,omitempty"`
	Models map[string]string `json:"models,omitempty"`
}

// ExecutionConfig controls task execution behavior
type ExecutionConfig struct {
	MaxAttempts   int    `json:"max_attempts"`
	HaltOnFailure bool   `json:"halt_on_failure"`
	AutoCommit    bool   `json:"auto_commit"`
	CommitFormat  string `json:"commit_format,omitempty"`
}

// ArchivalConfig controls archival behavior
type ArchivalConfig struct {
	AutoArchive bool `json:"auto_archive,omitempty"`
}

// PromptsConfig allows custom prompt templates
type PromptsConfig struct {
	CustomInstructions string `json:"custom_instructions,omitempty"`
}

// GetAgent returns the AgentConfig for the given agent name
// If the agent doesn't exist, returns an error with helpful message listing available agents
func (c *CLIConfig) GetAgent(name string) (*AgentConfig, error) {
	// If using legacy format (agents map is empty), create a default agent from old fields
	if len(c.Agents) == 0 && c.Tool != "" {
		return &AgentConfig{
			Tool:   c.Tool,
			Models: c.Models,
		}, nil
	}

	// If requesting specific agent, return it
	if name != "" {
		if agent, exists := c.Agents[name]; exists {
			return agent, nil
		}
		// Agent not found - list available ones
		available := make([]string, 0, len(c.Agents))
		for agentName := range c.Agents {
			available = append(available, agentName)
		}
		sort.Strings(available)
		return nil, fmt.Errorf("agent %q not found (available: %v)", name, available)
	}

	// If no name requested, return default agent
	if c.DefaultAgent != "" {
		if agent, exists := c.Agents[c.DefaultAgent]; exists {
			return agent, nil
		}
		return nil, fmt.Errorf("default agent %q not found", c.DefaultAgent)
	}

	// No default agent specified, use first alphabetically
	if len(c.Agents) > 0 {
		available := make([]string, 0, len(c.Agents))
		for agentName := range c.Agents {
			available = append(available, agentName)
		}
		sort.Strings(available)
		return c.Agents[available[0]], nil
	}

	return nil, fmt.Errorf("no agents configured")
}

// GetDefaultAgentName returns the effective default agent name
// If default_agent is set, returns it. Otherwise returns first agent alphabetically.
func (c *CLIConfig) GetDefaultAgentName() string {
	// If default is explicitly set, use it
	if c.DefaultAgent != "" {
		return c.DefaultAgent
	}

	// Return first alphabetically
	if len(c.Agents) > 0 {
		available := make([]string, 0, len(c.Agents))
		for agentName := range c.Agents {
			available = append(available, agentName)
		}
		sort.Strings(available)
		return available[0]
	}

	return ""
}

// migrateOldFormat converts old config format (with top-level tool/models) to new format (with agents map)
// Returns true if migration was performed
func (c *CLIConfig) migrateOldFormat() bool {
	// If already in new format (agents map exists), no migration needed
	if len(c.Agents) > 0 {
		return false
	}

	// If old format fields are empty, nothing to migrate
	if c.Tool == "" {
		return false
	}

	// Migrate old format to new format
	c.Agents = map[string]*AgentConfig{
		"default": {
			Tool:   c.Tool,
			Models: c.Models,
		},
	}
	c.DefaultAgent = "default"

	// Keep old fields for backwards compatibility on save
	// They'll be preserved as-is in JSON

	return true
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

	// Migrate old format to new format if needed
	cfg.CLI.migrateOldFormat()

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
			Name:         "",
			Path:         "",
			TechStack:    "",
			MainBranch:   "main",
			TaskIDPrefix: "",     // Will be derived from project name
			TaskIDFormat: "jira", // Default to JIRA format for new projects
		},
		Verification: VerificationConfig{
			Command:        "",
			TimeoutSeconds: 300,
		},
		CLI: CLIConfig{
			DefaultAgent: "claude",
			Agents: map[string]*AgentConfig{
				"claude": {
					Tool: "claude",
					Models: map[string]string{
						"simple":   "claude-haiku-4-5-20251001",
						"moderate": "claude-sonnet-4-5-20250929",
						"complex":  "claude-opus-4-6",
					},
				},
			},
			// Keep legacy fields for backwards compatibility
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

	// Set task ID format defaults
	if cfg.Project.TaskIDFormat == "" {
		cfg.Project.TaskIDFormat = defaults.Project.TaskIDFormat
	}

	// Derive task ID prefix from project name if not set
	if cfg.Project.TaskIDPrefix == "" && cfg.Project.Name != "" {
		cfg.Project.TaskIDPrefix = DeriveTaskIDPrefix(cfg.Project.Name)
	}

	if cfg.Verification.TimeoutSeconds <= 0 {
		cfg.Verification.TimeoutSeconds = defaults.Verification.TimeoutSeconds
	}

	// Apply CLI defaults
	if len(cfg.CLI.Agents) == 0 {
		cfg.CLI.Agents = defaults.CLI.Agents
	}
	if cfg.CLI.DefaultAgent == "" {
		cfg.CLI.DefaultAgent = defaults.CLI.DefaultAgent
	}

	// Also set legacy fields for backwards compatibility
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
}

// DeriveTaskIDPrefix generates a JIRA-style prefix from the project name
// Examples: "devloop" -> "DEV", "my-project" -> "MYP", "x" -> "X"
func DeriveTaskIDPrefix(name string) string {
	// Extract first letters of each word (separated by dash or space)
	var prefix string
	capNext := true

	for _, ch := range name {
		if ch == '-' || ch == '_' || ch == ' ' {
			capNext = true
			continue
		}

		if capNext {
			prefix += strings.ToUpper(string(ch))
			capNext = false
		}
	}

	// If we got multiple letters from word extraction, use it
	if len(prefix) > 1 {
		return prefix
	}

	// Otherwise, use first 3 letters of the name
	if len(name) >= 3 {
		return strings.ToUpper(name[:3])
	}

	// If name is shorter than 3 letters, use the whole name
	return strings.ToUpper(name)
}
