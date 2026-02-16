package prompts

import (
	"strings"
	"testing"

	"github.com/yourusername/devloop/internal/config"
	"github.com/yourusername/devloop/internal/processor"
	"github.com/yourusername/devloop/internal/storage"
)

func TestRenderTodoPrompt(t *testing.T) {
	tests := []struct {
		name           string
		project        config.ProjectConfig
		todos          []processor.TodoItem
		nextID         string
		expectContains []string
		expectError    bool
	}{
		{
			name: "basic todo prompt",
			project: config.ProjectConfig{
				Name:      "testproject",
				Path:      "/path/to/project",
				TechStack: "Go + PostgreSQL",
			},
			todos: []processor.TodoItem{
				{
					ID:       "TODO-1",
					Category: "Features",
					Content:  "Add user authentication",
					Priority: "high",
				},
				{
					ID:       "TODO-2",
					Category: "Bugs",
					Content:  "Fix login redirect",
					Priority: "medium",
				},
			},
			nextID: "1.1",
			expectContains: []string{
				"testproject",
				"Go + PostgreSQL",
				"/path/to/project",
				"TODO-1",
				"TODO-2",
				"Add user authentication",
				"Fix login redirect",
				"high",
				"medium",
				"1.1",
			},
			expectError: false,
		},
		{
			name: "empty todos",
			project: config.ProjectConfig{
				Name:      "emptyproject",
				Path:      "/empty",
				TechStack: "Node.js",
			},
			todos:  []processor.TodoItem{},
			nextID: "2.1",
			expectContains: []string{
				"emptyproject",
				"Node.js",
				"2.1",
			},
			expectError: false,
		},
		{
			name: "nil todos",
			project: config.ProjectConfig{
				Name:      "nilproject",
				Path:      "/nil",
				TechStack: "Python",
			},
			todos:  nil,
			nextID: "3.1",
			expectContains: []string{
				"nilproject",
				"Python",
				"3.1",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderTodoPrompt(tt.project, tt.todos, tt.nextID)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				for _, expected := range tt.expectContains {
					if !strings.Contains(result, expected) {
						t.Errorf("expected prompt to contain %q, but it didn't", expected)
					}
				}
			}
		})
	}
}

func TestRenderTaskPrompt(t *testing.T) {
	tests := []struct {
		name           string
		cfg            *config.Config
		task           *storage.Task
		attempt        int
		prevError      string
		expectContains []string
		expectError    bool
	}{
		{
			name: "basic task prompt without error",
			cfg: &config.Config{
				Project: config.ProjectConfig{
					Name:       "myproject",
					Path:       "/my/project",
					TechStack:  "React + TypeScript",
					MainBranch: "main",
				},
				Verification: config.VerificationConfig{
					Command: "npm test",
				},
				Files: config.FilesConfig{
					PRD:   "docs/PRD.md",
					Tasks: "docs/TASKS.md",
					Todo:  ".todo/TODO.md",
				},
				Prompts: config.PromptsConfig{
					CustomInstructions: "",
				},
			},
			task: &storage.Task{
				ID:          "1.1",
				Title:       "Implement user login",
				Complexity:  "moderate",
				Description: "Create a login form with validation",
				AcceptanceCriteria: []string{
					"Form renders correctly",
					"Validation works",
					"Submission calls API",
				},
				Metadata: storage.TaskMetadata{
					MaxAttempts: 2,
				},
			},
			attempt:   1,
			prevError: "",
			expectContains: []string{
				"myproject",
				"React + TypeScript",
				"/my/project",
				"main",
				"1.1",
				"Implement user login",
				"moderate",
				"Create a login form with validation",
				"Form renders correctly",
				"Validation works",
				"Submission calls API",
				"npm test",
				"Attempt: 1 of 2",
				"docs/PRD.md",
				"docs/TASKS.md",
			},
			expectError: false,
		},
		{
			name: "task prompt with previous error",
			cfg: &config.Config{
				Project: config.ProjectConfig{
					Name:       "errorproject",
					Path:       "/error/project",
					TechStack:  "Go",
					MainBranch: "develop",
				},
				Verification: config.VerificationConfig{
					Command: "go test ./...",
				},
				Files: config.FilesConfig{
					PRD:   "PRD.md",
					Tasks: "TASKS.md",
					Todo:  "TODO.md",
				},
				Prompts: config.PromptsConfig{
					CustomInstructions: "",
				},
			},
			task: &storage.Task{
				ID:          "2.3",
				Title:       "Fix bug in parser",
				Complexity:  "simple",
				Description: "The parser crashes on empty input",
				AcceptanceCriteria: []string{
					"Handles empty input gracefully",
					"Returns appropriate error",
				},
				Metadata: storage.TaskMetadata{
					MaxAttempts: 3,
				},
			},
			attempt:   2,
			prevError: "panic: runtime error: index out of range",
			expectContains: []string{
				"errorproject",
				"Go",
				"2.3",
				"Fix bug in parser",
				"Attempt: 2 of 3",
				"panic: runtime error: index out of range",
				"Previous Attempt Error",
				"go test ./...",
			},
			expectError: false,
		},
		{
			name: "task with custom instructions",
			cfg: &config.Config{
				Project: config.ProjectConfig{
					Name:       "customproject",
					Path:       "/custom",
					TechStack:  "Python",
					MainBranch: "main",
				},
				Verification: config.VerificationConfig{
					Command: "pytest",
				},
				Files: config.FilesConfig{},
				Prompts: config.PromptsConfig{
					CustomInstructions: "Always use type hints and docstrings",
				},
			},
			task: &storage.Task{
				ID:          "1.1",
				Title:       "Add feature",
				Complexity:  "moderate",
				Description: "New feature",
				Metadata: storage.TaskMetadata{
					MaxAttempts: 2,
				},
			},
			attempt:   1,
			prevError: "",
			expectContains: []string{
				"customproject",
				"Always use type hints and docstrings",
				"Custom Instructions",
			},
			expectError: false,
		},
		{
			name: "task with nil acceptance criteria",
			cfg: &config.Config{
				Project: config.ProjectConfig{
					Name:       "nilcriteria",
					Path:       "/nil",
					TechStack:  "Ruby",
					MainBranch: "main",
				},
				Verification: config.VerificationConfig{
					Command: "rspec",
				},
				Files: config.FilesConfig{},
				Prompts: config.PromptsConfig{},
			},
			task: &storage.Task{
				ID:                 "1.1",
				Title:              "Test task",
				Complexity:         "simple",
				Description:        "A test",
				AcceptanceCriteria: nil, // nil criteria
				Metadata: storage.TaskMetadata{
					MaxAttempts: 1,
				},
			},
			attempt:   1,
			prevError: "",
			expectContains: []string{
				"nilcriteria",
				"Test task",
				"Acceptance Criteria",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderTaskPrompt(tt.cfg, tt.task, tt.attempt, tt.prevError)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				for _, expected := range tt.expectContains {
					if !strings.Contains(result, expected) {
						t.Errorf("expected prompt to contain %q, but it didn't.\nGot:\n%s", expected, result)
					}
				}

				// Check that previous error section only appears when there's an error
				if tt.prevError == "" && strings.Contains(result, "Previous Attempt Error") {
					t.Errorf("prompt should not contain 'Previous Attempt Error' when prevError is empty")
				}

				// Check that custom instructions only appear when set
				if tt.cfg.Prompts.CustomInstructions == "" && strings.Contains(result, "Custom Instructions") {
					t.Errorf("prompt should not contain 'Custom Instructions' when not set")
				}
			}
		})
	}
}

