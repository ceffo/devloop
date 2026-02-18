package commands

import (
	"fmt"

	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/executor"
	"github.com/ceffo/devloop/internal/ui"
	"github.com/spf13/cobra"
)

// SessionCmd returns the session command with its subcommands
func SessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage execution sessions",
		Long: `View and manage execution sessions for crash recovery.

Subcommands:
  status   Show current session status
  recover  Recover from last checkpoint

Examples:
  devloop session status
  devloop session recover`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(sessionStatusCmd())
	cmd.AddCommand(sessionRecoverCmd())

	return cmd
}

// sessionStatusCmd shows the current session state
func sessionStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current session status",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")

			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			session, exists := executor.LoadSessionForResume(cfg)
			if !exists {
				fmt.Println(ui.Warning("No session found. Run 'devloop run' to start a session."))
				return nil
			}

			fmt.Println(ui.Section("Session Status"))
			fmt.Printf("  ID:              %s\n", session.ID)
			fmt.Printf("  Started:         %s\n", session.StartedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Last checkpoint: %s\n", orNone(session.LastCheckpoint))
			fmt.Printf("  Agent:           %s\n", orNone(session.AgentName))

			fmt.Printf("\n  Tasks completed: %d\n", len(session.TasksCompleted))
			for _, id := range session.TasksCompleted {
				fmt.Printf("    ✓ %s\n", id)
			}

			fmt.Printf("\n  Tasks failed:    %d\n", len(session.TasksFailed))
			for _, id := range session.TasksFailed {
				fmt.Printf("    ✗ %s\n", id)
			}

			if len(session.TaskSnapshot) > 0 {
				fmt.Printf("\n  Task snapshot (%d tasks at session start):\n", len(session.TaskSnapshot))
				for _, t := range session.TaskSnapshot {
					fmt.Printf("    [%s] %s (%s)\n", t.ID, t.Title, t.Complexity)
				}
			}

			if session.LastCheckpoint != "" {
				fmt.Printf("\n%s\n", ui.Info("Run 'devloop resume' or 'devloop run --resume' to continue from last checkpoint."))
			}

			return nil
		},
	}
}

// sessionRecoverCmd resumes from the last checkpoint
func sessionRecoverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recover",
		Short: "Recover execution from last checkpoint",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			agentName, _ := cmd.Flags().GetString("agent")

			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}

			_, exists := executor.LoadSessionForResume(cfg)
			if !exists {
				return fmt.Errorf("no previous session found - run 'devloop run' to start a new session")
			}

			return executor.ExecuteDevLoop(cmd.Context(), cfg, "", true, false, agentName)
		},
	}

	return cmd
}

// orNone returns the value or "(none)" if empty
func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}
