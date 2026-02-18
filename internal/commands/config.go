package commands

import (
	"encoding/json"
	"fmt"

	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/ui"
	"github.com/spf13/cobra"
)

// ConfigCmd returns the config command
func ConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage devloop configuration",
		Long: `View and validate devloop configuration.

Subcommands:
  show      Display current configuration
  validate  Validate configuration file

Example:
  devloop config show
  devloop config validate`,
	}

	cmd.AddCommand(configShowCmd())
	cmd.AddCommand(configValidateCmd())

	return cmd
}

// configShowCmd returns the config show subcommand
func configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Display current configuration",
		Long:  `Pretty-print the current devloop configuration with colors.`,
			RunE: func(cmd *cobra.Command, _ []string) error {
			// Get config path from global flag
			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return fmt.Errorf("failed to get config path: %w", err)
			}

			// Load config
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Pretty-print JSON with colors
			data, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal config: %w", err)
			}

			// Print header
			fmt.Println(ui.Heading2("Configuration:"))
			fmt.Printf("%s\n\n", ui.LabelValue("Path", configPath))

			// Print JSON (colorized if possible)
			fmt.Println(string(data))

			return nil
		},
	}
}

// configValidateCmd returns the config validate subcommand
func configValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long:  `Load configuration and validate all required fields and constraints.`,
			RunE: func(cmd *cobra.Command, _ []string) error {
			// Get config path from global flag
			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return fmt.Errorf("failed to get config path: %w", err)
			}

			// Load config
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("Failed to load config: %v", err)))
				return err
			}

			fmt.Println(ui.Success(fmt.Sprintf("Config loaded successfully from %s", configPath)))

			// Validate config
			if err := cfg.Validate(); err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("Validation failed: %v", err)))
				return err
			}

			fmt.Println(ui.Success("Configuration is valid"))

			// Print summary
			fmt.Println()
			fmt.Println(ui.Heading2("Configuration Summary:"))
			fmt.Printf("  Project:       %s\n", cfg.Project.Name)
			fmt.Printf("  Path:          %s\n", cfg.Project.Path)
			fmt.Printf("  Tech Stack:    %s\n", cfg.Project.TechStack)
			fmt.Printf("  Main Branch:   %s\n", cfg.Project.MainBranch)
			fmt.Printf("  Agents:        %s\n", cfg.CLI.GetDefaultAgentName())
			fmt.Printf("  Max Attempts:  %d\n", cfg.Execution.MaxAttempts)
			fmt.Printf("  Auto Commit:   %v\n", cfg.Execution.AutoCommit)

			return nil
		},
	}
}
