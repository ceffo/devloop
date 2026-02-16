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
		wave            int
		taskID          string
		continueSession bool
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute the development workflow",
		Long: `Execute pending tasks in the development workflow.

Tasks are executed in order, with AI agents handling implementation and
verification running after each attempt. Successfully completed tasks are
automatically committed (if enabled).

Flags:
  --wave N      Run only tasks in wave N
  --task ID     Run a specific task by ID
  --continue    Resume execution from last checkpoint

Examples:
  devloop run
  devloop run --wave 1
  devloop run --task 1.2
  devloop run --continue`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get config path from persistent flags
			configPath, _ := cmd.Flags().GetString("config")

			// Load configuration
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Validate configuration
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}

			// Execute dev loop
			err = executor.ExecuteDevLoop(cfg, wave, taskID, continueSession)
			if err != nil {
				return err
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().IntVar(&wave, "wave", 0, "filter by wave number (0 = all waves)")
	cmd.Flags().StringVar(&taskID, "task", "", "run specific task by ID")
	cmd.Flags().BoolVar(&continueSession, "continue", false, "resume from last checkpoint")

	return cmd
}
