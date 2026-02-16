package processor

import (
	"fmt"
	"os"
	"path/filepath"
)

// ExampleParseTodoFile demonstrates the TODO parser functionality.
func ExampleParseTodoFile() {
	// Create a temporary TODO file for demonstration
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "example-todo.md")

	content := `# Features
- [ ] !! High priority feature
- ! Medium priority task
- Regular task
- [x] Completed task (will be skipped)

## Bug Fixes
- Fix login issue
- [ ] ! Fix validation bug`

	os.WriteFile(tmpFile, []byte(content), 0644)
	defer os.Remove(tmpFile)

	// Parse the file
	items, err := ParseTodoFile(tmpFile)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Display results
	for _, item := range items {
		fmt.Printf("%s [%s] %s: %s\n",
			item.ID, item.Priority, item.Category, item.Content)
	}

	// Output:
	// TODO-1 [high] Features: High priority feature
	// TODO-2 [medium] Features: Medium priority task
	// TODO-3 [low] Features: Regular task
	// TODO-4 [low] Bug Fixes: Fix login issue
	// TODO-5 [medium] Bug Fixes: Fix validation bug
}
