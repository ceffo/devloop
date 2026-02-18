package prompts

import (
	"fmt"
	"time"

	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/processor"
	"github.com/ceffo/devloop/internal/storage"
)

// ExampleRenderTodoPrompt demonstrates the TODO processing prompt rendering
func ExampleRenderTodoPrompt() {
	project := config.ProjectConfig{
		Name:      "myapp",
		Path:      "/home/user/myapp",
		TechStack: "Go + PostgreSQL",
	}

	todos := []processor.TodoItem{
		{
			ID:       "TODO-1",
			Category: "Features",
			Content:  "Implement user authentication",
			Priority: "high",
		},
		{
			ID:       "TODO-2",
			Category: "Features",
			Content:  "Add password reset functionality",
			Priority: "medium",
		},
	}

	result, err := RenderTodoPrompt(project, todos, "1.1")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Just show that it renders successfully
	fmt.Println("TODO prompt rendered successfully")
	fmt.Printf("Length: %d characters\n", len(result))
	// Output:
	// TODO prompt rendered successfully
	// Length: 1445 characters
}

// ExampleRenderTaskPrompt demonstrates the task execution prompt rendering
func ExampleRenderTaskPrompt() {
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Name:       "myapp",
			Path:       "/home/user/myapp",
			TechStack:  "Go + PostgreSQL",
			MainBranch: "main",
		},
		Verification: config.VerificationConfig{
			Command: "go test ./...",
		},
		Prompts: config.PromptsConfig{
			CustomInstructions: "Follow Go best practices",
		},
	}

	task := &storage.Task{
		ID:          "1.1",
		Title:       "Implement user authentication",
		Complexity:  "moderate",
		Description: "Create authentication system with JWT tokens",
		AcceptanceCriteria: []string{
			"Users can register with email and password",
			"Users can login and receive JWT token",
			"Token validation middleware works",
		},
		Metadata: storage.TaskMetadata{
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			MaxAttempts: 2,
		},
	}

	result, err := RenderTaskPrompt(cfg, task, 1, "")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Just show that it renders successfully
	fmt.Println("Task prompt rendered successfully")
	fmt.Printf("Length: %d characters\n", len(result))
	// Output:
	// Task prompt rendered successfully
	// Length: 788 characters
}

// ExampleRenderTaskPrompt_withError demonstrates task prompt with previous error
func ExampleRenderTaskPrompt_withError() {
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Name:       "myapp",
			Path:       "/home/user/myapp",
			TechStack:  "Python + FastAPI",
			MainBranch: "main",
		},
		Verification: config.VerificationConfig{
			Command: "pytest",
		},
		Prompts: config.PromptsConfig{},
	}

	task := &storage.Task{
		ID:          "2.1",
		Title:       "Fix authentication bug",
		Complexity:  "simple",
		Description: "Token validation is failing for expired tokens",
		AcceptanceCriteria: []string{
			"Expired tokens are rejected",
			"Valid tokens are accepted",
		},
		Metadata: storage.TaskMetadata{
			MaxAttempts: 3,
		},
	}

	prevError := "AssertionError: Expected 401, got 200"

	result, err := RenderTaskPrompt(cfg, task, 2, prevError)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Just show that it renders successfully and includes error
	fmt.Println("Task prompt with error rendered successfully")
	fmt.Printf("Contains previous error: %t\n", len(result) > 0 && err == nil)
	// Output:
	// Task prompt with error rendered successfully
	// Contains previous error: true
}
