package processor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/storage"
)

func TestParseTodoFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []TodoItem
		wantErr  bool
	}{
		{
			name: "simple list items",
			content: `# Category 1
- Item 1
- Item 2
- Item 3`,
			expected: []TodoItem{
				{ID: "TODO-1", Category: "Category 1", Content: "Item 1", Priority: "low"},
				{ID: "TODO-2", Category: "Category 1", Content: "Item 2", Priority: "low"},
				{ID: "TODO-3", Category: "Category 1", Content: "Item 3", Priority: "low"},
			},
		},
		{
			name: "priority markers",
			content: `# Tasks
- !! High priority item
- ! Medium priority item
- Low priority item`,
			expected: []TodoItem{
				{ID: "TODO-1", Category: "Tasks", Content: "High priority item", Priority: "high"},
				{ID: "TODO-2", Category: "Tasks", Content: "Medium priority item", Priority: "medium"},
				{ID: "TODO-3", Category: "Tasks", Content: "Low priority item", Priority: "low"},
			},
		},
		{
			name: "multiple priority exclamation marks",
			content: `# Urgent
- !!! Very high priority
- !! High priority`,
			expected: []TodoItem{
				{ID: "TODO-1", Category: "Urgent", Content: "Very high priority", Priority: "high"},
				{ID: "TODO-2", Category: "Urgent", Content: "High priority", Priority: "high"},
			},
		},
		{
			name: "checkboxes - unchecked only",
			content: `# Checklist
- [ ] Todo item 1
- [x] Completed item
- [ ] Todo item 2
- [X] Another completed`,
			expected: []TodoItem{
				{ID: "TODO-1", Category: "Checklist", Content: "Todo item 1", Priority: "low"},
				{ID: "TODO-2", Category: "Checklist", Content: "Todo item 2", Priority: "low"},
			},
		},
		{
			name: "multiple categories",
			content: `# Category 1
- Item 1
- Item 2

## Category 2
- Item 3

### Category 3
- Item 4`,
			expected: []TodoItem{
				{ID: "TODO-1", Category: "Category 1", Content: "Item 1", Priority: "low"},
				{ID: "TODO-2", Category: "Category 1", Content: "Item 2", Priority: "low"},
				{ID: "TODO-3", Category: "Category 2", Content: "Item 3", Priority: "low"},
				{ID: "TODO-4", Category: "Category 3", Content: "Item 4", Priority: "low"},
			},
		},
		{
			name: "asterisk list markers",
			content: `# Tasks
* Task with asterisk
* Another task`,
			expected: []TodoItem{
				{ID: "TODO-1", Category: "Tasks", Content: "Task with asterisk", Priority: "low"},
				{ID: "TODO-2", Category: "Tasks", Content: "Another task", Priority: "low"},
			},
		},
		{
			name: "mixed indentation",
			content: `# Main
- Level 1
  - Level 2 (nested)
    - Level 3 (nested)`,
			expected: []TodoItem{
				{ID: "TODO-1", Category: "Main", Content: "Level 1", Priority: "low"},
				{ID: "TODO-2", Category: "Main", Content: "Level 2 (nested)", Priority: "low"},
				{ID: "TODO-3", Category: "Main", Content: "Level 3 (nested)", Priority: "low"},
			},
		},
		{
			name: "empty lines and no category",
			content: `
- Item without category


- Another item

# With Category
- Categorized item
`,
			expected: []TodoItem{
				{ID: "TODO-1", Category: "", Content: "Item without category", Priority: "low"},
				{ID: "TODO-2", Category: "", Content: "Another item", Priority: "low"},
				{ID: "TODO-3", Category: "With Category", Content: "Categorized item", Priority: "low"},
			},
		},
		{
			name: "priorities with checkboxes",
			content: `# Tasks
- [ ] ! Medium priority unchecked
- [x] !! High priority completed (should skip)
- [ ] !! High priority unchecked`,
			expected: []TodoItem{
				{ID: "TODO-1", Category: "Tasks", Content: "Medium priority unchecked", Priority: "medium"},
				{ID: "TODO-2", Category: "Tasks", Content: "High priority unchecked", Priority: "high"},
			},
		},
		{
			name:     "empty file",
			content:  ``,
			expected: []TodoItem{},
		},
		{
			name: "only headings",
			content: `# Category 1
## Category 2
### Category 3`,
			expected: []TodoItem{},
		},
		{
			name: "only completed items",
			content: `# Tasks
- [x] Completed 1
- [x] Completed 2`,
			expected: []TodoItem{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "TODO.md")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			// Parse the file
			got, err := ParseTodoFile(tmpFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTodoFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Compare results
			if len(got) != len(tt.expected) {
				t.Errorf("ParseTodoFile() got %d items, want %d items", len(got), len(tt.expected))
				t.Logf("Got: %+v", got)
				t.Logf("Want: %+v", tt.expected)
				return
			}

			for i, item := range got {
				expected := tt.expected[i]
				if item.ID != expected.ID {
					t.Errorf("Item %d: ID = %q, want %q", i, item.ID, expected.ID)
				}
				if item.Category != expected.Category {
					t.Errorf("Item %d: Category = %q, want %q", i, item.Category, expected.Category)
				}
				if item.Content != expected.Content {
					t.Errorf("Item %d: Content = %q, want %q", i, item.Content, expected.Content)
				}
				if item.Priority != expected.Priority {
					t.Errorf("Item %d: Priority = %q, want %q", i, item.Priority, expected.Priority)
				}
			}
		})
	}
}

