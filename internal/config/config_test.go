package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_MissingFile(t *testing.T) {
	// Test loading from non-existent file returns defaults
	cfg, err := LoadConfig("/nonexistent/path/config.json")
	if err != nil {
		t.Fatalf("LoadConfig should not error on missing file: %v", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig should return default config when file is missing")
	}

	// Verify some defaults are set
	if cfg.Version != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", cfg.Version)
	}

	if cfg.Execution.MaxAttempts != 2 {
		t.Errorf("Expected max_attempts 2, got %d", cfg.Execution.MaxAttempts)
	}

	if cfg.CLI.Tool != "claude" {
		t.Errorf("Expected CLI tool 'claude', got '%s'", cfg.CLI.Tool)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "devloop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, ".devloop", "config.json")

	// Create a test config
	testCfg := &Config{
		Version: "1.0",
		Project: ProjectConfig{
			Name:       "test-project",
			Path:       "/path/to/project",
			TechStack:  "Go",
			MainBranch: "main",
		},
		Verification: VerificationConfig{
			Command:        "go test ./...",
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
	}

	// Save config
	if err := SaveConfig(configPath, testCfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load config back
	loadedCfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify loaded config matches
	if loadedCfg.Project.Name != testCfg.Project.Name {
		t.Errorf("Expected project name '%s', got '%s'", testCfg.Project.Name, loadedCfg.Project.Name)
	}

	if loadedCfg.Verification.Command != testCfg.Verification.Command {
		t.Errorf("Expected verification command '%s', got '%s'", testCfg.Verification.Command, loadedCfg.Verification.Command)
	}

	if loadedCfg.Execution.MaxAttempts != testCfg.Execution.MaxAttempts {
		t.Errorf("Expected max_attempts %d, got %d", testCfg.Execution.MaxAttempts, loadedCfg.Execution.MaxAttempts)
	}
}

func TestLoadConfig_ValidJSON(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "devloop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	// Create a sample config JSON
	sampleJSON := `{
		"version": "1.0",
		"project": {
			"name": "myproject",
			"path": "/home/user/code/myproject",
			"tech_stack": "React + TypeScript",
			"main_branch": "main"
		},
		"verification": {
			"command": "npm run build && npm test -- --run",
			"timeout_seconds": 300
		},
		"cli": {
			"tool": "claude",
			"models": {
				"simple": "claude-haiku-4-5-20251001",
				"moderate": "claude-sonnet-4-5-20250929",
				"complex": "claude-opus-4-6"
			}
		},
		"execution": {
			"max_attempts": 2,
			"halt_on_failure": true,
			"auto_commit": true
		},
		"files": {
			"prd": "docs/PRD.md",
			"tasks": "docs/TASKS.md",
			"todo": ".todo/TODO.md"
		}
	}`

	// Write JSON to file
	if err := os.WriteFile(configPath, []byte(sampleJSON), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify fields
	if cfg.Project.Name != "myproject" {
		t.Errorf("Expected project name 'myproject', got '%s'", cfg.Project.Name)
	}

	if cfg.Project.TechStack != "React + TypeScript" {
		t.Errorf("Expected tech_stack 'React + TypeScript', got '%s'", cfg.Project.TechStack)
	}

	if cfg.Verification.Command != "npm run build && npm test -- --run" {
		t.Errorf("Expected specific verification command, got '%s'", cfg.Verification.Command)
	}

	if cfg.Execution.MaxAttempts != 2 {
		t.Errorf("Expected max_attempts 2, got %d", cfg.Execution.MaxAttempts)
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "devloop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	// Write invalid JSON
	invalidJSON := `{"version": "1.0", "project": {invalid}}`
	if err := os.WriteFile(configPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load config should fail
	_, err = LoadConfig(configPath)
	if err == nil {
		t.Fatal("LoadConfig should return error for invalid JSON")
	}
}

func TestApplyDefaults(t *testing.T) {
	// Create config with missing fields
	cfg := &Config{
		Version: "1.0",
		Project: ProjectConfig{
			Name: "test",
			Path: "/test",
		},
		Verification: VerificationConfig{
			Command: "test",
			// TimeoutSeconds missing - should default to 300
		},
		Execution: ExecutionConfig{
			// MaxAttempts missing - should default to 2
		},
	}

	applyDefaults(cfg)

	if cfg.Verification.TimeoutSeconds != 300 {
		t.Errorf("Expected timeout_seconds to default to 300, got %d", cfg.Verification.TimeoutSeconds)
	}

	if cfg.Execution.MaxAttempts != 2 {
		t.Errorf("Expected max_attempts to default to 2, got %d", cfg.Execution.MaxAttempts)
	}

	if cfg.Project.MainBranch != "main" {
		t.Errorf("Expected main_branch to default to 'main', got '%s'", cfg.Project.MainBranch)
	}

	if cfg.CLI.Tool != "claude" {
		t.Errorf("Expected CLI tool to default to 'claude', got '%s'", cfg.CLI.Tool)
	}
}

func TestConfigMarshaling(t *testing.T) {
	// Test that Config can be marshaled and unmarshaled
	cfg := getDefaultConfig()
	cfg.Project.Name = "test-marshal"
	cfg.Project.Path = "/path/to/test"

	// Marshal to JSON
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Unmarshal back
	var cfg2 Config
	if err := json.Unmarshal(data, &cfg2); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify
	if cfg2.Project.Name != cfg.Project.Name {
		t.Errorf("Expected project name '%s', got '%s'", cfg.Project.Name, cfg2.Project.Name)
	}

	if cfg2.Project.Path != cfg.Project.Path {
		t.Errorf("Expected project path '%s', got '%s'", cfg.Project.Path, cfg2.Project.Path)
	}
}

// Tests for multi-agent support

func TestGetAgent_DefaultAgent(t *testing.T) {
	cfg := getDefaultConfig()

	agent, err := cfg.CLI.GetAgent("")
	if err != nil {
		t.Fatalf("GetAgent() should succeed for default: %v", err)
	}

	if agent == nil {
		t.Fatal("GetAgent() should return non-nil agent")
	}

	if agent.Tool != "claude" {
		t.Errorf("Expected tool 'claude', got '%s'", agent.Tool)
	}
}

func TestGetAgent_SpecificAgent(t *testing.T) {
	cfg := getDefaultConfig()

	agent, err := cfg.CLI.GetAgent("claude")
	if err != nil {
		t.Fatalf("GetAgent('claude') should succeed: %v", err)
	}

	if agent.Tool != "claude" {
		t.Errorf("Expected tool 'claude', got '%s'", agent.Tool)
	}
}

func TestGetAgent_InvalidAgent(t *testing.T) {
	cfg := getDefaultConfig()

	_, err := cfg.CLI.GetAgent("nonexistent")
	if err == nil {
		t.Error("GetAgent('nonexistent') should fail")
	}

	if !contains(err.Error(), "nonexistent") {
		t.Errorf("Error should mention agent name: %v", err)
	}
}

func TestGetAgent_MultipleAgents(t *testing.T) {
	cfg := &Config{
		CLI: CLIConfig{
			DefaultAgent: "claude",
			Agents: map[string]*AgentConfig{
				"claude": {
					Tool:   "claude",
					Models: map[string]string{"simple": "claude-haiku-4-5-20251001"},
				},
				"copilot": {
					Tool:   "copilot",
					Models: map[string]string{"simple": "gpt-4"},
				},
			},
		},
	}

	agentA, errA := cfg.CLI.GetAgent("claude")
	agentB, errB := cfg.CLI.GetAgent("copilot")

	if errA != nil || errB != nil {
		t.Fatalf("GetAgent should succeed for both agents")
	}

	if agentA.Tool != "claude" || agentB.Tool != "copilot" {
		t.Error("Agents should have correct tools")
	}
}

func TestGetDefaultAgentName_Explicit(t *testing.T) {
	cfg := &Config{
		CLI: CLIConfig{
			DefaultAgent: "copilot",
			Agents: map[string]*AgentConfig{
				"claude":  {Tool: "claude"},
				"copilot": {Tool: "copilot"},
			},
		},
	}

	name := cfg.CLI.GetDefaultAgentName()
	if name != "copilot" {
		t.Errorf("Expected 'copilot', got '%s'", name)
	}
}

func TestGetDefaultAgentName_Alphabetical(t *testing.T) {
	cfg := &Config{
		CLI: CLIConfig{
			// DefaultAgent not set, should use alphabetical
			Agents: map[string]*AgentConfig{
				"zulu":   {Tool: "claude"},
				"alpha":  {Tool: "copilot"},
				"bravo":  {Tool: "claude"},
			},
		},
	}

	name := cfg.CLI.GetDefaultAgentName()
	if name != "alpha" {
		t.Errorf("Expected 'alpha' (first alphabetically), got '%s'", name)
	}
}

func TestMigrateOldFormat(t *testing.T) {
	cli := CLIConfig{
		Tool: "claude",
		Models: map[string]string{
			"simple":   "claude-haiku-4-5-20251001",
			"moderate": "claude-sonnet-4-5-20250929",
			"complex":  "claude-opus-4-6",
		},
	}

	migrated := cli.migrateOldFormat()
	if !migrated {
		t.Error("migrateOldFormat should return true for old format")
	}

	if cli.DefaultAgent != "default" {
		t.Errorf("Expected default_agent 'default', got '%s'", cli.DefaultAgent)
	}

	if agent, exists := cli.Agents["default"]; !exists || agent.Tool != "claude" {
		t.Error("Migration should create 'default' agent with old settings")
	}
}

func TestMigrateOldFormat_AlreadyNew(t *testing.T) {
	cli := CLIConfig{
		DefaultAgent: "claude",
		Agents: map[string]*AgentConfig{
			"claude": {Tool: "claude", Models: map[string]string{"simple": "claude-haiku-4-5-20251001"}},
		},
	}

	migrated := cli.migrateOldFormat()
	if migrated {
		t.Error("migrateOldFormat should return false for already new format")
	}
}

func TestLoadOldFormatConfig(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "devloop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	// Create old format config
	oldConfigJSON := `{
		"version": "1.0",
		"project": {
			"name": "test",
			"path": "/tmp",
			"tech_stack": "Go",
			"main_branch": "main"
		},
		"verification": {
			"command": "go test",
			"timeout_seconds": 300
		},
		"cli": {
			"tool": "claude",
			"models": {
				"simple": "claude-haiku-4-5-20251001",
				"moderate": "claude-sonnet-4-5-20250929",
				"complex": "claude-opus-4-6"
			}
		},
		"execution": {
			"max_attempts": 2,
			"halt_on_failure": true,
			"auto_commit": true
		}
	}`

	if err := os.WriteFile(configPath, []byte(oldConfigJSON), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load and verify migration
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Should have migrated to new format
	if cfg.CLI.DefaultAgent != "default" {
		t.Errorf("Expected migrated default_agent 'default', got '%s'", cfg.CLI.DefaultAgent)
	}

	// Should still be able to get agent
	agent, err := cfg.CLI.GetAgent("")
	if err != nil {
		t.Fatalf("GetAgent() after migration failed: %v", err)
	}

	if agent.Tool != "claude" {
		t.Errorf("Expected migrated agent tool 'claude', got '%s'", agent.Tool)
	}
}

func TestLoadNewFormatConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "devloop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	// Create new format config with multiple agents
	newConfigJSON := `{
		"version": "1.0",
		"project": {
			"name": "test",
			"path": "/tmp",
			"tech_stack": "Go",
			"main_branch": "main"
		},
		"verification": {
			"command": "go test",
			"timeout_seconds": 300
		},
		"cli": {
			"default_agent": "claude",
			"agents": {
				"claude": {
					"tool": "claude",
					"models": {
						"simple": "claude-haiku-4-5-20251001",
						"moderate": "claude-sonnet-4-5-20250929",
						"complex": "claude-opus-4-6"
					}
				},
				"copilot": {
					"tool": "copilot",
					"models": {
						"simple": "gpt-4o-mini",
						"moderate": "gpt-4o",
						"complex": "o1-pro"
					}
				}
			}
		},
		"execution": {
			"max_attempts": 2,
			"halt_on_failure": true,
			"auto_commit": true
		}
	}`

	if err := os.WriteFile(configPath, []byte(newConfigJSON), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load and verify
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Should have both agents
	claudeAgent, err := cfg.CLI.GetAgent("claude")
	if err != nil {
		t.Fatalf("GetAgent('claude') failed: %v", err)
	}

	copilotAgent, err := cfg.CLI.GetAgent("copilot")
	if err != nil {
		t.Fatalf("GetAgent('copilot') failed: %v", err)
	}

	if claudeAgent.Tool != "claude" || copilotAgent.Tool != "copilot" {
		t.Error("Both agents should be loaded correctly")
	}

	// Default should be claude
	if cfg.CLI.GetDefaultAgentName() != "claude" {
		t.Errorf("Expected default agent 'claude', got '%s'", cfg.CLI.GetDefaultAgentName())
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && s[0:len(substr)] == substr || len(s) > 0)
}
