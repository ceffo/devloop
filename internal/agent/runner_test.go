package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewClaudeRunner(t *testing.T) {
	runner := NewClaudeRunner()
	if runner == nil {
		t.Fatal("NewClaudeRunner returned nil")
	}
}

func TestNewCopilotRunner(t *testing.T) {
	runner := NewCopilotRunner()
	if runner == nil {
		t.Fatal("NewCopilotRunner returned nil")
	}
}

func TestNewAgentRunner(t *testing.T) {
	tests := []struct {
		name         string
		tool         string
		wantErr      bool
		expectedType string
	}{
		{
			name:         "claude tool",
			tool:         "claude",
			wantErr:      false,
			expectedType: "*agent.ClaudeRunner",
		},
		{
			name:         "copilot tool",
			tool:         "copilot",
			wantErr:      false,
			expectedType: "*agent.CopilotRunner",
		},
		{
			name:    "unsupported tool",
			tool:    "unknown",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner, err := NewAgentRunner(tt.tool)
			if tt.wantErr {
				if err == nil {
					t.Error("NewAgentRunner expected error but got none")
				}
				if runner != nil {
					t.Error("NewAgentRunner should return nil runner on error")
				}
				return
			}

			if err != nil {
				t.Errorf("NewAgentRunner unexpected error: %v", err)
			}
			if runner == nil {
				t.Error("NewAgentRunner returned nil runner")
			}
		})
	}
}

func TestCopilotRunner_Run(t *testing.T) {
	runner := NewCopilotRunner()
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")

	result, err := runner.Run("test-model", "test prompt", logPath)

	// Should return an error since it's not implemented
	if err == nil {
		t.Error("CopilotRunner.Run expected error but got none")
	}

	if result == nil {
		t.Fatal("CopilotRunner.Run returned nil result")
	}

	if result.Success {
		t.Error("CopilotRunner.Run should not succeed (stub implementation)")
	}

	if result.Error == nil {
		t.Error("CopilotRunner.Run result should have Error set")
	}
}

func TestClaudeRunner_Run_LogFileCreation(t *testing.T) {
	runner := NewClaudeRunner()
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "logs", "test.log")

	// Use a command that will fail (claude likely not in PATH or with invalid args)
	// But we can still test that the log file is created
	result, _ := runner.Run("test-model", "test prompt", logPath)

	if result == nil {
		t.Fatal("ClaudeRunner.Run returned nil result")
	}

	// Check that log file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("ClaudeRunner.Run did not create log file")
	}

	// Check that LogPath is set correctly
	if result.LogPath != logPath {
		t.Errorf("Result.LogPath = %q, want %q", result.LogPath, logPath)
	}

	// Read log file content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)

	// Verify log file contains expected headers
	expectedHeaders := []string{
		"=== Claude Agent Execution ===",
		"Timestamp:",
		"Model: test-model",
		"Log Path:",
		"=== Prompt ===",
		"test prompt",
		"=== Output ===",
	}

	for _, header := range expectedHeaders {
		if !strings.Contains(contentStr, header) {
			t.Errorf("Log file missing expected header: %q", header)
		}
	}
}

func TestClaudeRunner_Run_DirectoryCreation(t *testing.T) {
	runner := NewClaudeRunner()
	tempDir := t.TempDir()

	// Use nested directory that doesn't exist
	logPath := filepath.Join(tempDir, "deep", "nested", "logs", "test.log")

	result, _ := runner.Run("test-model", "test prompt", logPath)

	if result == nil {
		t.Fatal("ClaudeRunner.Run returned nil result")
	}

	// Check that nested directories were created
	logDir := filepath.Dir(logPath)
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Error("ClaudeRunner.Run did not create nested log directories")
	}

	// Check that log file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("ClaudeRunner.Run did not create log file in nested directory")
	}
}

func TestAgentResult(t *testing.T) {
	// Test that AgentResult struct can be instantiated and has expected fields
	result := &AgentResult{
		Success: true,
		Output:  "test output",
		LogPath: "/path/to/log",
		Error:   nil,
	}

	if !result.Success {
		t.Error("AgentResult.Success not set correctly")
	}

	if result.Output != "test output" {
		t.Errorf("AgentResult.Output = %q, want %q", result.Output, "test output")
	}

	if result.LogPath != "/path/to/log" {
		t.Errorf("AgentResult.LogPath = %q, want %q", result.LogPath, "/path/to/log")
	}

	if result.Error != nil {
		t.Errorf("AgentResult.Error = %v, want nil", result.Error)
	}
}

func TestAgentRunner_Interface(t *testing.T) {
	// Verify that ClaudeRunner implements AgentRunner interface
	var _ AgentRunner = (*ClaudeRunner)(nil)

	// Verify that CopilotRunner implements AgentRunner interface
	var _ AgentRunner = (*CopilotRunner)(nil)
}
