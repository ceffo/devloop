package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/storage"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// TasksCmd returns the tasks command with subcommands
func TasksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "Manage and view tasks",
		Long: `List, show, and update tasks.

Subcommands:
  list    List tasks with optional filtering
  show    Show detailed task information
  update  Update task status

Example:
  devloop tasks list
  devloop tasks list --status pending
  devloop tasks show 1.2
  devloop tasks update 1.2 --status completed`,
	}

	// Add subcommands
	cmd.AddCommand(tasksListCmd())
	cmd.AddCommand(tasksShowCmd())
	cmd.AddCommand(tasksUpdateCmd())

	return cmd
}

// tasksListCmd returns the tasks list subcommand
func tasksListCmd() *cobra.Command {
	var (
		statusFilter     string
		waveFilter       int
		complexityFilter string
		tagsFilter       string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks with optional filtering",
		Long: `List all tasks with optional filtering by status, wave, complexity, or tags.

The output is displayed as a formatted table with columns:
  - ID: Task identifier (hierarchical)
  - Title: Task title
  - Status: Current task status (color-coded)
  - Complexity: Task complexity level
  - Attempts: Number of execution attempts
  - Duration: Total execution time in seconds

Examples:
  devloop tasks list
  devloop tasks list --status pending
  devloop tasks list --wave 1
  devloop tasks list --complexity moderate
  devloop tasks list --tags backend,api`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get config path from persistent flags
			configPath, _ := cmd.Flags().GetString("config")

			// Load configuration
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Create storage
			store := storage.NewStorage(cfg)

			// Build filter
			filter := storage.Filter{
				Status:     statusFilter,
				Wave:       waveFilter,
				Complexity: complexityFilter,
			}

			// Parse tags if provided
			if tagsFilter != "" {
				filter.Tags = strings.Split(tagsFilter, ",")
				// Trim spaces from each tag
				for i := range filter.Tags {
					filter.Tags[i] = strings.TrimSpace(filter.Tags[i])
				}
			}

			// Query tasks
			tasks, err := store.QueryTasks(filter)
			if err != nil {
				return fmt.Errorf("failed to query tasks: %w", err)
			}

			// Display results
			if len(tasks) == 0 {
				fmt.Println("No tasks found matching the specified criteria.")
				return nil
			}

			displayTasksTable(tasks)

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVar(&statusFilter, "status", "", "filter by status (pending, in_progress, completed, failed, blocked, archived)")
	cmd.Flags().IntVar(&waveFilter, "wave", 0, "filter by wave number")
	cmd.Flags().StringVar(&complexityFilter, "complexity", "", "filter by complexity (simple, moderate, complex)")
	cmd.Flags().StringVar(&tagsFilter, "tags", "", "filter by tags (comma-separated)")

	return cmd
}

// displayTasksTable renders tasks in a formatted table
func displayTasksTable(tasks []*storage.Task) {
	table := tablewriter.NewWriter(os.Stdout)

	// Set header
	table.Header("ID", "Title", "Status", "Complexity", "Attempts", "Duration")

	// Add rows
	for _, task := range tasks {
		table.Append(
			task.ID,
			truncateTitle(task.Title, 50),
			formatStatus(task.Status),
			task.Complexity,
			strconv.Itoa(len(task.Execution.Attempts)),
			formatDuration(task.Execution.TotalDuration),
		)
	}

	// Render the table
	if err := table.Render(); err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering table: %v\n", err)
	}
}

// formatStatus returns a color-coded status string
func formatStatus(status string) string {
	switch status {
	case "completed":
		return color.GreenString(status)
	case "in_progress":
		return color.YellowString(status)
	case "failed":
		return color.RedString(status)
	case "blocked":
		return color.MagentaString(status)
	case "archived":
		return color.CyanString(status)
	default:
		return status
	}
}

