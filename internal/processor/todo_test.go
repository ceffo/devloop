package processor

import (
	"os"
	"path/filepath"
	"testing"
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
			name: "empty file",
			content: ``,
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
