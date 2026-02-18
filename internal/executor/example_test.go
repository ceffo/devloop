package executor_test

import (
	"fmt"
	"path/filepath"

	"github.com/ceffo/devloop/internal/agent"
)

// Example demonstrates how to use the Runner interface
func ExampleRunner() {
	// Create a Claude runner
	runner := agent.NewClaudeRunner()

	// Define execution parameters
	model := "claude-sonnet-4-5-20250929"
	prompt := "Write a simple Hello World function in Go"
	logPath := filepath.Join(".devloop", "logs", "task-1.1-attempt-1.log")

	// Execute the agent
	result, err := runner.Run(model, prompt, logPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Check result
	if result.Success {
		fmt.Println("Agent execution succeeded")
		fmt.Printf("Output logged to: %s\n", result.LogPath)
	} else {
		fmt.Println("Agent execution failed")
		if result.Error != nil {
			fmt.Printf("Error: %v\n", result.Error)
		}
	}
}

// Example demonstrates how to create an agent runner based on config
func ExampleNewAgentRunner() {
	// Get the tool name from config (e.g., "claude" or "copilot")
	tool := "claude"

	// Create appropriate runner
	runner, err := agent.NewAgentRunner(tool)
	if err != nil {
		fmt.Printf("Error creating runner: %v\n", err)
		return
	}

	fmt.Printf("Created runner for tool: %s\n", tool)
	_ = runner // Use the runner
}
