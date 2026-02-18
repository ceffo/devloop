package processor

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/ceffo/devloop/internal/agent"
	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/storage"
)

// prdProcessingPromptTemplate is the template for decomposing a PRD into structured tasks
const prdProcessingPromptTemplate = `You are a task planning AI for the devloop system.

## Project Context
Name: {{.ProjectName}}
Tech Stack: {{.TechStack}}
Path: {{.ProjectPath}}

## Your Task
Analyze the following Product Requirements Document (PRD) and break it down into
structured, atomic, implementable tasks.

## PRD Content
{{.PRDContent}}

## Instructions
1. Decompose the PRD into atomic tasks — each should be independently implementable in one session
2. Assign task IDs starting from {{.NextID}}
3. For each task, determine:
   - Title (brief, imperative, < 60 characters)
   - Description (what to implement and why, 2-4 sentences)
   - Complexity: simple | moderate | complex
   - Acceptance criteria (3-5 concrete, testable outcomes)
   - Dependencies (blocked_by task IDs — only true hard dependencies, not soft ordering)
   - Tags (for categorization, e.g., "api", "storage", "cli", "testing")

4. Output format: JSON array of task objects
   [
     {
       "id": "{{.NextID}}",
       "title": "...",
       "description": "...",
       "complexity": "simple|moderate|complex",
       "acceptance_criteria": ["...", "..."],
       "blocked_by": [],
       "tags": ["..."]
     }
   ]

## Guidelines
- simple: Single-file changes, < 50 lines, straightforward logic
- moderate: Multi-file changes, 50-200 lines, state management or coordination
- complex: Algorithms, intricate workflows, architectural decisions
- Make every task atomic and independently testable
- blocked_by should reference IDs of other tasks in this output — only add when there is a true implementation dependency
- Do not create tasks for things already marked as complete in the PRD

Output the JSON array only, no additional text.`

// prdPromptData holds data for rendering the PRD processing prompt
type prdPromptData struct {
	ProjectName string
	TechStack   string
	ProjectPath string
	PRDContent  string
	NextID      string
}

// ReadPRDFile reads the full content of a PRD file.
func ReadPRDFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read PRD file: %w", err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("PRD file is empty: %s", path)
	}
	return string(data), nil
}

// renderPRDPrompt generates a prompt for decomposing a PRD into structured tasks
func renderPRDPrompt(project config.ProjectConfig, prdContent string, nextID string) (string, error) {
	data := prdPromptData{
		ProjectName: project.Name,
		TechStack:   project.TechStack,
		ProjectPath: project.Path,
		PRDContent:  prdContent,
		NextID:      nextID,
	}

	tmpl, err := template.New("prd").Parse(prdProcessingPromptTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse PRD prompt template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render PRD prompt: %w", err)
	}

	return buf.String(), nil
}

// ProcessPRD reads a PRD file and converts it into structured tasks using an AI agent.
func ProcessPRD(cfg *config.Config, filePath string) ([]*storage.Task, error) {
	prdContent, err := ReadPRDFile(filePath)
	if err != nil {
		return nil, err
	}

	store := storage.NewStorage(cfg)
	nextID, err := generateNextTaskID(store, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to generate next task ID: %w", err)
	}

	prompt, err := renderPRDPrompt(cfg.Project, prdContent, nextID)
	if err != nil {
		return nil, fmt.Errorf("failed to render PRD prompt: %w", err)
	}

	defaultAgentName := cfg.CLI.GetDefaultAgentName()
	agentConfig, err := cfg.CLI.GetAgent(defaultAgentName)
	if err != nil {
		return nil, fmt.Errorf("failed to get default agent configuration: %w", err)
	}

	agentRunner, err := agent.NewAgentRunner(agentConfig.Tool)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent runner: %w", err)
	}

	logPath := filepath.Join(cfg.Project.Path, ".devloop", "logs",
		fmt.Sprintf("prd-processing-%d.log", time.Now().Unix()))
	model := agentConfig.Models["complex"]
	result, err := agentRunner.Run(model, prompt, logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to run AI agent: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("AI agent execution failed: %v", result.Error)
	}

	tasks, err := parseTasksFromJSON(result.Output, cfg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tasks from JSON: %w", err)
	}

	for _, task := range tasks {
		task.Metadata.SourceType = "prd"
	}

	return tasks, nil
}