func TestParseTodoFile_FileErrors(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		_, err := ParseTodoFile("/nonexistent/path/TODO.md")
		if err == nil {
			t.Error("ParseTodoFile() expected error for non-existent file, got nil")
		}
	})

	t.Run("directory instead of file", func(t *testing.T) {
		tmpDir := t.TempDir()
		_, err := ParseTodoFile(tmpDir)
		if err == nil {
			t.Error("ParseTodoFile() expected error for directory, got nil")
		}
	})
}

func TestParseTodoFile_ComplexExample(t *testing.T) {
	content := `# Wave 1: Core Infrastructure

## Configuration
- [ ] !! Implement configuration loading
- [ ] ! Add validation logic
- [x] Write tests (completed, should skip)

## Storage
- Task data structures
- ! JSONL operations
- [ ] Query system

# Wave 2: CLI Commands

- [ ] !! Setup cobra framework
- [ ] Implement init command
- [ ] ! Add config commands
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "TODO.md")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	items, err := ParseTodoFile(tmpFile)
	if err != nil {
		t.Fatalf("ParseTodoFile() error = %v", err)
	}

	expected := []TodoItem{
		{ID: "TODO-1", Category: "Configuration", Content: "Implement configuration loading", Priority: "high"},
		{ID: "TODO-2", Category: "Configuration", Content: "Add validation logic", Priority: "medium"},
		{ID: "TODO-3", Category: "Storage", Content: "Task data structures", Priority: "low"},
		{ID: "TODO-4", Category: "Storage", Content: "JSONL operations", Priority: "medium"},
		{ID: "TODO-5", Category: "Storage", Content: "Query system", Priority: "low"},
		{ID: "TODO-6", Category: "Wave 2: CLI Commands", Content: "Setup cobra framework", Priority: "high"},
		{ID: "TODO-7", Category: "Wave 2: CLI Commands", Content: "Implement init command", Priority: "low"},
		{ID: "TODO-8", Category: "Wave 2: CLI Commands", Content: "Add config commands", Priority: "medium"},
	}

	if len(items) != len(expected) {
		t.Fatalf("got %d items, want %d items", len(items), len(expected))
	}

	for i, item := range items {
		exp := expected[i]
		if item.ID != exp.ID || item.Category != exp.Category ||
			item.Content != exp.Content || item.Priority != exp.Priority {
			t.Errorf("Item %d mismatch:\ngot:  %+v\nwant: %+v", i, item, exp)
		}
	}
}

func TestGenerateNextTaskID(t *testing.T) {
	tests := []struct {
		name     string
		existing []storage.Task
		expected string
	}{
		{
			name:     "no existing tasks",
			existing: []storage.Task{},
			expected: "1.1",
		},
		{
			name: "single task in wave 1",
			existing: []storage.Task{
				{ID: "1.1"},
			},
			expected: "1.2",
		},
		{
			name: "multiple tasks in wave 1",
			existing: []storage.Task{
				{ID: "1.1"},
				{ID: "1.2"},
				{ID: "1.3"},
			},
			expected: "1.4",
		},
		{
			name: "multiple waves",
			existing: []storage.Task{
				{ID: "1.1"},
				{ID: "1.2"},
				{ID: "2.1"},
				{ID: "2.2"},
			},
			expected: "2.3",
		},
		{
			name: "out of order tasks",
			existing: []storage.Task{
				{ID: "1.3"},
				{ID: "2.1"},
				{ID: "1.1"},
				{ID: "1.2"},
			},
			expected: "2.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory and storage
			tmpDir := t.TempDir()
			cfg := &config.Config{
				Project: config.ProjectConfig{
					Path:         tmpDir,
					TaskIDFormat: "hierarchical", // Test hierarchical format
				},
			}
			store := storage.NewStorage(cfg)

			// Save existing tasks
			for i := range tt.existing {
				if err := store.SaveTask(&tt.existing[i]); err != nil {
					t.Fatalf("failed to save task: %v", err)
				}
			}

			// Generate next ID
			got, err := generateNextTaskID(store, cfg)
			if err != nil {
				t.Fatalf("generateNextTaskID() error = %v", err)
			}

			if got != tt.expected {
				t.Errorf("generateNextTaskID() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParseTasksFromJSON(t *testing.T) {
	cfg := &config.Config{
		CLI: config.CLIConfig{
			Agents: map[string]*config.AgentConfig{
				"test": {
					Tool: "claude",
					Models: map[string]string{
						"simple":   "model-simple",
						"moderate": "model-moderate",
						"complex":  "model-complex",
					},
				},
			},
		},
		Execution: config.ExecutionConfig{
			MaxAttempts: 2,
		},
	}

	todos := []TodoItem{
		{ID: "TODO-1", Content: "Test item"},
	}

	tests := []struct {
		name       string
		output     string
		wantErr    bool
		wantCount  int
		checkFirst func(t *testing.T, task *storage.Task)
	}{
		{
			name: "valid JSON array",
			output: `[
				{
					"id": "1.1",
					"title": "Test task",
					"description": "Test description",
					"complexity": "simple",
					"acceptance_criteria": ["Criterion 1", "Criterion 2"],
					"blocked_by": [],
					"tags": ["test"]
				}
			]`,
			wantErr:   false,
			wantCount: 1,
			checkFirst: func(t *testing.T, task *storage.Task) {
				if task.ID != "1.1" {
					t.Errorf("ID = %q, want %q", task.ID, "1.1")
				}
				if task.Title != "Test task" {
					t.Errorf("Title = %q, want %q", task.Title, "Test task")
				}
				if task.Complexity != "simple" {
					t.Errorf("Complexity = %q, want %q", task.Complexity, "simple")
				}
				if task.Status != "pending" {
					t.Errorf("Status = %q, want %q", task.Status, "pending")
				}
				if len(task.AcceptanceCriteria) != 2 {
					t.Errorf("len(AcceptanceCriteria) = %d, want %d", len(task.AcceptanceCriteria), 2)
				}
			},
		},
		{
			name: "JSON with surrounding text",
			output: `Here are the tasks:
			[
				{
					"id": "2.1",
					"title": "Another task",
					"description": "Description",
					"complexity": "moderate",
					"acceptance_criteria": ["Test"],
					"blocked_by": ["1.1"],
					"tags": []
				}
			]
			That's the end.`,
			wantErr:   false,
			wantCount: 1,
			checkFirst: func(t *testing.T, task *storage.Task) {
				if len(task.BlockedBy) != 1 || task.BlockedBy[0] != "1.1" {
					t.Errorf("BlockedBy = %v, want [\"1.1\"]", task.BlockedBy)
				}
			},
		},
		{
			name:      "no JSON array",
			output:    "This is just text without JSON",
			wantErr:   true,
			wantCount: 0,
		},
		{
			name:      "invalid JSON",
			output:    "[{invalid json}]",
			wantErr:   true,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks, err := parseTasksFromJSON(tt.output, cfg, todos)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTasksFromJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(tasks) != tt.wantCount {
					t.Errorf("got %d tasks, want %d", len(tasks), tt.wantCount)
					return
				}

				if tt.wantCount > 0 && tt.checkFirst != nil {
					tt.checkFirst(t, tasks[0])
				}
			}
		})
	}
}

