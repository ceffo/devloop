package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// KnowledgeCmd returns the knowledge command with its subcommands.
func KnowledgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "knowledge",
		Short: "Manage knowledge base via mulch",
		Long: `Manage the mulch knowledge base.

Subcommands:
  status          Show knowledge base status
  query [domain]  Query knowledge for a domain
  search <query>  Search the knowledge base
  compact         Compact knowledge base (runs mulch compact --auto)
  doctor          Run knowledge base diagnostics
  diff [ref]      Show knowledge diff from a git ref
  prime [domains] Prime knowledge for specified domains

Examples:
  devloop knowledge status
  devloop knowledge query mypackage
  devloop knowledge search "authentication flow"
  devloop knowledge compact
  devloop knowledge doctor
  devloop knowledge diff HEAD~1
  devloop knowledge prime mypackage otherpackage`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(knowledgeStatusCmd())
	cmd.AddCommand(knowledgeQueryCmd())
	cmd.AddCommand(knowledgeSearchCmd())
	cmd.AddCommand(knowledgeCompactCmd())
	cmd.AddCommand(knowledgeDoctorCmd())
	cmd.AddCommand(knowledgeDiffCmd())
	cmd.AddCommand(knowledgePrimeCmd())

	return cmd
}

// checkMulch verifies mulch is installed, returning an error with install instructions if not.
func checkMulch() error {
	if _, err := exec.LookPath("mulch"); err != nil {
		fmt.Fprintln(os.Stderr, "mulch is not installed or not found in PATH.")
		fmt.Fprintln(os.Stderr, "Install mulch using:")
		fmt.Fprintln(os.Stderr, "  npm install -g mulch-cli")
		fmt.Fprintln(os.Stderr, "\nAfter installing, run 'devloop init' to set up the knowledge backend.")
		return fmt.Errorf("mulch not found in PATH")
	}
	return nil
}

// runMulch runs a mulch subcommand, wiring stdin/stdout/stderr through.
func runMulch(args ...string) error {
	if err := checkMulch(); err != nil {
		return err
	}
	cmd := exec.Command("mulch", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func knowledgeStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show knowledge base status",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runMulch("status")
		},
	}
}

func knowledgeQueryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "query [domain]",
		Short: "Query knowledge for a domain",
		RunE: func(_ *cobra.Command, args []string) error {
			return runMulch(append([]string{"query"}, args...)...)
		},
	}
}

func knowledgeSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search the knowledge base",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runMulch(append([]string{"search"}, args...)...)
		},
	}
}

func knowledgeCompactCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "compact",
		Short: "Compact the knowledge base",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runMulch("compact", "--auto")
		},
	}
}

func knowledgeDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run knowledge base diagnostics",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runMulch("doctor")
		},
	}
}

func knowledgeDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff [ref]",
		Short: "Show knowledge diff from a git ref",
		RunE: func(_ *cobra.Command, args []string) error {
			return runMulch(append([]string{"diff"}, args...)...)
		},
	}
}

func knowledgePrimeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prime [domains...]",
		Short: "Prime knowledge for specified domains",
		RunE: func(_ *cobra.Command, args []string) error {
			return runMulch(append([]string{"prime"}, args...)...)
		},
	}
}
