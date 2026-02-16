package executor

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/storage"
)

// AutoCommit creates a git commit for a completed task
// It generates a commit message from the configured template, executes git add and commit,
// and returns the commit hash.
func AutoCommit(cfg *config.Config, task *storage.Task) (string, error) {
	// Validate that auto-commit is enabled
	if !cfg.Execution.AutoCommit {
		return "", fmt.Errorf("auto-commit is not enabled in configuration")
	}

	// Generate commit message from template
	commitMsg := generateCommitMessage(cfg.Execution.CommitFormat, task)

	// Execute git add -A
	addCmd := exec.Command("git", "add", "-A")
	addCmd.Dir = cfg.Project.Path

	var addOutput bytes.Buffer
	addCmd.Stdout = &addOutput
	addCmd.Stderr = &addOutput

	if err := addCmd.Run(); err != nil {
		return "", fmt.Errorf("git add failed: %w (output: %s)", err, addOutput.String())
	}

	// Execute git commit
	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	commitCmd.Dir = cfg.Project.Path

	var commitOutput bytes.Buffer
	commitCmd.Stdout = &commitOutput
	commitCmd.Stderr = &commitOutput

	if err := commitCmd.Run(); err != nil {
		return "", fmt.Errorf("git commit failed: %w (output: %s)", err, commitOutput.String())
	}

	// Extract commit hash from output
	commitHash, err := extractCommitHash(commitOutput.String())
	if err != nil {
		return "", fmt.Errorf("failed to extract commit hash: %w", err)
	}

	return commitHash, nil
}

// generateCommitMessage creates a commit message from a template string
// Supported template variables:
//   - {task_id}: Task ID
//   - {title}: Task title
//   - {description}: Task description
//   - {complexity}: Task complexity level
func generateCommitMessage(template string, task *storage.Task) string {
	msg := template

	// Replace template variables
	msg = strings.ReplaceAll(msg, "{task_id}", task.ID)
	msg = strings.ReplaceAll(msg, "{title}", task.Title)
	msg = strings.ReplaceAll(msg, "{description}", task.Description)
	msg = strings.ReplaceAll(msg, "{complexity}", task.Complexity)

	return msg
}

// extractCommitHash extracts the commit hash from git commit output
// Git commit output typically contains a line like:
// [main 1234567] commit message
// or
// [detached HEAD 1234567] commit message
func extractCommitHash(output string) (string, error) {
	// Try to find commit hash in format: [branch hash] message
	// Hash is typically 7 characters (short hash)
	re := regexp.MustCompile(`\[(?:[\w-]+\s+)?([0-9a-f]{7,})\]`)
	matches := re.FindStringSubmatch(output)

	if len(matches) >= 2 {
		return matches[1], nil
	}

	// Fallback: try to get hash using git rev-parse
	cmd := exec.Command("git", "rev-parse", "HEAD")
	var hashOutput bytes.Buffer
	cmd.Stdout = &hashOutput
	cmd.Stderr = &hashOutput

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get commit hash via rev-parse: %w", err)
	}

	hash := strings.TrimSpace(hashOutput.String())
	if hash == "" {
		return "", fmt.Errorf("empty commit hash")
	}

	// Return short hash (first 7 characters)
	if len(hash) > 7 {
		return hash[:7], nil
	}

	return hash, nil
}
