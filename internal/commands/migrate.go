package commands

import (
	"fmt"
	"strings"

	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/storage"
	"github.com/spf13/cobra"
)

// MigrateTaskIDsOptions controls migration behavior
type MigrateTaskIDsOptions struct {
	DryRun bool
}

// MigrateTaskIDs migrates task IDs from hierarchical to JIRA format
func MigrateTaskIDs(cfg *config.Config, opts MigrateTaskIDsOptions) error {
	store := storage.NewStorage(cfg)
	tasks, err := store.LoadTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks to migrate.")
		return nil
	}

	// Detect current format
	if len(tasks) == 0 {
		fmt.Println("No tasks to migrate.")
		return nil
	}

	currentFormat := storage.DetectIDFormat(tasks[0].ID)
	if currentFormat == "jira" {
		fmt.Println("Tasks are already in JIRA format. No migration needed.")
		return nil
	}

	if currentFormat == "unknown" {
		fmt.Println("Unable to detect task ID format. Skipping migration.")
		return nil
	}

	// Build ID mapping: old → new
	prefix := cfg.Project.TaskIDPrefix
	if prefix == "" {
		prefix = config.DeriveTaskIDPrefix(cfg.Project.Name)
	}

	mapping := make(map[string]string)
	hierarchicalTasks := make([]*storage.Task, 0)

	// First pass: identify all hierarchical tasks to migrate
	for _, task := range tasks {
		if storage.DetectIDFormat(task.ID) == "hierarchical" {
			hierarchicalTasks = append(hierarchicalTasks, task)
		}
	}

	// Generate new IDs
	nextNum := 1
	for _, task := range hierarchicalTasks {
		newID := fmt.Sprintf("%s-%d", prefix, nextNum)
		mapping[task.ID] = newID
		nextNum++
	}

	// Second pass: update task IDs and BlockedBy references
	for _, task := range tasks {
		// Update task ID if it's being migrated
		if newID, ok := mapping[task.ID]; ok {
			task.ID = newID
		}

		// Update BlockedBy references
		for i, blockedID := range task.BlockedBy {
			if newID, ok := mapping[blockedID]; ok {
				task.BlockedBy[i] = newID
			}
		}
	}

	// Print migration report
	printMigrationReport(mapping, opts.DryRun)

	// Apply migration if not dry-run
	if !opts.DryRun {
		// Delete all tasks and re-save them with new IDs
		// This is necessary because UpdateTask searches by ID
		if err := store.DeleteAllTasks(); err != nil {
			return fmt.Errorf("failed to clear tasks for migration: %w", err)
		}

		// Save all tasks with updated IDs
		for _, task := range tasks {
			if err := store.SaveTask(task); err != nil {
				return fmt.Errorf("failed to save migrated task %s: %w", task.ID, err)
			}
		}

		// Update config to reflect new format
		cfg.Project.TaskIDFormat = "jira"
		cfg.Project.TaskIDPrefix = prefix
		configPath := store.ConfigPath()
		if err := config.SaveConfig(configPath, cfg); err != nil {
			return fmt.Errorf("failed to save updated config: %w", err)
		}

		fmt.Println("\n✓ Migration completed successfully!")
	} else {
		fmt.Println("\n(Dry-run mode - no changes applied)")
	}

	return nil
}

// printMigrationReport displays the migration plan/results
func printMigrationReport(mapping map[string]string, dryRun bool) {
	if dryRun {
		fmt.Println("\n=== Dry-Run: Migration Plan ===")
	} else {
		fmt.Println("\n=== Migration Report ===")
	}

	// Sort IDs for consistent output
	var oldIDs []string
	for oldID := range mapping {
		oldIDs = append(oldIDs, oldID)
	}

	// Sort hierarchical IDs numerically
	sortHierarchicalIDs(oldIDs)

	maxOldLen := 0
	for _, oldID := range oldIDs {
		if len(oldID) > maxOldLen {
			maxOldLen = len(oldID)
		}
	}

	fmt.Printf("\n%-*s  →  %s\n", maxOldLen, "Old ID", "New ID")
	fmt.Println(strings.Repeat("-", maxOldLen+14))

	for _, oldID := range oldIDs {
		newID := mapping[oldID]
		fmt.Printf("%-*s  →  %s\n", maxOldLen, oldID, newID)
	}

	fmt.Printf("\nTotal tasks migrated: %d\n", len(mapping))
}

// sortHierarchicalIDs sorts hierarchical task IDs numerically
func sortHierarchicalIDs(ids []string) {
	// Simple bubble sort for small lists
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			if storage.CompareTaskIDsUniversal(ids[j], ids[i]) {
				ids[i], ids[j] = ids[j], ids[i]
			}
		}
	}
}

// MigrateCmd returns the migrate-ids command
func MigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate-ids",
		Short: "Migrate task IDs from hierarchical to JIRA format",
		Long: `Convert task IDs from hierarchical format (e.g., 1.1, 2.5) to JIRA format (e.g., DEV-1, DEV-2).

This command:
- Detects the current task ID format
- Generates new JIRA-style IDs based on your project name
- Updates all task references and dependencies
- Preserves task data and execution history

Use --dry-run to preview changes before applying them.

Examples:
  devloop migrate-ids --dry-run    # Preview migration
  devloop migrate-ids              # Apply migration`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get config path from persistent flags
			configPath, _ := cmd.Flags().GetString("config")
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			// Load configuration
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			opts := MigrateTaskIDsOptions{
				DryRun: dryRun,
			}

			return MigrateTaskIDs(cfg, opts)
		},
	}

	return cmd
}
