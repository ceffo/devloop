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

	if cfg.CLI.GetDefaultAgentName() != "claude" {
		t.Errorf("Expected default agent 'claude', got '%s'", cfg.CLI.GetDefaultAgentName())
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
		},
		Execution: ExecutionConfig{
			MaxAttempts:   2,
			HaltOnFailure: true,
			AutoCommit:    true,
			CommitFormat:  "task {task_id}: {title}",
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

	if cfg.CLI.GetDefaultAgentName() != "claude" {
		t.Errorf("Expected default agent to be 'claude', got '%s'", cfg.CLI.GetDefaultAgentName())
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

	// Empty name should return first agent alphabetically
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
			// No DefaultAgent field anymore; first alphabetically wins
			Agents: map[string]*AgentConfig{
				"claude":  {Tool: "claude"},
				"copilot": {Tool: "copilot"},
			},
		},
	}

	name := cfg.CLI.GetDefaultAgentName()
	if name != "claude" {
		t.Errorf("Expected 'claude' (first alphabetically), got '%s'", name)
	}
}

func TestGetDefaultAgentName_Alphabetical(t *testing.T) {
	cfg := &Config{
		CLI: CLIConfig{
			// DefaultAgent not set, should use alphabetical
			Agents: map[string]*AgentConfig{
				"zulu":  {Tool: "claude"},
				"alpha": {Tool: "copilot"},
				"bravo": {Tool: "claude"},
			},
		},
	}

	name := cfg.CLI.GetDefaultAgentName()
	if name != "alpha" {
		t.Errorf("Expected 'alpha' (first alphabetically), got '%s'", name)
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

	// Default should be claude (first alphabetically)
	if cfg.CLI.GetDefaultAgentName() != "claude" {
		t.Errorf("Expected default agent 'claude', got '%s'", cfg.CLI.GetDefaultAgentName())
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && s[0:len(substr)] == substr || len(s) > 0)
}

// Tests for CoordinatorConfig defaults

func TestCoordinatorConfig_EnabledDefaultsMaxSubtasks(t *testing.T) {
	// Test that MaxSubtasks defaults to 5 when Enabled is true and MaxSubtasks is 0
	tempDir, err := os.MkdirTemp("", "devloop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	coordinatorJSON := `{
		"version": "1.0",
		"project": {"name": "test", "path": "/tmp", "tech_stack": "Go"},
		"verification": {"command": "go test", "timeout_seconds": 300},
		"cli": {"agents": {"claude": {"tool": "claude", "models": {"simple": "claude-haiku"}}}},
		"execution": {
			"max_attempts": 2,
			"coordinator": {
				"enabled": true,
				"agent": "claude",
				"model": "claude-opus-4-6",
				"max_subtasks": 0
			}
		}
	}`

	if err := os.WriteFile(configPath, []byte(coordinatorJSON), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if !cfg.Execution.Coordinator.Enabled {
		t.Errorf("Expected coordinator enabled to be true, got %v", cfg.Execution.Coordinator.Enabled)
	}

	if cfg.Execution.Coordinator.MaxSubtasks != 5 {
		t.Errorf("Expected max_subtasks to default to 5, got %d", cfg.Execution.Coordinator.MaxSubtasks)
	}

	if cfg.Execution.Coordinator.Agent != "claude" {
		t.Errorf("Expected agent to be 'claude', got '%s'", cfg.Execution.Coordinator.Agent)
	}

	if cfg.Execution.Coordinator.Model != "claude-opus-4-6" {
		t.Errorf("Expected model to be 'claude-opus-4-6', got '%s'", cfg.Execution.Coordinator.Model)
	}
}

func TestCoordinatorConfig_PreservesCustomMaxSubtasks(t *testing.T) {
	// Test that custom MaxSubtasks values are preserved
	tempDir, err := os.MkdirTemp("", "devloop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	coordinatorJSON := `{
		"version": "1.0",
		"project": {"name": "test", "path": "/tmp", "tech_stack": "Go"},
		"verification": {"command": "go test", "timeout_seconds": 300},
		"cli": {"agents": {"claude": {"tool": "claude", "models": {"simple": "claude-haiku"}}}},
		"execution": {
			"max_attempts": 2,
			"coordinator": {
				"enabled": true,
				"agent": "claude",
				"model": "claude-opus-4-6",
				"max_subtasks": 10
			}
		}
	}`

	if err := os.WriteFile(configPath, []byte(coordinatorJSON), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Execution.Coordinator.MaxSubtasks != 10 {
		t.Errorf("Expected max_subtasks to remain 10, got %d", cfg.Execution.Coordinator.MaxSubtasks)
	}
}

func TestCoordinatorConfig_DisabledDoesNotDefaultMaxSubtasks(t *testing.T) {
	// Test that MaxSubtasks doesn't get defaulted when Enabled is false
	tempDir, err := os.MkdirTemp("", "devloop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	coordinatorJSON := `{
		"version": "1.0",
		"project": {"name": "test", "path": "/tmp", "tech_stack": "Go"},
		"verification": {"command": "go test", "timeout_seconds": 300},
		"cli": {"agents": {"claude": {"tool": "claude", "models": {"simple": "claude-haiku"}}}},
		"execution": {
			"max_attempts": 2,
			"coordinator": {
				"enabled": false,
				"agent": "claude",
				"model": "claude-opus-4-6",
				"max_subtasks": 0
			}
		}
	}`

	if err := os.WriteFile(configPath, []byte(coordinatorJSON), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Execution.Coordinator.Enabled {
		t.Errorf("Expected coordinator enabled to be false, got %v", cfg.Execution.Coordinator.Enabled)
	}

	if cfg.Execution.Coordinator.MaxSubtasks != 0 {
		t.Errorf("Expected max_subtasks to remain 0 when disabled, got %d", cfg.Execution.Coordinator.MaxSubtasks)
	}
}

func TestCoordinatorConfig_BackwardCompatibility(t *testing.T) {
	// Test that old configs without coordinator field still load correctly
	tempDir, err := os.MkdirTemp("", "devloop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	oldJSON := `{
		"version": "1.0",
		"project": {"name": "test", "path": "/tmp", "tech_stack": "Go"},
		"verification": {"command": "go test", "timeout_seconds": 300},
		"cli": {"agents": {"claude": {"tool": "claude", "models": {"simple": "claude-haiku"}}}},
		"execution": {
			"max_attempts": 2,
			"halt_on_failure": true,
			"auto_commit": true
		}
	}`

	if err := os.WriteFile(configPath, []byte(oldJSON), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig should return valid config")
	}

	// Verify the old fields still work
	if cfg.Execution.MaxAttempts != 2 {
		t.Errorf("Expected max_attempts 2, got %d", cfg.Execution.MaxAttempts)
	}

	if !cfg.Execution.HaltOnFailure {
		t.Errorf("Expected halt_on_failure to be true")
	}

	if !cfg.Execution.AutoCommit {
		t.Errorf("Expected auto_commit to be true")
	}

	// Verify coordinator field has zero values (not set)
	if cfg.Execution.Coordinator.Enabled {
		t.Errorf("Expected coordinator enabled to be false for old config")
	}

	if cfg.Execution.Coordinator.MaxSubtasks != 0 {
		t.Errorf("Expected max_subtasks to be 0 for old config, got %d", cfg.Execution.Coordinator.MaxSubtasks)
	}
}
