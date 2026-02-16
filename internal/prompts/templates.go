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

## Referenced Files
{{if .PRDPath}}- PRD: {{.PRDPath}}{{end}}
{{if .TasksPath}}- Tasks: {{.TasksPath}}{{end}}
{{if .TodoPath}}- TODO: {{.TodoPath}}{{end}}

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
	PRDPath             string
	TasksPath           string
	TodoPath            string
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
		PRDPath:             cfg.Files.PRD,
		TasksPath:           cfg.Files.Tasks,
		TodoPath:            cfg.Files.Todo,
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
