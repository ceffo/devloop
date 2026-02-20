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

// ProcessCmd returns the process command with subcommands for each input type.
func ProcessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "process",
		Short: "Process input files into tasks",
		Long: `Process input files into structured tasks using AI agents.

Examples:
  devloop process todo .todo/TODO.md
  devloop process prd docs/DESIGN.md --review`,
	}

	cmd.AddCommand(processTodoCmd())
	cmd.AddCommand(processPrdCmd())
	cmd.AddCommand(processTasksCmd())

	return cmd
}

// processTodoCmd returns the 'process todo' subcommand
func processTodoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "todo FILE",
		Short: "Process TODO file into structured tasks",
		Long: `Parse a TODO markdown file and convert items into structured tasks using AI.

The AI agent will:
  1. Parse TODO items from the markdown file
  2. Group related items into logical tasks
  3. Assign task IDs
  4. Determine complexity and model selection
  5. Generate acceptance criteria
  6. Identify dependencies

Examples:
  devloop process todo .todo/TODO.md`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			agentName, _ := cmd.Flags().GetString("agent")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}

			todoFilePath := args[0]

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
			fmt.Println("Processing TODO items with AI agent...")

			tasks, err := processor.ProcessTodoItems(cfg, todos, agentName)
			if err != nil {
				return fmt.Errorf("failed to process TODO items: %w", err)
			}

			if len(tasks) == 0 {
				fmt.Println("No tasks generated.")
				return nil
			}

			displayTasksSummary(tasks)

			if dryRun {
				fmt.Printf("[dry-run] Would save %d task(s) to storage\n", len(tasks))
				fmt.Println("[dry-run] No tasks were saved")
				return nil
			}

			store, err := storage.NewTaskStore(cfg)
			if err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}
			savedCount := 0
			for _, task := range tasks {
				if err := store.SaveTask(task); err != nil {
					return fmt.Errorf("failed to save task %s: %w", task.ID, err)
				}
				savedCount++
			}

			tasksFilePath := filepath.Join(cfg.Project.Path, ".devloop", "tasks.jsonl")
			fmt.Printf("\n%s\n", ui.Success(fmt.Sprintf("Successfully saved %d task(s) to %s", savedCount, tasksFilePath)))
			return nil
		},
	}

	return cmd
}

// processPrdCmd returns the 'process prd' subcommand
func processPrdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prd FILE",
		Short: "Process PRD file into structured tasks",
		Long: `Read a Product Requirements Document and convert it into structured tasks using AI.

The AI agent will:
  1. Analyze the full PRD content
  2. Decompose requirements into atomic, implementable tasks
  3. Assign task IDs and complexity ratings
  4. Generate acceptance criteria
  5. Identify dependencies between tasks

Examples:
  devloop process prd docs/DESIGN.md`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]

			configPath, _ := cmd.Flags().GetString("config")
			agentName, _ := cmd.Flags().GetString("agent")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}

			fmt.Printf("Processing PRD file: %s\n", filePath)
			fmt.Println("Running AI agent to decompose requirements into tasks...")

			tasks, err := processor.ProcessPRD(cfg, filePath, agentName)
			if err != nil {
				return fmt.Errorf("failed to process PRD: %w", err)
			}

			if len(tasks) == 0 {
				fmt.Println("No tasks generated.")
				return nil
			}

			displayTasksSummary(tasks)

			if dryRun {
				fmt.Printf("[dry-run] Would save %d task(s) to storage\n", len(tasks))
				fmt.Println("[dry-run] No tasks were saved")
				return nil
			}

			store, err := storage.NewTaskStore(cfg)
			if err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}
			savedCount := 0
			for _, task := range tasks {
				if err := store.SaveTask(task); err != nil {
					return fmt.Errorf("failed to save task %s: %w", task.ID, err)
				}
				savedCount++
			}

			tasksFilePath := filepath.Join(cfg.Project.Path, ".devloop", "tasks.jsonl")
			fmt.Printf("\n%s\n", ui.Success(fmt.Sprintf("Successfully saved %d task(s) to %s", savedCount, tasksFilePath)))
			return nil
		},
	}

	return cmd
}

// processTasksCmd returns the 'process tasks' subcommand
func processTasksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tasks FILE",
		Short: "Import a TASKS.md file into task storage",
		Long: `Parse a TASKS.md file (as produced by 'process prd') and save all tasks
to the devloop task storage without re-running the AI agent.

This is useful to recover from an interrupted 'process prd' run where the markdown
file was already written but the save step was cancelled.

Examples:
  devloop process tasks docs/TASKS.md`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]

			configPath, _ := cmd.Flags().GetString("config")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}

			fmt.Printf("Importing tasks from: %s\n", filePath)

			tasks, err := processor.ParseTasksMarkdown(filePath, cfg)
			if err != nil {
				return fmt.Errorf("failed to parse tasks file: %w", err)
			}

			if len(tasks) == 0 {
				fmt.Println("No tasks found in file.")
				return nil
			}

			displayTasksSummary(tasks)

			if dryRun {
				fmt.Printf("[dry-run] Would save %d task(s) to storage\n", len(tasks))
				fmt.Println("[dry-run] No tasks were saved")
				return nil
			}

			store, err := storage.NewTaskStore(cfg)
			if err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}
			savedCount := 0
			for _, task := range tasks {
				if err := store.SaveTask(task); err != nil {
					return fmt.Errorf("failed to save task %s: %w", task.ID, err)
				}
				savedCount++
			}

			tasksFilePath := filepath.Join(cfg.Project.Path, ".devloop", "tasks.jsonl")
			fmt.Printf("\n%s\n", ui.Success(fmt.Sprintf("Successfully imported %d task(s) to %s", savedCount, tasksFilePath)))
			return nil
		},
	}

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

		_ = table.Append(
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

// truncateString truncates a string to the specified length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
