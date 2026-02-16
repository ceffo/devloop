package executor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yourusername/devloop/internal/config"
)

func TestRunVerification_Success(t *testing.T) {
	// Create temp directory for test project
	tempDir := t.TempDir()

	// Create .devloop/logs directory
	logsDir := filepath.Join(tempDir, ".devloop", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatalf("Failed to create logs directory: %v", err)
	}

	// Create test config with successful command
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: tempDir,
		},
		Verification: config.VerificationConfig{
			Command:        "echo 'test passed'",
			TimeoutSeconds: 5,
		},
	}

	// Run verification
	result, err := RunVerification(cfg, "1.1")
	if err != nil {
		t.Fatalf("RunVerification returned error: %v", err)
	}

	// Check success
	if !result.Success {
		t.Errorf("Expected success=true, got success=false")
	}

	// Check output
	if !strings.Contains(result.Output, "test passed") {
		t.Errorf("Expected output to contain 'test passed', got: %s", result.Output)
	}

	// Check log path
	if result.LogPath == "" {
		t.Error("Expected log path to be set")
	}

	// Verify log file exists
	if _, err := os.Stat(result.LogPath); os.IsNotExist(err) {
		t.Errorf("Log file does not exist at %s", result.LogPath)
	}

	// Check log file contains expected content
	logContent, err := os.ReadFile(result.LogPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	logStr := string(logContent)

	expectedStrings := []string{
		"Verification for task: 1.1",
		"Command: echo 'test passed'",
		"test passed",
		"--- Output ---",
	}
	for _, expected := range expectedStrings {
		if !strings.Contains(logStr, expected) {
			t.Errorf("Log file missing expected string: %s", expected)
		}
	}

	// Check duration
	if result.Duration < 0 {
		t.Errorf("Expected non-negative duration, got: %d", result.Duration)
	}
}

func TestRunVerification_Failure(t *testing.T) {
	// Create temp directory for test project
	tempDir := t.TempDir()

	// Create test config with failing command
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: tempDir,
		},
		Verification: config.VerificationConfig{
			Command:        "exit 1",
			TimeoutSeconds: 5,
		},
	}

	// Run verification
	result, err := RunVerification(cfg, "2.1")
	if err != nil {
		t.Fatalf("RunVerification returned error: %v", err)
	}

	// Check failure
	if result.Success {
		t.Error("Expected success=false for failing command, got success=true")
	}

	// Check log path exists
	if result.LogPath == "" {
		t.Error("Expected log path to be set even on failure")
	}

	// Verify log file exists
	if _, err := os.Stat(result.LogPath); os.IsNotExist(err) {
		t.Errorf("Log file does not exist at %s", result.LogPath)
	}

	// Check log file contains error
	logContent, err := os.ReadFile(result.LogPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	logStr := string(logContent)

	if !strings.Contains(logStr, "Error:") {
		t.Error("Log file should contain error information")
	}
}

func TestRunVerification_CapturesStderr(t *testing.T) {
	// Create temp directory for test project
	tempDir := t.TempDir()

	// Create test config with command that writes to stderr
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: tempDir,
		},
		Verification: config.VerificationConfig{
			Command:        "echo 'error message' >&2",
			TimeoutSeconds: 5,
		},
	}

	// Run verification
	result, err := RunVerification(cfg, "3.1")
	if err != nil {
		t.Fatalf("RunVerification returned error: %v", err)
	}

	// Check that stderr was captured
	if !strings.Contains(result.Output, "error message") {
		t.Errorf("Expected output to contain stderr message 'error message', got: %s", result.Output)
	}
}

func TestRunVerification_Timeout(t *testing.T) {
	// Create temp directory for test project
	tempDir := t.TempDir()

	// Create test config with command that sleeps longer than timeout
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: tempDir,
		},
		Verification: config.VerificationConfig{
			Command:        "sleep 10",
			TimeoutSeconds: 1,
		},
	}

	// Run verification
	result, err := RunVerification(cfg, "4.1")
	if err != nil {
		t.Fatalf("RunVerification returned error: %v", err)
	}

	// Check failure due to timeout
	if result.Success {
		t.Error("Expected success=false for timeout, got success=true")
	}

	// Check timeout message in output
	if !strings.Contains(result.Output, "TIMEOUT") {
		t.Errorf("Expected output to contain TIMEOUT message, got: %s", result.Output)
	}

	// Check log path exists
	if result.LogPath == "" {
		t.Error("Expected log path to be set even on timeout")
	}
}

