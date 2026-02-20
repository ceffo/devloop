package prompts

import (
	"strings"
	"testing"

	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/processor"
	"github.com/ceffo/devloop/internal/storage"
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

func TestRenderCoordinatorPrompt(t *testing.T) {
	tests := []struct {
		name           string
		cfg            *config.Config
		task           *storage.Task
		expectContains []string
		expectError    bool
	}{
		{
			name: "jsonl backend",
			cfg: &config.Config{
				Project: config.ProjectConfig{
					Name:      "myproject",
					Path:      "/my/project",
					TechStack: "Go 1.21+",
				},
				Storage: config.StorageConfig{
					Backend: "jsonl",
				},
				Execution: config.ExecutionConfig{
					Coordinator: config.CoordinatorConfig{
						MaxSubtasks: 4,
					},
				},
			},
			task: &storage.Task{
				ID:          "DEV-10",
				Title:       "Implement storage abstraction",
				Description: "Refactor storage layer to support multiple backends",
				AcceptanceCriteria: []string{
					"TaskStore interface defined",
					"JSONLStore implements interface",
					"All tests pass",
				},
			},
			expectContains: []string{
				"myproject",
				"Go 1.21+",
				"/my/project",
				"DEV-10",
				"Implement storage abstraction",
				"Refactor storage layer",
				"TaskStore interface defined",
				"JSONLStore implements interface",
				"All tests pass",
				"4",
				"coordinator-output.json",
				SentinelProceed,
				SentinelDecomposed,
			},
			expectError: false,
		},
		{
			name: "beads backend",
			cfg: &config.Config{
				Project: config.ProjectConfig{
					Name:      "beadsproject",
					Path:      "/beads/project",
					TechStack: "Go + Beads",
				},
				Storage: config.StorageConfig{
					Backend: "beads",
				},
				Execution: config.ExecutionConfig{
					Coordinator: config.CoordinatorConfig{
						MaxSubtasks: 3,
					},
				},
			},
			task: &storage.Task{
				ID:          "BD-5",
				Title:       "Complex beads task",
				Description: "A complex task requiring beads",
				AcceptanceCriteria: []string{
					"Beads integration works",
				},
			},
			expectContains: []string{
				"beadsproject",
				"Go + Beads",
				"/beads/project",
				"BD-5",
				"Complex beads task",
				"3",
				"bd update BD-5 --claim",
				"--parent BD-5",
				SentinelProceed,
				SentinelDecomposed,
			},
			expectError: false,
		},
		{
			name: "default max subtasks when zero",
			cfg: &config.Config{
				Project: config.ProjectConfig{
					Name:      "proj",
					TechStack: "Go",
					Path:      "/proj",
				},
				Storage: config.StorageConfig{Backend: "jsonl"},
				Execution: config.ExecutionConfig{
					Coordinator: config.CoordinatorConfig{MaxSubtasks: 0},
				},
			},
			task: &storage.Task{
				ID:          "DEV-1",
				Title:       "Task",
				Description: "Desc",
			},
			expectContains: []string{
				"5", // default max subtasks
			},
			expectError: false,
		},
		{
			name: "nil acceptance criteria",
			cfg: &config.Config{
				Project: config.ProjectConfig{
					Name:      "proj",
					TechStack: "Go",
					Path:      "/proj",
				},
				Storage:   config.StorageConfig{Backend: "jsonl"},
				Execution: config.ExecutionConfig{Coordinator: config.CoordinatorConfig{MaxSubtasks: 5}},
			},
			task: &storage.Task{
				ID:                 "DEV-2",
				Title:              "Task with no criteria",
				Description:        "Desc",
				AcceptanceCriteria: nil,
			},
			expectContains: []string{
				"DEV-2",
				"Task with no criteria",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderCoordinatorPrompt(tt.cfg, tt.task)

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

				// Ensure no template delimiters leak through
				if strings.Contains(result, "{{") || strings.Contains(result, "}}") {
					t.Errorf("output contains unrendered template delimiters:\n%s", result)
				}
			}
		})
	}
}

func TestRenderCoordinatorPromptBackendInstructions(t *testing.T) {
	baseTask := &storage.Task{
		ID:          "DEV-99",
		Title:       "Test task",
		Description: "Test description",
	}

	t.Run("jsonl instructions do not contain bd commands", func(t *testing.T) {
		cfg := &config.Config{
			Storage: config.StorageConfig{Backend: "jsonl"},
			Execution: config.ExecutionConfig{
				Coordinator: config.CoordinatorConfig{MaxSubtasks: 5},
			},
		}
		result, err := RenderCoordinatorPrompt(cfg, baseTask)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(result, "bd update") || strings.Contains(result, "bd create") {
			t.Errorf("JSONL backend should not include bd commands")
		}
		if !strings.Contains(result, "coordinator-output.json") {
			t.Errorf("JSONL backend should mention coordinator-output.json")
		}
	})

	t.Run("beads instructions contain task ID in commands", func(t *testing.T) {
		cfg := &config.Config{
			Storage: config.StorageConfig{Backend: "beads"},
			Execution: config.ExecutionConfig{
				Coordinator: config.CoordinatorConfig{MaxSubtasks: 5},
			},
		}
		result, err := RenderCoordinatorPrompt(cfg, baseTask)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "bd update DEV-99 --claim") {
			t.Errorf("Beads backend should include bd update with task ID")
		}
		if !strings.Contains(result, "--parent DEV-99") {
			t.Errorf("Beads backend should include bd create with --parent and task ID")
		}
		if strings.Contains(result, "coordinator-output.json") {
			t.Errorf("Beads backend should not mention coordinator-output.json")
		}
	})
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