func TestRenderTodoPromptOutput(t *testing.T) {
	// Test that output is valid and doesn't have template errors
	project := config.ProjectConfig{
		Name:      "test",
		Path:      "/test",
		TechStack: "Go",
	}
	todos := []processor.TodoItem{
		{ID: "TODO-1", Category: "Test", Content: "Test item", Priority: "low"},
	}

	result, err := RenderTodoPrompt(project, todos, "1.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that template delimiters are not in output (would indicate parse error)
	if strings.Contains(result, "{{") || strings.Contains(result, "}}") {
		t.Errorf("output contains template delimiters, indicating incomplete rendering:\n%s", result)
	}

	// Check basic structure
	if !strings.Contains(result, "Project Context") {
		t.Errorf("output missing 'Project Context' section")
	}
	if !strings.Contains(result, "TODO Items") {
		t.Errorf("output missing 'TODO Items' section")
	}
	if !strings.Contains(result, "Instructions") {
		t.Errorf("output missing 'Instructions' section")
	}
}

func TestRenderTaskPromptOutput(t *testing.T) {
	// Test that output is valid and doesn't have template errors
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Name:       "test",
			Path:       "/test",
			TechStack:  "Go",
			MainBranch: "main",
		},
		Verification: config.VerificationConfig{
			Command: "go test",
		},
		Files:   config.FilesConfig{},
		Prompts: config.PromptsConfig{},
	}
	task := &storage.Task{
		ID:          "1.1",
		Title:       "Test task",
		Complexity:  "simple",
		Description: "Test description",
		AcceptanceCriteria: []string{
			"Criterion 1",
			"Criterion 2",
		},
		Metadata: storage.TaskMetadata{
			MaxAttempts: 2,
		},
	}

	result, err := RenderTaskPrompt(cfg, task, 1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that template delimiters are not in output
	if strings.Contains(result, "{{") || strings.Contains(result, "}}") {
		t.Errorf("output contains template delimiters, indicating incomplete rendering:\n%s", result)
	}

	// Check basic structure
	if !strings.Contains(result, "Project Context") {
		t.Errorf("output missing 'Project Context' section")
	}
	if !strings.Contains(result, "Task Details") {
		t.Errorf("output missing 'Task Details' section")
	}
	if !strings.Contains(result, "Description") {
		t.Errorf("output missing 'Description' section")
	}
	if !strings.Contains(result, "Acceptance Criteria") {
		t.Errorf("output missing 'Acceptance Criteria' section")
	}
	if !strings.Contains(result, "Verification") {
		t.Errorf("output missing 'Verification' section")
	}
}
