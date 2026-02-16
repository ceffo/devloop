package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/yourusername/devloop/internal/config"
)

// VerifyResult holds the results of a verification execution
type VerifyResult struct {
	Success  bool   // Whether verification passed (exit code 0)
	Output   string // Combined stdout/stderr output
	LogPath  string // Path to verification log file
	Duration int    // Execution duration in seconds
}

// RunVerification executes the verification command for a task
// It runs the command from config.Verification in the project directory,
// respects the configured timeout, captures output, and writes a log file.
func RunVerification(cfg *config.Config, taskID string) (*VerifyResult, error) {
	// Validate input
	if cfg.Verification.Command == "" {
		return nil, fmt.Errorf("verification command is not configured")
	}

	// Create logs directory if it doesn't exist
	logsDir := filepath.Join(cfg.Project.Path, ".devloop", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Generate log file path with timestamp
	timestamp := time.Now().Format("20060102-150405")
	logPath := filepath.Join(logsDir, fmt.Sprintf("verify-%s-%s.log", taskID, timestamp))

	// Track start time for duration calculation
	startTime := time.Now()

	// Create context with timeout
	timeoutDuration := time.Duration(cfg.Verification.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	// Prepare command
	// Run command in shell to support complex commands with pipes, etc.
	cmd := exec.CommandContext(ctx, "sh", "-c", cfg.Verification.Command)
	cmd.Dir = cfg.Project.Path

	// Capture stdout and stderr
	var outputBuf bytes.Buffer
	cmd.Stdout = &outputBuf
	cmd.Stderr = &outputBuf

	// Execute command
	execErr := cmd.Run()

	// Calculate duration
	duration := int(time.Since(startTime).Seconds())

	// Get output
	output := outputBuf.String()

	// Write log file
	logContent := fmt.Sprintf("Verification for task: %s\n", taskID)
	logContent += fmt.Sprintf("Command: %s\n", cfg.Verification.Command)
	logContent += fmt.Sprintf("Started at: %s\n", startTime.Format(time.RFC3339))
	logContent += fmt.Sprintf("Duration: %d seconds\n", duration)
	logContent += fmt.Sprintf("Working directory: %s\n", cfg.Project.Path)
	logContent += "\n--- Output ---\n"
	logContent += output
	logContent += "\n--- End Output ---\n"

	if execErr != nil {
		logContent += fmt.Sprintf("\nError: %v\n", execErr)
	}

	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write verification log: %w", err)
	}

	// Determine success
	success := execErr == nil

	// Handle timeout separately
	if ctx.Err() == context.DeadlineExceeded {
		return &VerifyResult{
			Success:  false,
			Output:   output + fmt.Sprintf("\n[TIMEOUT] Command exceeded %d seconds", cfg.Verification.TimeoutSeconds),
			LogPath:  logPath,
			Duration: duration,
		}, nil
	}

	return &VerifyResult{
		Success:  success,
		Output:   output,
		LogPath:  logPath,
		Duration: duration,
	}, nil
}
