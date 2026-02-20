// Package main is the entry point for the devloop CLI.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ceffo/devloop/internal/commands"
	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	configPath string
	verbose    bool
	agentFlag  string
	dryRun     bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "devloop",
	Short:   "Agent-driven development workflow system",
	Long:    `devloop is a project-agnostic tool for automating development workflows using AI agents. It replaces brittle bash scripts with a structured, queryable, crash-safe system for managing and executing development tasks.`,
	Version: "0.2.0",
}

// initCmd represents the init command
var initCmd = commands.InitCmd()

// configCmd represents the config command
var configCmd = commands.ConfigCmd()

// processCmd represents the process command
var processCmd = commands.ProcessCmd()

// runCmd represents the run command
var runCmd = commands.RunCmd()

// tasksCmd represents the tasks command
var tasksCmd = commands.TasksCmd()

// archiveCmd represents the archive command
var archiveCmd = commands.ArchiveCmd()

// sessionCmd represents the session command
var sessionCmd = commands.SessionCmd()

// resumeCmd represents the resume command
var resumeCmd = commands.ResumeCmd()

// knowledgeCmd represents the knowledge command
var knowledgeCmd = commands.KnowledgeCmd()

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&configPath, "config", ".devloop/config.json", "path to configuration file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().StringVar(&agentFlag, "agent", "", "agent name from config to use (default: first in list)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would be done without performing any side effects")

	// Add subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(processCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(tasksCmd)
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(sessionCmd)
	rootCmd.AddCommand(knowledgeCmd)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	err := fang.Execute(ctx, rootCmd, fang.WithVersion("0.2.0"))
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
