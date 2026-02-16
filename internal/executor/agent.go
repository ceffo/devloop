package executor

// This file is deprecated. AgentRunner has been moved to internal/agent package.
// Kept for backwards compatibility with existing code.
// Import github.com/yourusername/devloop/internal/agent instead.

import "github.com/yourusername/devloop/internal/agent"

// Re-export types from agent package for backwards compatibility
type AgentRunner = agent.AgentRunner
type AgentResult = agent.AgentResult

// NewAgentRunner creates an appropriate AgentRunner based on the tool name
func NewAgentRunner(tool string) (AgentRunner, error) {
	return agent.NewAgentRunner(tool)
}
