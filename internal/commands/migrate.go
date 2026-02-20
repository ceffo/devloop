package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/storage"
	"github.com/spf13/cobra"
)

// MigrateCmd returns the migrate command
func MigrateCmd() *cobra.Command {
	var toFlag string

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate tasks between storage backends",
		Long: `Migrate tasks between JSONL and Beads storage backends.

Directions:
  --to beads  Read tasks from tasks.jsonl, create each in Beads with correct
              status and dependency mappings, run bd sync, then rename
              tasks.jsonl to tasks.jsonl.migrated.
  --to jsonl  Read all tasks from Beads and write them to tasks.jsonl with
              reverse status mapping.

Examples:
  devloop migrate --to beads
  devloop migrate --to jsonl`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if toFlag == "" {
				return fmt.Errorf("--to flag is required (values: beads, jsonl)")
			}

			configPath, _ := cmd.Flags().GetString("config")
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			ctx := context.Background()

			switch toFlag {
			case "beads":
				return migrateToBeads(ctx, cfg)
			case "jsonl":
				return migrateToJSONL(ctx, cfg)
			default:
				return fmt.Errorf("unknown migration target %q (valid: beads, jsonl)", toFlag)
			}
		},
	}

	cmd.Flags().StringVar(&toFlag, "to", "", "migration target: beads or jsonl (required)")

	return cmd
}

// migrateToBeads reads tasks from JSONL, creates each in Beads with correct
// status and dependency mappings, runs bd sync, and renames tasks.jsonl.
func migrateToBeads(ctx context.Context, cfg *config.Config) error {
	jsonlStore := storage.NewJSONLStore(cfg)

	tasks, err := jsonlStore.LoadTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks from JSONL: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found in tasks.jsonl. Nothing to migrate.")
		return nil
	}

	beadsStore, err := storage.NewBeadsStore(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize Beads store: %w", err)
	}

	fmt.Printf("Migrating %d task(s) to Beads...\n\n", len(tasks))

	migrated := 0
	skipped := 0
	for _, task := range tasks {
		fmt.Printf("  [%s] %s ...", task.ID, truncateTitle(task.Title, 50))

		if err := beadsStore.SaveTask(ctx, task); err != nil {
			fmt.Printf(" FAILED\n    error: %v\n", err)
			skipped++
			continue
		}

		// Set status if not pending (SaveTask creates tasks as "open"/pending by default)
		if task.Status != "pending" {
			// Resolve the beads ID that was just created
			beadsID, resolveErr := resolveAfterCreate(ctx, beadsStore, task.ID)
			if resolveErr != nil {
				fmt.Printf(" status WARNING\n    could not resolve beads ID to set status %q: %v\n", task.Status, resolveErr)
			} else if setErr := beadsStore.SetMigrationStatus(ctx, beadsID, task.Status); setErr != nil {
				fmt.Printf(" status WARNING\n    could not set status %q: %v\n", task.Status, setErr)
			}
		}

		fmt.Printf(" OK (status: %s)\n", task.Status)
		migrated++
	}

	fmt.Println()

	// Run bd sync
	fmt.Print("Running bd sync...")
	if syncErr := beadsStore.Sync(); syncErr != nil {
		fmt.Printf(" WARNING: %v\n", syncErr)
	} else {
		fmt.Println(" OK")
	}

	// Rename tasks.jsonl → tasks.jsonl.migrated
	tasksFile := jsonlStore.TasksFilePath()
	migratedFile := tasksFile + ".migrated"
	migratedFile = uniqueMigratedPath(migratedFile)
	if renameErr := os.Rename(tasksFile, migratedFile); renameErr != nil {
		fmt.Printf("Warning: could not rename tasks.jsonl: %v\n", renameErr)
	} else {
		fmt.Printf("Renamed %s → %s\n", tasksFile, migratedFile)
	}

	fmt.Printf("\nMigration summary: %d migrated, %d skipped\n", migrated, skipped)
	return nil
}

// resolveAfterCreate resolves a devloop ID to its Beads ID using a KV lookup.
// This is used after SaveTask to find the newly created task's Beads ID.
func resolveAfterCreate(ctx context.Context, store *storage.BeadsStore, devloopID string) (string, error) {
	// Use a short retry since the KV entry is written synchronously in SaveTask
	resolveCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// GetTask internally resolves the ID via KV lookup
	task, err := store.GetTask(resolveCtx, devloopID)
	if err != nil {
		return "", err
	}

	// task.ID at this point is the beads hash ID returned by bd show
	return task.ID, nil
}

// uniqueMigratedPath returns a path that doesn't conflict with an existing file.
// If the path exists, it appends a timestamp suffix.
func uniqueMigratedPath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	return path + "." + time.Now().Format("20060102150405")
}

// migrateToJSONL reads all tasks from Beads and writes them to tasks.jsonl.
func migrateToJSONL(ctx context.Context, cfg *config.Config) error {
	beadsStore, err := storage.NewBeadsStore(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize Beads store: %w", err)
	}

	tasks, err := beadsStore.LoadAllTasks(ctx)
	if err != nil {
		return fmt.Errorf("failed to load tasks from Beads: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found in Beads. Nothing to migrate.")
		return nil
	}

	fmt.Printf("Migrating %d task(s) to JSONL...\n\n", len(tasks))

	jsonlStore := storage.NewJSONLStore(cfg)

	// Ensure the .devloop directory exists before writing
	tasksFile := jsonlStore.TasksFilePath()
	if err := ensureParentDir(tasksFile); err != nil {
		return fmt.Errorf("failed to create .devloop directory: %w", err)
	}

	// Truncate any existing tasks.jsonl so we start fresh
	if err := os.WriteFile(tasksFile, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to reset tasks.jsonl: %w", err)
	}

	migrated := 0
	skipped := 0
	for _, task := range tasks {
		fmt.Printf("  [%s] %s ...", task.ID, truncateTitle(task.Title, 50))

		if err := jsonlStore.SaveTask(task); err != nil {
			fmt.Printf(" FAILED\n    error: %v\n", err)
			skipped++
			continue
		}

		fmt.Printf(" OK (status: %s)\n", task.Status)
		migrated++
	}

	fmt.Printf("\nMigration summary: %d migrated, %d skipped\n", migrated, skipped)
	fmt.Printf("Tasks written to: %s\n", tasksFile)
	return nil
}

// ensureParentDir creates parent directories for the given file path.
func ensureParentDir(filePath string) error {
	dir := filePath
	for i := len(dir) - 1; i >= 0; i-- {
		if dir[i] == '/' || dir[i] == '\\' {
			dir = dir[:i]
			break
		}
	}
	return os.MkdirAll(dir, 0755)
}
