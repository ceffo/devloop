package commands

import (
	"fmt"

	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/executor"
	"github.com/spf13/cobra"
)

// RunCmd returns the run command
func RunCmd() *cobra.Command {
	var (
		taskID          string
		continueSession bool
		resumeSession   bool
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute the development workflow",
		Long: `Execute pending tasks in the development workflow.

Tasks are executed in order, with AI agents handling implementation and
verification running after each attempt. Successfully completed tasks are
automatically committed (if enabled).

Flags:
  --task ID      Run a specific task by ID
  --continue     Resume execution from last checkpoint
  --dry-run      Show what would be executed without running

Global flags:
  --agent NAME   Use specific agent from config (default: first in list)

Examples:
  devloop run
  devloop run --task DEV-2
  devloop --agent claude run
  devloop run --continue
  devloop run --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get config path from persistent flags
			configPath, _ := cmd.Flags().GetString("config")
			agentName, _ := cmd.Flags().GetString("agent")
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			if dryRun {
				fmt.Println("[dry-run] Would execute the development workflow")
				if taskID != "" {
					fmt.Printf("[dry-run] Would run specific task: %s\n", taskID)
				} else {
					fmt.Println("[dry-run] Would run all pending tasks in order")
				}
				if continueSession || resumeSession {
					fmt.Println("[dry-run] Would resume from last checkpoint")
				}
				fmt.Println("[dry-run] No tasks will be executed")
				return nil
			}

			// Load configuration
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Validate configuration
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}

			// --resume is an alias for --continue
			shouldResume := continueSession || resumeSession

			// Execute dev loop
			err = executor.ExecuteDevLoop(cfg, taskID, shouldResume, dryRun, agentName)
			if err != nil {
				return err
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVar(&taskID, "task", "", "run specific task by ID")
	cmd.Flags().BoolVar(&continueSession, "continue", false, "resume from last checkpoint")
	cmd.Flags().BoolVar(&resumeSession, "resume", false, "resume from last checkpoint (alias for --continue)")

	return cmd
}