// formatDuration formats duration in seconds to a human-readable string
func formatDuration(seconds int) string {
	if seconds == 0 {
		return "-"
	}

	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}

	minutes := seconds / 60
	remainingSeconds := seconds % 60

	if minutes < 60 {
		if remainingSeconds > 0 {
			return fmt.Sprintf("%dm%ds", minutes, remainingSeconds)
		}
		return fmt.Sprintf("%dm", minutes)
	}

	hours := minutes / 60
	remainingMinutes := minutes % 60

	if remainingMinutes > 0 {
		return fmt.Sprintf("%dh%dm", hours, remainingMinutes)
	}
	return fmt.Sprintf("%dh", hours)
}

// truncateTitle truncates a title to the specified length with ellipsis
func truncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}
	return title[:maxLen-3] + "..."
}

// tasksShowCmd returns the tasks show subcommand
func tasksShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show TASK_ID",
		Short: "Show detailed task information",
		Long: `Display complete details for a specific task.

Shows all task fields including:
  - Metadata (ID, wave, complexity, status, timestamps)
  - Description and acceptance criteria
  - Execution history with attempt details
  - Results (if completed)

Example:
  devloop tasks show 1.2
  devloop tasks show 2.5`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			// Get config path from persistent flags
			configPath, _ := cmd.Flags().GetString("config")

			// Load configuration
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Create storage
			store := storage.NewStorage(cfg)

			// Get task
			task, err := store.GetTask(taskID)
			if err != nil {
				return fmt.Errorf("failed to get task: %w", err)
			}

			if task == nil {
				return fmt.Errorf("task not found: %s", taskID)
			}

			// Display task details
			displayTaskDetails(task)

			return nil
		},
	}

	return cmd
}

// displayTaskDetails renders detailed task information
func displayTaskDetails(task *storage.Task) {
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)

	// Header
	bold.Printf("\n=== Task %s: %s ===\n\n", task.ID, task.Title)

	// Metadata Section
	cyan.Println("Metadata:")
	fmt.Printf("  Wave:         %d\n", task.Wave)
	fmt.Printf("  Status:       %s\n", formatStatus(task.Status))
	fmt.Printf("  Complexity:   %s\n", task.Complexity)
	fmt.Printf("  Model:        %s\n", task.Model)
	fmt.Printf("  Created:      %s\n", task.Metadata.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Updated:      %s\n", task.Metadata.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Max Attempts: %d\n", task.Metadata.MaxAttempts)

	if task.Metadata.SourceType != "" {
		fmt.Printf("  Source Type:  %s\n", task.Metadata.SourceType)
	}
	if task.Metadata.SourceTodoItem != "" {
		fmt.Printf("  Source TODO:  %s\n", task.Metadata.SourceTodoItem)
	}

	// Dependencies
	if len(task.BlockedBy) > 0 {
		fmt.Printf("  Blocked By:   %s\n", strings.Join(task.BlockedBy, ", "))
	}

	// Tags
	if len(task.Tags) > 0 {
		fmt.Printf("  Tags:         %s\n", strings.Join(task.Tags, ", "))
	}

	// Description Section
	fmt.Println()
	cyan.Println("Description:")
	if task.Description != "" {
		fmt.Printf("  %s\n", strings.ReplaceAll(task.Description, "\n", "\n  "))
	} else {
		fmt.Println("  (no description)")
	}

	// Acceptance Criteria Section
	fmt.Println()
	cyan.Println("Acceptance Criteria:")
	if len(task.AcceptanceCriteria) > 0 {
		for _, criterion := range task.AcceptanceCriteria {
			fmt.Printf("  • %s\n", criterion)
		}
	} else {
		fmt.Println("  (no criteria defined)")
	}

	// Execution History Section
	fmt.Println()
	cyan.Println("Execution History:")
	if len(task.Execution.Attempts) > 0 {
		fmt.Printf("  Total Duration: %s\n", formatDuration(task.Execution.TotalDuration))
		fmt.Printf("  Attempts:       %d\n\n", len(task.Execution.Attempts))

		// Display attempts table
		table := tablewriter.NewWriter(os.Stdout)
		table.Header("#", "Model", "Duration", "Result", "Log Path")

		for _, attempt := range task.Execution.Attempts {
			result := formatAttemptResult(attempt.Result, attempt.Success)
			duration := formatDuration(attempt.Duration)

			table.Append(
				strconv.Itoa(attempt.AttemptNumber),
				attempt.Model,
				duration,
				result,
				attempt.LogPath,
			)

			// If there's a verification log, add it as a sub-row
			if attempt.VerifyLogPath != "" {
				table.Append(
					"",
					"",
					"",
					"verify",
					attempt.VerifyLogPath,
				)
			}

			// If there's an error, show it
			if attempt.Error != "" {
				fmt.Printf("\n  Error (Attempt %d): %s\n", attempt.AttemptNumber, attempt.Error)
			}
		}

		if err := table.Render(); err != nil {
			fmt.Fprintf(os.Stderr, "Error rendering table: %v\n", err)
		}
	} else {
		fmt.Println("  (no attempts yet)")
	}

	// Results Section (if completed)
	if task.Results != nil {
		fmt.Println()
		cyan.Println("Results:")
		fmt.Printf("  Completed At:  %s\n", task.Results.CompletedAt.Format("2006-01-02 15:04:05"))

		if task.Results.CommitHash != "" {
			fmt.Printf("  Commit Hash:   %s\n", task.Results.CommitHash)
		}

		if task.Results.VerificationOutput != "" {
			fmt.Println("\n  Verification Output:")
			// Show first 10 lines of verification output
			lines := strings.Split(task.Results.VerificationOutput, "\n")
			maxLines := 10
			if len(lines) > maxLines {
				for i := 0; i < maxLines; i++ {
					fmt.Printf("    %s\n", lines[i])
				}
				fmt.Printf("    ... (%d more lines)\n", len(lines)-maxLines)
			} else {
				for _, line := range lines {
					fmt.Printf("    %s\n", line)
				}
			}
		}
	}

	fmt.Println()
}

