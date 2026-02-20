package prompts

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/processor"
	"github.com/ceffo/devloop/internal/storage"
)

// TodoProcessingPrompt is the template for converting TODO items into structured tasks
const TodoProcessingPrompt = `You are a task planning AI for the devloop system.

## Project Context
Name: {{.ProjectName}}
Tech Stack: {{.TechStack}}
Path: {{.ProjectPath}}

## Your Task
Analyze the following TODO items and convert them into structured, executable tasks.

## TODO Items
{{range .Todos}}
- [{{.ID}}] {{.Category}} | Priority: {{.Priority}}
  Content: {{.Content}}
{{end}}

## Instructions
1. Group related TODO items into logical tasks
2. Assign hierarchical task IDs starting from {{.NextID}}
3. For each task, determine:
   - Title (brief, imperative)
   - Description (detailed explanation)
   - Complexity: simple | moderate | complex
   - Acceptance criteria (3-5 testable outcomes)
   - Dependencies (blockedBy task IDs)
   - Tags (for categorization)

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
- simple: Single-file changes, < 50 lines
- moderate: Multi-file changes, 50-200 lines, state management
- complex: Algorithms, intricate workflows, architectural decisions
- Make tasks atomic and independently testable
- Clearly specify acceptance criteria
- Only add blockedBy if there's a true dependency

Output the JSON array only, no additional text.`

// TaskExecutionPrompt is the template for executing individual tasks
const TaskExecutionPrompt = `You are executing a development task in the devloop system.

## Project Context
Name: {{.ProjectName}}
Tech Stack: {{.TechStack}}
Path: {{.ProjectPath}}
Main Branch: {{.MainBranch}}

## Task Details
ID: {{.TaskID}}
Title: {{.Title}}
Complexity: {{.Complexity}}
Attempt: {{.Attempt}} of {{.MaxAttempts}}

## Description
{{.Description}}

## Acceptance Criteria
{{range .AcceptanceCriteria}}- {{.}}
{{end}}

## Verification
After completing this task, the following command will be run to verify success:
` + "`" + `{{.VerificationCommand}}` + "`" + `

{{if .PreviousError}}## Previous Attempt Error
The previous attempt failed with the following error:
` + "```" + `
{{.PreviousError}}
` + "```" + `

Please analyze the error and correct the implementation.
{{end}}

{{if .CustomInstructions}}## Custom Instructions
{{.CustomInstructions}}
{{end}}

## Your Task
Implement this task according to the description and acceptance criteria.
Ensure all criteria are met and the verification command will pass.`

// TodoPromptData holds data for rendering the TODO processing prompt
type TodoPromptData struct {
	ProjectName string
	TechStack   string
	ProjectPath string
	Todos       []processor.TodoItem
	NextID      string
}

// TaskPromptData holds data for rendering the task execution prompt
type TaskPromptData struct {
	ProjectName         string
	TechStack           string
	ProjectPath         string
	MainBranch          string
	TaskID              string
	Title               string
	Complexity          string
	Attempt             int
	MaxAttempts         int
	Description         string
	AcceptanceCriteria  []string
	VerificationCommand string
	PreviousError       string
	CustomInstructions  string
}

// RenderTodoPrompt generates a prompt for processing TODO items into tasks
func RenderTodoPrompt(project config.ProjectConfig, todos []processor.TodoItem, nextID string) (string, error) {
	data := TodoPromptData{
		ProjectName: project.Name,
		TechStack:   project.TechStack,
		ProjectPath: project.Path,
		Todos:       todos,
		NextID:      nextID,
	}

	tmpl, err := template.New("todo").Parse(TodoProcessingPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to parse TODO prompt template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render TODO prompt: %w", err)
	}

	return buf.String(), nil
}

// Sentinel values output by the coordinator agent to indicate its decision.
const (
	SentinelProceed    = "##DEVLOOP:PROCEED##"
	SentinelDecomposed = "##DEVLOOP:DECOMPOSED##"
)