func TestRenderTodoPrompt(t *testing.T) {
	project := config.ProjectConfig{
		Name:      "TestProject",
		TechStack: "Go 1.21",
		Path:      "/test/path",
	}

	todos := []TodoItem{
		{ID: "TODO-1", Category: "Features", Content: "Add feature X", Priority: "high"},
		{ID: "TODO-2", Category: "Bugs", Content: "Fix bug Y", Priority: "medium"},
	}

	prompt, err := renderTodoPrompt(project, todos, "1.1")
	if err != nil {
		t.Fatalf("renderTodoPrompt() error = %v", err)
	}

	// Check that prompt contains expected elements
	expectedElements := []string{
		"TestProject",
		"Go 1.21",
		"/test/path",
		"TODO-1",
		"TODO-2",
		"Features",
		"Bugs",
		"Add feature X",
		"Fix bug Y",
		"1.1",
		"high",
		"medium",
	}

	for _, elem := range expectedElements {
		if !contains(prompt, elem) {
			t.Errorf("renderTodoPrompt() output missing %q", elem)
		}
	}
}

func TestTaskJSON_Unmarshal(t *testing.T) {
	jsonData := `{
		"id": "1.1",
		"title": "Test",
		"description": "Desc",
		"complexity": "simple",
		"acceptance_criteria": ["a", "b"],
		"blocked_by": ["1.0"],
		"tags": ["tag1"]
	}`

	var task taskJSON
	if err := json.Unmarshal([]byte(jsonData), &task); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if task.ID != "1.1" {
		t.Errorf("ID = %q, want %q", task.ID, "1.1")
	}
	if task.Title != "Test" {
		t.Errorf("Title = %q, want %q", task.Title, "Test")
	}
	if len(task.AcceptanceCriteria) != 2 {
		t.Errorf("len(AcceptanceCriteria) = %d, want 2", len(task.AcceptanceCriteria))
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && hasSubstring(s, substr))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
