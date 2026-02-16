package commands

import (
	"encoding/json"
	"fmt"

	"github.com/ceffo/devloop/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	// Colors for output
	green = color.New(color.FgGreen)
	red   = color.New(color.FgRed)
	cyan  = color.New(color.FgCyan)
	bold  = color.New(color.Bold)
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
		RunE: func(cmd *cobra.Command, args []string) error {
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
			bold.Println("Configuration:")
			cyan.Printf("Path: %s\n\n", configPath)

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
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get config path from global flag
			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return fmt.Errorf("failed to get config path: %w", err)
			}

			// Load config
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				red.Printf("✗ Failed to load config: %v\n", err)
				return err
			}

			green.Printf("✓ Config loaded successfully from %s\n", configPath)

			// Validate config
			if err := cfg.Validate(); err != nil {
				red.Printf("✗ Validation failed: %v\n", err)
				return err
			}

			green.Println("✓ Configuration is valid")

			// Print summary
			fmt.Println()
			bold.Println("Configuration Summary:")
			fmt.Printf("  Project:       %s\n", cfg.Project.Name)
			fmt.Printf("  Path:          %s\n", cfg.Project.Path)
			fmt.Printf("  Tech Stack:    %s\n", cfg.Project.TechStack)
			fmt.Printf("  Main Branch:   %s\n", cfg.Project.MainBranch)
			fmt.Printf("  CLI Tool:      %s\n", cfg.CLI.Tool)
			fmt.Printf("  Max Attempts:  %d\n", cfg.Execution.MaxAttempts)
			fmt.Printf("  Auto Commit:   %v\n", cfg.Execution.AutoCommit)

			return nil
		},
	}
}
