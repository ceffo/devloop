// Package testutils provides shared helpers for devloop tests.
package testutils

import "github.com/ceffo/devloop/internal/config"

// MakeTestConfig returns a minimal but valid Config for use in tests.
// projectPath should be a temporary directory created by the test.
func MakeTestConfig(projectPath string) *config.Config {
	return &config.Config{
		Version: "1.0",
		Project: config.ProjectConfig{
			Name:       "test-project",
			Path:       projectPath,
			TechStack:  "Go",
			MainBranch: "main",
		},
		Verification: config.VerificationConfig{
			Command:        "go test",
			TimeoutSeconds: 300,
		},
		CLI: config.CLIConfig{
			Agents: map[string]*config.AgentConfig{
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
		Execution: config.ExecutionConfig{
			MaxAttempts:   2,
			HaltOnFailure: true,
			AutoCommit:    true,
			CommitFormat:  "task {task_id}: {title}",
		},
	}
}
