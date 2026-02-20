package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// VersionCmd returns the version command
func VersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version of devloop",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Println("devloop version 0.2.0")
			return nil
		},
	}
}