func TestRunVerification_EmptyCommand(t *testing.T) {
	// Create temp directory for test project
	tempDir := t.TempDir()

	// Create test config with empty command
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: tempDir,
		},
		Verification: config.VerificationConfig{
			Command:        "",
			TimeoutSeconds: 5,
		},
	}

	// Run verification
	_, err := RunVerification(cfg, "5.1")
	if err == nil {
		t.Error("Expected error for empty command, got nil")
	}

	if !strings.Contains(err.Error(), "verification command is not configured") {
		t.Errorf("Expected error about missing command, got: %v", err)
	}
}

func TestRunVerification_ComplexCommand(t *testing.T) {
	// Create temp directory for test project
	tempDir := t.TempDir()

	// Create test config with complex command (pipes, multiple commands)
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: tempDir,
		},
		Verification: config.VerificationConfig{
			Command:        "echo 'line1' && echo 'line2' | grep 'line'",
			TimeoutSeconds: 5,
		},
	}

	// Run verification
	result, err := RunVerification(cfg, "6.1")
	if err != nil {
		t.Fatalf("RunVerification returned error: %v", err)
	}

	// Check success
	if !result.Success {
		t.Error("Expected success=true for complex command")
	}

	// Check output contains both lines
	if !strings.Contains(result.Output, "line1") {
		t.Error("Expected output to contain 'line1'")
	}
	if !strings.Contains(result.Output, "line2") {
		t.Error("Expected output to contain 'line2'")
	}
}

func TestRunVerification_LogFileFormat(t *testing.T) {
	// Create temp directory for test project
	tempDir := t.TempDir()

	// Create test config
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: tempDir,
		},
		Verification: config.VerificationConfig{
			Command:        "echo 'test'",
			TimeoutSeconds: 5,
		},
	}

	// Run verification
	result, err := RunVerification(cfg, "7.1")
	if err != nil {
		t.Fatalf("RunVerification returned error: %v", err)
	}

	// Check log file name format
	logFileName := filepath.Base(result.LogPath)
	if !strings.HasPrefix(logFileName, "verify-7.1-") {
		t.Errorf("Expected log file to start with 'verify-7.1-', got: %s", logFileName)
	}
	if !strings.HasSuffix(logFileName, ".log") {
		t.Errorf("Expected log file to end with '.log', got: %s", logFileName)
	}

	// Check log file is in correct directory
	expectedDir := filepath.Join(tempDir, ".devloop", "logs")
	logDir := filepath.Dir(result.LogPath)
	if logDir != expectedDir {
		t.Errorf("Expected log in %s, got: %s", expectedDir, logDir)
	}
}

func TestRunVerification_CreatesLogsDirectory(t *testing.T) {
	// Create temp directory for test project (without logs directory)
	tempDir := t.TempDir()

	// Create test config
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: tempDir,
		},
		Verification: config.VerificationConfig{
			Command:        "echo 'test'",
			TimeoutSeconds: 5,
		},
	}

	// Verify logs directory doesn't exist initially
	logsDir := filepath.Join(tempDir, ".devloop", "logs")
	if _, err := os.Stat(logsDir); !os.IsNotExist(err) {
		t.Skip("Logs directory already exists, skipping test")
	}

	// Run verification
	_, err := RunVerification(cfg, "8.1")
	if err != nil {
		t.Fatalf("RunVerification returned error: %v", err)
	}

	// Verify logs directory was created
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		t.Error("Expected logs directory to be created")
	}
}

func TestRunVerification_DurationTracking(t *testing.T) {
	// Create temp directory for test project
	tempDir := t.TempDir()

	// Create test config with command that takes some time
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Path: tempDir,
		},
		Verification: config.VerificationConfig{
			Command:        "sleep 0.5",
			TimeoutSeconds: 5,
		},
	}

	// Run verification
	result, err := RunVerification(cfg, "9.1")
	if err != nil {
		t.Fatalf("RunVerification returned error: %v", err)
	}

	// Check that duration is at least 0 seconds (sleep 0.5 might round to 0)
	if result.Duration < 0 {
		t.Errorf("Expected duration >= 0, got: %d", result.Duration)
	}

	// Duration should be reasonable (not more than timeout + 1 second buffer)
	if result.Duration > cfg.Verification.TimeoutSeconds+1 {
		t.Errorf("Expected duration <= %d, got: %d", cfg.Verification.TimeoutSeconds+1, result.Duration)
	}
}
