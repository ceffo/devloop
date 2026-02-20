// Package agent provides interfaces and implementations for running AI CLI tools.
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

// Runner defines the interface for running AI CLI tools
type Runner interface {
	Run(model, prompt, logPath string) (*Result, error)
	// RunWithOutput is like Run but calls outputFn for each output line in real time.
	// Pass nil for outputFn to behave like Run.
	RunWithOutput(model, prompt, logPath string, outputFn OutputCallback) (*Result, error)
}

// Result contains the result of an agent execution
type Result struct {
	Success bool
	Output  string
	LogPath string
	Error   error
}

// runCLIAgent is the shared implementation for running any AI CLI agent.
// binary is the CLI binary name, cmdArgs builds the command arguments from model and prompt.
// header is the label used in the log file header (e.g. "Claude", "Copilot").
func runCLIAgent(binary, header, model, prompt, logPath string, buildArgs func(model, prompt string) []string, outputFn OutputCallback) (*Result, error) {
	result := &Result{LogPath: logPath}

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

	fmt.Fprintf(logFile, "=== %s Agent Execution ===\n", header)
	fmt.Fprintf(logFile, "Timestamp: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(logFile, "Model: %s\n", model)
	fmt.Fprintf(logFile, "Log Path: %s\n", logPath)
	fmt.Fprintf(logFile, "=== Prompt ===\n%s\n", prompt)
	fmt.Fprintf(logFile, "=== Output ===\n")

	cmd := exec.Command(binary, buildArgs(model, prompt)...)
	// Unset CLAUDECODE so the subprocess is not treated as a nested Claude Code session.
	// Claude Code blocks launch when CLAUDECODE is set in the environment.
	cmd.Env = filterEnv(os.Environ(), "CLAUDECODE")

	var runErr error
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

		runErr = cmd.Run()
		_ = pw.Close()
		wg.Wait()
	} else {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		runErr = cmd.Run()
	}

	output, readErr := os.ReadFile(logPath)
	if readErr != nil {
		result.Error = fmt.Errorf("failed to read log file: %w", readErr)
		return result, result.Error
	}
	result.Output = string(output)

	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			result.Success = false
			result.Error = fmt.Errorf("%s command failed with exit code %d: %w", binary, exitErr.ExitCode(), runErr)
			return result, nil
		}
		result.Error = fmt.Errorf("failed to execute %s command: %w", binary, runErr)
		return result, result.Error
	}

	result.Success = true
	return result, nil
}

// ClaudeRunner implements Runner for the Claude CLI
type ClaudeRunner struct{}

// NewClaudeRunner creates a new ClaudeRunner
func NewClaudeRunner() *ClaudeRunner {
	return &ClaudeRunner{}
}

// Run executes the Claude CLI with the given model and prompt.
// Logs are written to logPath.
func (c *ClaudeRunner) Run(model, prompt, logPath string) (*Result, error) {
	return c.RunWithOutput(model, prompt, logPath, nil)
}

// RunWithOutput executes the Claude CLI, streaming each output line to outputFn.
func (c *ClaudeRunner) RunWithOutput(model, prompt, logPath string, outputFn OutputCallback) (*Result, error) {
	return runCLIAgent("claude", "Claude", model, prompt, logPath, func(model, prompt string) []string {
		return []string{"--model", model, "--dangerously-skip-permissions", "-p", prompt}
	}, outputFn)
}

// CopilotRunner implements Runner for GitHub Copilot CLI
type CopilotRunner struct{}

// NewCopilotRunner creates a new CopilotRunner
func NewCopilotRunner() *CopilotRunner {
	return &CopilotRunner{}
}

// Run executes the Copilot CLI with the given model and prompt.
func (c *CopilotRunner) Run(model, prompt, logPath string) (*Result, error) {
	return c.RunWithOutput(model, prompt, logPath, nil)
}

// RunWithOutput executes the Copilot CLI, streaming each output line to outputFn.
func (c *CopilotRunner) RunWithOutput(model, prompt, logPath string, outputFn OutputCallback) (*Result, error) {
	return runCLIAgent("copilot", "Copilot", model, prompt, logPath, func(model, prompt string) []string {
		return []string{"--model", model, "--allow-all", "-p", prompt}
	}, outputFn)
}

// filterEnv returns os.Environ() with any variables whose name matches key removed.
func filterEnv(env []string, key string) []string {
	prefix := key + "="
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}

// NewAgentRunner creates an appropriate Runner based on the tool name
func NewAgentRunner(tool string) (Runner, error) {
	switch tool {
	case "claude":
		return NewClaudeRunner(), nil
	case "copilot":
		return NewCopilotRunner(), nil
	default:
		return nil, fmt.Errorf("unsupported agent tool: %s", tool)
	}
}
