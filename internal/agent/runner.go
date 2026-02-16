package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// AgentRunner defines the interface for running AI CLI tools
type AgentRunner interface {
	Run(model, prompt, logPath string) (*AgentResult, error)
}

// AgentResult contains the result of an agent execution
type AgentResult struct {
	Success bool
	Output  string
	LogPath string
	Error   error
}

// ClaudeRunner implements AgentRunner for the Claude CLI
type ClaudeRunner struct{}

// NewClaudeRunner creates a new ClaudeRunner
func NewClaudeRunner() *ClaudeRunner {
	return &ClaudeRunner{}
}

// Run executes the Claude CLI with the given model and prompt
// Logs are written to logPath
func (c *ClaudeRunner) Run(model, prompt, logPath string) (*AgentResult, error) {
	result := &AgentResult{
		LogPath: logPath,
	}

	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create log directory: %w", err)
		return result, result.Error
	}

	// Create log file
	logFile, err := os.Create(logPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to create log file: %w", err)
		return result, result.Error
	}
	defer logFile.Close()

	// Write execution metadata to log
	fmt.Fprintf(logFile, "=== Claude Agent Execution ===\n")
	fmt.Fprintf(logFile, "Timestamp: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(logFile, "Model: %s\n", model)
	fmt.Fprintf(logFile, "Log Path: %s\n", logPath)
	fmt.Fprintf(logFile, "=== Prompt ===\n%s\n", prompt)
	fmt.Fprintf(logFile, "=== Output ===\n")

	// Build command: claude --model MODEL --dangerously-skip-permissions -p "PROMPT"
	cmd := exec.Command("claude", "--model", model, "--dangerously-skip-permissions", "-p", prompt)

	// Redirect stdout and stderr to both the log file and capture
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Execute command
	err = cmd.Run()

	// Read the output from the log file
	output, readErr := os.ReadFile(logPath)
	if readErr != nil {
		result.Error = fmt.Errorf("failed to read log file: %w", readErr)
		return result, result.Error
	}
	result.Output = string(output)

	// Check exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.Success = false
			result.Error = fmt.Errorf("claude command failed with exit code %d: %w", exitErr.ExitCode(), err)
			return result, nil // Return nil error as this is an expected failure case
		}
		result.Error = fmt.Errorf("failed to execute claude command: %w", err)
		return result, result.Error
	}

	result.Success = true
	return result, nil
}

// CopilotRunner implements AgentRunner for GitHub Copilot CLI (stub)
type CopilotRunner struct{}

// NewCopilotRunner creates a new CopilotRunner
func NewCopilotRunner() *CopilotRunner {
	return &CopilotRunner{}
}

// Run executes the Copilot CLI with the given model and prompt
func (c *CopilotRunner) Run(model, prompt, logPath string) (*AgentResult, error) {
	result := &AgentResult{
		LogPath: logPath,
	}

	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create log directory: %w", err)
		return result, result.Error
	}

	// Create log file
	logFile, err := os.Create(logPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to create log file: %w", err)
		return result, result.Error
	}
	defer logFile.Close()

	// Write execution metadata to log
	fmt.Fprintf(logFile, "=== Copilot Agent Execution ===\n")
	fmt.Fprintf(logFile, "Timestamp: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(logFile, "Model: %s\n", model)
	fmt.Fprintf(logFile, "Log Path: %s\n", logPath)
	fmt.Fprintf(logFile, "=== Prompt ===\n%s\n", prompt)
	fmt.Fprintf(logFile, "=== Output ===\n")

	// Build command: copilot -p "PROMPT" --model MODEL --allow-all-tools --silent
	cmd := exec.Command("copilot", "-p", prompt, "--model", model, "--allow-all-tools", "--silent")

	// Redirect stdout and stderr to both the log file and capture
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Execute command
	err = cmd.Run()

	// Read the output from the log file
	output, readErr := os.ReadFile(logPath)
	if readErr != nil {
		result.Error = fmt.Errorf("failed to read log file: %w", readErr)
		return result, result.Error
	}
	result.Output = string(output)

	// For CopilotRunner, this is a stub: always return an error indicating not implemented
	result.Success = false
	result.Error = fmt.Errorf("copilot runner not implemented")
	return result, result.Error
}

// NewAgentRunner creates an appropriate AgentRunner based on the tool name
func NewAgentRunner(tool string) (AgentRunner, error) {
	switch tool {
	case "claude":
		return NewClaudeRunner(), nil
	case "copilot":
		return NewCopilotRunner(), nil
	default:
		return nil, fmt.Errorf("unsupported agent tool: %s", tool)
	}
}