// CoordinatorPrompt is the template for the coordinator agent that decides whether
// to decompose a complex task into focused subtasks.
const CoordinatorPrompt = `You are the devloop Coordinator Agent. Your role is to analyze a complex task and decide
whether to decompose it into smaller, focused subtasks.

## Project Context
Name: {{.ProjectName}}
Tech Stack: {{.TechStack}}
Path: {{.ProjectPath}}

## Task to Evaluate
ID: {{.TaskID}}
Title: {{.Title}}
Description: {{.Description}}

Acceptance Criteria:
{{range .AcceptanceCriteria}}- {{.}}
{{end}}
## Task Store Commands
You have access to the following commands to create and link subtasks:
{{.TaskStoreInstructions}}

## Instructions
1. Analyze this task carefully.
2. If the task is focused enough to be implemented in one coding session (even if it's complex):
   Output exactly on its own line: ##DEVLOOP:PROCEED##
3. If the task genuinely needs to be split (multiple distinct components or concerns):
   - Create 2 to {{.MaxSubtasks}} focused subtasks using the task store commands
   - Each subtask must be independently implementable and verifiable
   - Use dependencies to order subtasks when necessary
   - After creating subtasks, output on its own line: ##DEVLOOP:DECOMPOSED##

Output ONLY the commands (one per line) followed by the sentinel.
Do not explain your reasoning. Do not include any other text.`

// jsonlCoordinatorInstructions are the task store commands for the JSONL backend.
const jsonlCoordinatorInstructions = `Write subtask definitions as a JSON array to the file .devloop/coordinator-output.json.
Each object must have: title, description, complexity, acceptance_criteria, blocked_by, tags.`

// beadsCoordinatorInstructions is the format string for Beads backend task store commands.
// The single %s placeholder is replaced with the parent task ID.
const beadsCoordinatorInstructionsFmt = `bd update %s --claim                                        : claim the parent task
bd create "<title>" --parent %s --body-file=desc.md --json  : create subtask (auto-assigns %s.1, .2, etc.)`

// CoordinatorPromptData holds data for rendering the coordinator prompt.
type CoordinatorPromptData struct {
	ProjectName           string
	TechStack             string
	ProjectPath           string
	TaskID                string
	Title                 string
	Description           string
	AcceptanceCriteria    []string
	TaskStoreInstructions string
	MaxSubtasks           int
}

// RenderCoordinatorPrompt generates a prompt for the coordinator agent.
// The TaskStoreInstructions field is populated based on cfg.Storage.Backend.
func RenderCoordinatorPrompt(cfg *config.Config, task *storage.Task) (string, error) {
	criteria := task.AcceptanceCriteria
	if criteria == nil {
		criteria = []string{}
	}

	maxSubtasks := cfg.Execution.Coordinator.MaxSubtasks
	if maxSubtasks <= 0 {
		maxSubtasks = 5
	}

	var taskStoreInstructions string
	if cfg.Storage.Backend == "beads" {
		taskStoreInstructions = fmt.Sprintf(beadsCoordinatorInstructionsFmt, task.ID, task.ID, task.ID)
	} else {
		taskStoreInstructions = jsonlCoordinatorInstructions
	}

	data := CoordinatorPromptData{
		ProjectName:           cfg.Project.Name,
		TechStack:             cfg.Project.TechStack,
		ProjectPath:           cfg.Project.Path,
		TaskID:                task.ID,
		Title:                 task.Title,
		Description:           task.Description,
		AcceptanceCriteria:    criteria,
		TaskStoreInstructions: taskStoreInstructions,
		MaxSubtasks:           maxSubtasks,
	}

	tmpl, err := template.New("coordinator").Parse(CoordinatorPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to parse coordinator prompt template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render coordinator prompt: %w", err)
	}

	return buf.String(), nil
}

// RenderTaskPrompt generates a prompt for executing a specific task
func RenderTaskPrompt(cfg *config.Config, task *storage.Task, attempt int, prevError string) (string, error) {
	// Build acceptance criteria list
	criteria := task.AcceptanceCriteria
	if criteria == nil {
		criteria = []string{}
	}

	// Get previous error if provided
	previousError := strings.TrimSpace(prevError)

	data := TaskPromptData{
		ProjectName:         cfg.Project.Name,
		TechStack:           cfg.Project.TechStack,
		ProjectPath:         cfg.Project.Path,
		MainBranch:          cfg.Project.MainBranch,
		TaskID:              task.ID,
		Title:               task.Title,
		Complexity:          task.Complexity,
		Attempt:             attempt,
		MaxAttempts:         task.Metadata.MaxAttempts,
		Description:         task.Description,
		AcceptanceCriteria:  criteria,
		VerificationCommand: cfg.Verification.Command,
		PreviousError:       previousError,
		CustomInstructions:  cfg.Prompts.CustomInstructions,
	}

	tmpl, err := template.New("task").Parse(TaskExecutionPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to parse task prompt template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render task prompt: %w", err)
	}

	return buf.String(), nil
}
