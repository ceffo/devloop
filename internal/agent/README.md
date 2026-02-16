# Agent Package

This package contains the AgentRunner interface and implementations for running AI CLI tools.

## Purpose

The `agent` package was created to break an import cycle between `executor` and `processor` packages. Both packages need to run AI agents, so the AgentRunner interface and implementations were extracted into this shared package.

## Components

- **AgentRunner**: Interface for running AI CLI tools
- **AgentResult**: Result structure containing execution status, output, and error information
- **ClaudeRunner**: Implementation for Claude CLI
- **CopilotRunner**: Stub implementation for GitHub Copilot CLI (future)
- **NewAgentRunner**: Factory function to create appropriate runner based on tool name

## Usage

```go
import "github.com/yourusername/devloop/internal/agent"

// Create a runner for a specific tool
runner, err := agent.NewAgentRunner("claude")
if err != nil {
    // Handle error
}

// Run the agent
result, err := runner.Run(model, prompt, logPath)
if err != nil {
    // Handle error
}

if result.Success {
    // Agent execution succeeded
}
```

## Architecture

The package dependency graph:
- `agent` (no dependencies on executor/processor)
- `prompts` imports `processor`
- `executor` imports `agent` and `prompts`
- `processor` imports `agent`

This structure avoids circular dependencies while allowing code reuse.
