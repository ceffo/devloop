package commands

import (
	"fmt"

	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/executor"
	"github.com/spf13/cobra"
)

// ResumeCmd returns the resume command
func ResumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Resume the last execution session from where it left off",
		Long: `Resume the last execution session from the last checkpoint.

If a previous session exists, execution continues from the next incomplete task.
If no previous session exists, an error is printed.

Examples:
  devloop resume
  devloop --agent claude resume
  devloop resume --dry-run`,
RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			agentName, _ := cmd.Flags().GetString("agent")
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			if dryRun {
				fmt.Println("[dry-run] Would resume the last execution session from the last checkpoint")
				fmt.Println("[dry-run] No tasks will be executed")
				return nil
			}

			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}

			// Check that a previous session exists before attempting to resume
			_, exists := executor.LoadSessionForResume(cfg)
			if !exists {
				return fmt.Errorf("no previous session found - run 'devloop run' to start a new session")
			}

			return executor.ExecuteDevLoop(cmd.Context(), cfg, "", true, dryRun, agentName, false)
		},
	}

	return cmd
}