// formatAttemptResult formats the attempt result with color coding
func formatAttemptResult(result string, success bool) string {
	if success {
		return color.GreenString(result)
	}
	return color.RedString(result)
}

// tasksUpdateCmd returns the tasks update subcommand
func tasksUpdateCmd() *cobra.Command {
	var statusFlag string

	cmd := &cobra.Command{
		Use:   "update TASK_ID",
		Short: "Update task status",
		Long: `Update the status of a specific task.

Valid status values:
  - pending      Task is ready to be executed
  - in_progress  Task is currently being executed
  - completed    Task has been successfully completed
  - failed       Task execution failed
  - blocked      Task is blocked by dependencies
  - archived     Task has been archived

Example:
  devloop tasks update 1.2 --status completed
  devloop tasks update 2.5 --status blocked`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			// Validate that status flag is provided
			if statusFlag == "" {
				return fmt.Errorf("--status flag is required")
			}

			// Validate status value
			validStatuses := map[string]bool{
				"pending":     true,
				"in_progress": true,
				"completed":   true,
				"failed":      true,
				"blocked":     true,
				"archived":    true,
			}

			if !validStatuses[statusFlag] {
				return fmt.Errorf("invalid status: %s (valid values: pending, in_progress, completed, failed, blocked, archived)", statusFlag)
			}

			// Get config path from persistent flags
			configPath, _ := cmd.Flags().GetString("config")

			// Load configuration
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Create storage
			store := storage.NewStorage(cfg)

			// Get task
			task, err := store.GetTask(taskID)
			if err != nil {
				return fmt.Errorf("task not found: %s", taskID)
			}

			// Store old status for confirmation message
			oldStatus := task.Status

			// Update task status and timestamp
			task.Status = statusFlag
			task.Metadata.UpdatedAt = time.Now()

			// Save updated task
			if err := store.UpdateTask(task); err != nil {
				return fmt.Errorf("failed to update task: %w", err)
			}

			// Display success message
			fmt.Printf("Task %s status updated: %s → %s\n",
				color.CyanString(taskID),
				formatStatus(oldStatus),
				formatStatus(statusFlag))

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVar(&statusFlag, "status", "", "new status (required: pending, in_progress, completed, failed, blocked, archived)")
	cmd.MarkFlagRequired("status")

	return cmd
}
