package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/processor"
	"github.com/ceffo/devloop/internal/storage"
	"github.com/ceffo/devloop/internal/ui"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// TodoCmd returns the todo command with subcommands
func TodoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "todo",
		Short: "Process TODO items into tasks",
		Long: `Process TODO markdown files into structured tasks using AI agents.

The AI agent analyzes TODO items and generates executable tasks with:
  - Hierarchical IDs
  - Complexity assessment
  - Model selection
  - Dependencies
  - Acceptance criteria

Example:
  devloop todo process .todo/TODO.md
  devloop todo process .todo/TODO.md --review`,
	}

	// Add subcommands
	cmd.AddCommand(todoProcessCmd())

	return cmd
}

// todoProcessCmd returns the todo process subcommand
func todoProcessCmd() *cobra.Command {
	var (
		reviewFlag bool
		waveFlag   int
	)

	cmd := &cobra.Command{
		Use:   "process FILE",
		Short: "Process TODO file into structured tasks",
		Long: `Parse a TODO markdown file and convert items into structured tasks using AI.

The AI agent will:
  1. Parse TODO items from the markdown file
  2. Group related items into logical tasks
  3. Assign hierarchical task IDs
  4. Determine complexity and model selection
  5. Generate acceptance criteria
  6. Identify dependencies

Flags:
  --review    Display generated tasks and prompt for confirmation before saving
  --wave N    Assign tasks to specific wave (default: auto-detect next wave)

Examples:
  devloop todo process .todo/TODO.md
  devloop todo process .todo/TODO.md --review
  devloop todo process .todo/TODO.md --wave 2`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			todoFilePath := args[0]

			// Get config path from persistent flags
			configPath, _ := cmd.Flags().GetString("config")

			// Load configuration
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Validate config
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}

			// Parse TODO file
			fmt.Printf("Parsing TODO file: %s\n", todoFilePath)
			todos, err := processor.ParseTodoFile(todoFilePath)
			if err != nil {
				return fmt.Errorf("failed to parse TODO file: %w", err)
			}

			if len(todos) == 0 {
				fmt.Println("No TODO items found in file.")
				return nil
			}

			fmt.Printf("Found %d TODO item(s)\n\n", len(todos))

			// Call ProcessTodoItems
			fmt.Println("Processing TODO items with AI agent...")
			tasks, err := processor.ProcessTodoItems(cfg, todos, reviewFlag)
			if err != nil {
				return fmt.Errorf("failed to process TODO items: %w", err)
			}

			if len(tasks) == 0 {
				fmt.Println("No tasks generated.")
				return nil
			}

			// Override wave if specified
			if waveFlag > 0 {
				for _, task := range tasks {
					task.Wave = waveFlag
					// Update task ID to reflect new wave
					// Extract task number from current ID (e.g., "1.2" -> "2")
					parts := splitTaskID(task.ID)
					if len(parts) == 2 {
						task.ID = fmt.Sprintf("%d.%s", waveFlag, parts[1])
					}
				}
				fmt.Printf("Assigned tasks to wave %d\n\n", waveFlag)
			}

			// Display summary table of generated tasks
			displayTasksSummary(tasks)

			// Save tasks to JSONL if approved (review mode already handled in ProcessTodoItems)
			if !reviewFlag {
				// Not in review mode, ask for confirmation here
				if !confirmSave(len(tasks)) {
					fmt.Println("Task processing cancelled.")
					return nil
				}
			}

			// Save tasks to storage
			store := storage.NewStorage(cfg)
			savedCount := 0
			for _, task := range tasks {
				if err := store.SaveTask(task); err != nil {
					return fmt.Errorf("failed to save task %s: %w", task.ID, err)
				}
				savedCount++
			}

			// Success message
			tasksFilePath := filepath.Join(cfg.Project.Path, ".devloop", "tasks.jsonl")
			fmt.Printf("\n%s\n", ui.Success(fmt.Sprintf("Successfully saved %d task(s) to %s", savedCount, tasksFilePath)))

			return nil
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&reviewFlag, "review", false, "show tasks and confirm before saving")
	cmd.Flags().IntVar(&waveFlag, "wave", 0, "assign to specific wave (default: auto-detect next wave)")

	return cmd
}

// displayTasksSummary shows a summary table of generated tasks
func displayTasksSummary(tasks []*storage.Task) {
	fmt.Println("=== Generated Tasks Summary ===")
	fmt.Println()

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("ID", "Title", "Complexity", "Dependencies")

	for _, task := range tasks {
		dependencies := "-"
		if len(task.BlockedBy) > 0 {
			dependencies = fmt.Sprintf("%v", task.BlockedBy)
		}

		table.Append(
			task.ID,
			truncateString(task.Title, 40),
			task.Complexity,
			dependencies,
		)
	}

	if err := table.Render(); err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering table: %v\n", err)
	}
	fmt.Println()
}

// confirmSave prompts the user to confirm saving tasks
func confirmSave(count int) bool {
	fmt.Printf("Save %d task(s) to storage? [y/N]: ", count)

	var response string
	fmt.Scanln(&response)

	response = toLower(trim(response))
	return response == "y" || response == "yes"
}

// splitTaskID splits a task ID into wave and task number parts
func splitTaskID(id string) []string {
	parts := []string{}
	current := ""
	for _, ch := range id {
		if ch == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// truncateString truncates a string to the specified length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// toLower returns the lowercase version of a string
func toLower(s string) string {
	result := ""
	for _, ch := range s {
		if ch >= 'A' && ch <= 'Z' {
			result += string(ch + 32)
		} else {
			result += string(ch)
		}
	}
	return result
}

// trim removes leading and trailing whitespace from a string
func trim(s string) string {
	start := 0
	end := len(s)

	// Trim leading whitespace
	for start < end && isWhitespace(rune(s[start])) {
		start++
	}

	// Trim trailing whitespace
	for end > start && isWhitespace(rune(s[end-1])) {
		end--
	}

	return s[start:end]
}

// isWhitespace checks if a rune is whitespace
func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}
