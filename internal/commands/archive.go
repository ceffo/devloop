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
	var (
		waveFlag int
		autoFlag bool
	)

	cmd := &cobra.Command{
		Use:   "archive",
		Short: "Archive completed waves",
		Long: `Archive completed waves to prevent context bloat.

Archived waves are exported to JSONL and markdown summaries in .devloop/archive/
and removed from active task context.

Example:
  devloop archive --wave 1
  devloop archive --auto`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get config path from persistent flags
			configPath, _ := cmd.Flags().GetString("config")

			// Load configuration
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Create storage and archiver
			store := storage.NewStorage(cfg)
			arc := archiver.NewArchiver(cfg, store)

			// Validate flags
			if waveFlag > 0 && autoFlag {
				return fmt.Errorf("cannot use both --wave and --auto flags together")
			}

			if waveFlag == 0 && !autoFlag {
				return fmt.Errorf("must specify either --wave N or --auto")
			}

			// Handle auto mode
			if autoFlag {
				return archiveAuto(arc, store)
			}

			// Handle specific wave
			return archiveWave(arc, store, waveFlag)
		},
	}

	// Add flags
	cmd.Flags().IntVar(&waveFlag, "wave", 0, "archive a specific wave number")
	cmd.Flags().BoolVar(&autoFlag, "auto", false, "automatically archive all fully completed waves")

	return cmd
}

// archiveWave archives a specific wave
func archiveWave(arc *archiver.Archiver, store *storage.Storage, wave int) error {
	fmt.Printf("Archiving wave %d...\n\n", wave)

	// Check if wave has any tasks
	tasks, err := store.QueryTasks(storage.Filter{Wave: wave})
	if err != nil {
		return fmt.Errorf("failed to query tasks for wave %d: %w", wave, err)
	}

	if len(tasks) == 0 {
		return fmt.Errorf("no tasks found for wave %d", wave)
	}

	// Check if wave has any completed tasks
	completedTasks, err := store.QueryTasks(storage.Filter{
		Wave:   wave,
		Status: "completed",
	})
	if err != nil {
		return fmt.Errorf("failed to query completed tasks for wave %d: %w", wave, err)
	}

	if len(completedTasks) == 0 {
		return fmt.Errorf("no completed tasks found for wave %d", wave)
	}

	// Export wave to JSONL
	result, err := arc.ExportWave(wave)
	if err != nil {
		return fmt.Errorf("failed to export wave %d: %w", wave, err)
	}

	// Generate and write markdown summary
	mdSummary, err := arc.GenerateMarkdownSummary(wave, completedTasks)
	if err != nil {
		return fmt.Errorf("failed to generate markdown summary: %w", err)
	}

	mdPath, err := arc.WriteMarkdownSummary(wave, mdSummary)
	if err != nil {
		return fmt.Errorf("failed to write markdown summary: %w", err)
	}

	// Display results
	fmt.Println(ui.Success(fmt.Sprintf("Wave %d archived successfully", wave)))
	fmt.Printf("\n%s\n", ui.Heading2("Archive Summary:"))
	fmt.Printf("  Tasks Archived:  %d\n", result.Total)
	fmt.Printf("  JSONL Output:    %s\n", result.OutputPath)
	fmt.Printf("  Markdown Output: %s\n\n", mdPath)

	return nil
}

// archiveAuto automatically archives all fully completed waves
func archiveAuto(arc *archiver.Archiver, store *storage.Storage) error {
	fmt.Println("Scanning for fully completed waves...")
	fmt.Println()

	// Load all tasks
	allTasks, err := store.LoadTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	if len(allTasks) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	// Find all unique waves and check if they're fully completed
	waveMap := make(map[int][]*storage.Task)
	for _, task := range allTasks {
		waveMap[task.Wave] = append(waveMap[task.Wave], task)
	}

	// Determine which waves are fully completed
	fullyCompletedWaves := []int{}
	for wave, tasks := range waveMap {
		if isWaveFullyCompleted(tasks) {
			fullyCompletedWaves = append(fullyCompletedWaves, wave)
		}
	}

	if len(fullyCompletedWaves) == 0 {
		fmt.Println("No fully completed waves found.")
		return nil
	}

	// Archive each fully completed wave
	totalArchived := 0
	for _, wave := range fullyCompletedWaves {
		fmt.Printf("Archiving wave %d...\n", wave)

		// Get completed tasks for this wave
		completedTasks, err := store.QueryTasks(storage.Filter{
			Wave:   wave,
			Status: "completed",
		})
		if err != nil {
			fmt.Println(ui.Warning(fmt.Sprintf("Failed to query completed tasks for wave %d: %v", wave, err)))
			continue
		}

		// Export wave
		result, err := arc.ExportWave(wave)
		if err != nil {
			fmt.Println(ui.Warning(fmt.Sprintf("Failed to export wave %d: %v", wave, err)))
			continue
		}

		// Generate and write markdown summary
		mdSummary, err := arc.GenerateMarkdownSummary(wave, completedTasks)
		if err != nil {
			fmt.Println(ui.Warning(fmt.Sprintf("Failed to generate markdown summary for wave %d: %v", wave, err)))
			continue
		}

		_, err = arc.WriteMarkdownSummary(wave, mdSummary)
		if err != nil {
			fmt.Println(ui.Warning(fmt.Sprintf("Failed to write markdown summary for wave %d: %v", wave, err)))
			continue
		}

		totalArchived += result.Total
		fmt.Printf("  ✓ Wave %d: %d tasks archived\n", wave, result.Total)
	}

	// Display summary
	fmt.Printf("\n%s\n", ui.Success(fmt.Sprintf("Archived %d waves with %d total tasks", len(fullyCompletedWaves), totalArchived)))

	return nil
}

// isWaveFullyCompleted checks if all tasks in a wave are completed or archived
func isWaveFullyCompleted(tasks []*storage.Task) bool {
	if len(tasks) == 0 {
		return false
	}

	for _, task := range tasks {
		// Only consider completed or archived as finished
		// Tasks that are pending, in_progress, failed, or blocked mean the wave is not complete
		if task.Status != "completed" && task.Status != "archived" {
			return false
		}
	}

	return true
}
