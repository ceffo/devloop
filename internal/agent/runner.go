package agent

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// OutputCallback is called for each line of agent stdout/stderr output.
// It is called from a goroutine; implementations must be goroutine-safe.
type OutputCallback func(line string)

// AgentRunner defines the interface for running AI CLI tools
type AgentRunner interface {
	Run(model, prompt, logPath string) (*AgentResult, error)
	// RunWithOutput is like Run but calls outputFn for each output line in real time.
	// Pass nil for outputFn to behave like Run.
	RunWithOutput(model, prompt, logPath string, outputFn OutputCallback) (*AgentResult, error)
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

// Run executes the Claude CLI with the given model and prompt.
// Logs are written to logPath.
func (c *ClaudeRunner) Run(model, prompt, logPath string) (*AgentResult, error) {
	return c.RunWithOutput(model, prompt, logPath, nil)
}

// RunWithOutput executes the Claude CLI, streaming each output line to outputFn.
func (c *ClaudeRunner) RunWithOutput(model, prompt, logPath string, outputFn OutputCallback) (*AgentResult, error) {
	result := &AgentResult{LogPath: logPath}

	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create log directory: %w", err)
		return result, result.Error
	}

	logFile, err := os.Create(logPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to create log file: %w", err)
		return result, result.Error
	}
	defer logFile.Close()

	fmt.Fprintf(logFile, "=== Claude Agent Execution ===\n")
	fmt.Fprintf(logFile, "Timestamp: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(logFile, "Model: %s\n", model)
	fmt.Fprintf(logFile, "Log Path: %s\n", logPath)
	fmt.Fprintf(logFile, "=== Prompt ===\n%s\n", prompt)
	fmt.Fprintf(logFile, "=== Output ===\n")

	cmd := exec.Command("claude", "--model", model, "--dangerously-skip-permissions", "-p", prompt)

	if outputFn != nil {
		// Stream output: pipe stdout/stderr through a scanner that calls outputFn per line,
		// while also writing to the log file.
		pr, pw := io.Pipe()
		cmd.Stdout = io.MultiWriter(logFile, pw)
		cmd.Stderr = io.MultiWriter(logFile, pw)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(pr)
			for scanner.Scan() {
				outputFn(scanner.Text())
			}
		}()

		err = cmd.Run()
		pw.Close()
		wg.Wait()
	} else {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		err = cmd.Run()
	}

	output, readErr := os.ReadFile(logPath)
	if readErr != nil {
		result.Error = fmt.Errorf("failed to read log file: %w", readErr)
		return result, result.Error
	}
	result.Output = string(output)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.Success = false
			result.Error = fmt.Errorf("claude command failed with exit code %d: %w", exitErr.ExitCode(), err)
			return result, nil
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

// Run executes the Copilot CLI with the given model and prompt.
func (c *CopilotRunner) Run(model, prompt, logPath string) (*AgentResult, error) {
	return c.RunWithOutput(model, prompt, logPath, nil)
}

// RunWithOutput executes the Copilot CLI, streaming each output line to outputFn.
// This is currently a stub and always returns an error.
func (c *CopilotRunner) RunWithOutput(model, prompt, logPath string, outputFn OutputCallback) (*AgentResult, error) {
	result := &AgentResult{LogPath: logPath}

	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create log directory: %w", err)
		return result, result.Error
	}

	logFile, err := os.Create(logPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to create log file: %w", err)
		return result, result.Error
	}
	defer logFile.Close()

	fmt.Fprintf(logFile, "=== Copilot Agent Execution ===\n")
	fmt.Fprintf(logFile, "Timestamp: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(logFile, "Model: %s\n", model)
	fmt.Fprintf(logFile, "Log Path: %s\n", logPath)
	fmt.Fprintf(logFile, "=== Prompt ===\n%s\n", prompt)
	fmt.Fprintf(logFile, "=== Output ===\n")

	// Stub: not implemented
	_ = outputFn
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
