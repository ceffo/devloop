// Package commands implements the devloop CLI commands.
package commands

import (
	"fmt"

	"github.com/ceffo/devloop/internal/archiver"
	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/storage"
	"github.com/ceffo/devloop/internal/ui"
	"github.com/spf13/cobra"
)

// ArchiveCmd returns the archive command
func ArchiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive",
		Short: "Archive completed tasks",
		Long: `Archive completed tasks to prevent context bloat.

Completed tasks are exported to JSONL and a markdown summary in .devloop/archive/
and their status is updated to 'archived' in active storage.

Example:
  devloop archive`,
RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			store := storage.NewStorage(cfg)

			completedTasks, err := store.QueryTasks(storage.Filter{Status: "completed"})
			if err != nil {
				return fmt.Errorf("failed to query completed tasks: %w", err)
			}

			if len(completedTasks) == 0 {
				fmt.Println("No completed tasks to archive.")
				return nil
			}

			if dryRun {
				fmt.Printf("[dry-run] Would archive %d completed task(s)\n", len(completedTasks))
				for _, t := range completedTasks {
					fmt.Printf("[dry-run]   - %s: %s\n", t.ID, t.Title)
				}
				fmt.Println("[dry-run] No files were written")
				return nil
			}

			fmt.Printf("Archiving %d completed task(s)...\n\n", len(completedTasks))

			arc := archiver.NewArchiver(cfg, store)

			result, err := arc.ExportCompleted()
			if err != nil {
				return fmt.Errorf("failed to export completed tasks: %w", err)
			}

			mdSummary, err := arc.GenerateMarkdownSummary(completedTasks)
			if err != nil {
				return fmt.Errorf("failed to generate markdown summary: %w", err)
			}

			mdPath, err := arc.WriteMarkdownSummary(mdSummary)
			if err != nil {
				return fmt.Errorf("failed to write markdown summary: %w", err)
			}

			fmt.Println(ui.Success(fmt.Sprintf("Archived %d task(s) successfully", result.Total)))
			fmt.Printf("\n%s\n", ui.Heading2("Archive Summary:"))
			fmt.Printf("  Tasks Archived:  %d\n", result.Total)
			fmt.Printf("  JSONL Output:    %s\n", result.OutputPath)
			fmt.Printf("  Markdown Output: %s\n\n", mdPath)

			return nil
		},
	}

	return cmd
}
