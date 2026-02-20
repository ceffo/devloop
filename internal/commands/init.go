package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/ui"
	"github.com/spf13/cobra"
)

// InitCmd creates the devloop init command
func InitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize devloop in the current project",
		Long: `Initialize devloop in the current project by creating the .devloop directory structure
and configuration file.

This command will:
  - Create .devloop/ directory with subdirectories (logs/, archive/, state/)
  - Generate config.json with auto-detected project settings
  - Create empty tasks.jsonl file
  - Detect project metadata (name, path, tech stack)

Example:
  devloop init`,
		RunE: runInit,
	}
}

func runInit(cmd *cobra.Command, _ []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	devloopDir := filepath.Join(cwd, ".devloop")

	// Detect project settings
	projectName := detectProjectName(cwd)
	techStack := detectTechStack(cwd)
	configPath := filepath.Join(devloopDir, "config.json")

	if dryRun {
		fmt.Println("[dry-run] Would initialize devloop in the current project")
		fmt.Printf("[dry-run] Would create directory: %s\n", devloopDir)
		fmt.Printf("[dry-run] Would create subdirectories: logs/, archive/, state/\n")
		fmt.Printf("[dry-run] Would write config to: %s\n", configPath)
		fmt.Printf("[dry-run] Would write tasks file: %s\n", filepath.Join(devloopDir, "tasks.jsonl"))
		fmt.Printf("[dry-run] Detected project name: %s\n", projectName)
		if techStack != "" {
			fmt.Printf("[dry-run] Detected tech stack: %s\n", techStack)
		}
		fmt.Println("[dry-run] No files were created")
		return nil
	}

	// Check if .devloop/ already exists
	if _, err := os.Stat(devloopDir); err == nil {
		// Directory exists, prompt user
		fmt.Printf("⚠  .devloop/ directory already exists in %s\n", cwd)
		fmt.Print("Do you want to overwrite it? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read user input: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Initialization cancelled.")
			return nil
		}
	}

	// Initialize project with spinner
	spinner := ui.NewSpinner("Initializing project...")
	spinner.Start()

	// Create directory structure
	if err := createDirectoryStructure(devloopDir); err != nil {
		spinner.Stop()
		return err
	}

	// Generate config with detected values
	cfg := createInitialConfig(projectName, cwd, techStack)

	// Save config
	if err := config.SaveConfig(configPath, cfg); err != nil {
		spinner.Stop()
		return err
	}

	// Create empty tasks.jsonl
	tasksPath := filepath.Join(devloopDir, "tasks.jsonl")
	if err := os.WriteFile(tasksPath, []byte(""), 0644); err != nil {
		spinner.Stop()
		return fmt.Errorf("failed to create tasks.jsonl: %w", err)
	}

	spinner.Stop()

	// Initialize beads backend if configured
	if cfg.Storage.Backend == "beads" {
		if err := initBeadsBackend(cwd); err != nil {
			fmt.Printf("Warning: %v\n", err)
		}
	}

	// Initialize mulch knowledge backend if configured
	if cfg.Knowledge.Backend == "mulch" {
		if err := initMulchBackend(cwd, cfg); err != nil {
			fmt.Printf("Warning: %v\n", err)
		}
	}

	// Success message
	fmt.Println(ui.Success("devloop initialized successfully!"))
	fmt.Printf("  Project: %s\n", projectName)
	fmt.Printf("  Path: %s\n", cwd)
	if techStack != "" {
		fmt.Printf("  Tech stack: %s\n", techStack)
	}
	fmt.Printf("\nConfiguration saved to: %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review and edit config.json if needed")
	fmt.Println("  2. Run 'devloop config validate' to check configuration")
	fmt.Println("  3. Process TODO items with 'devloop todo process <file>'")

	return nil
}

// createDirectoryStructure creates the .devloop directory and subdirectories
func createDirectoryStructure(devloopDir string) error {
	// Create main .devloop directory
	if err := os.MkdirAll(devloopDir, 0755); err != nil {
		return fmt.Errorf("failed to create .devloop directory: %w", err)
	}

	// Create subdirectories
	subdirs := []string{"logs", "archive", "state"}
	for _, subdir := range subdirs {
		path := filepath.Join(devloopDir, subdir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", subdir, err)
		}
	}

	return nil
}

// detectProjectName extracts the project name from the directory name
func detectProjectName(cwd string) string {
	return filepath.Base(cwd)
}

// detectTechStack attempts to detect the project's tech stack
func detectTechStack(cwd string) string {
	// Check for common project files
	checks := map[string]string{
		"package.json":     "Node.js",
		"go.mod":           "Go",
		"Cargo.toml":       "Rust",
		"pom.xml":          "Java (Maven)",
		"build.gradle":     "Java (Gradle)",
		"requirements.txt": "Python",
		"Pipfile":          "Python (Pipenv)",
		"pyproject.toml":   "Python",
		"Gemfile":          "Ruby",
		"composer.json":    "PHP",
		"mix.exs":          "Elixir",
		"Package.swift":    "Swift",
	}

	var detected []string
	for file, stack := range checks {
		if _, err := os.Stat(filepath.Join(cwd, file)); err == nil {
			detected = append(detected, stack)
		}
	}

	if len(detected) == 0 {
		return ""
	}

	// Return the first detected stack, or combine if multiple
	if len(detected) == 1 {
		return detected[0]
	}

	return strings.Join(detected, " + ")
}

// createInitialConfig generates a config with detected project settings
func createInitialConfig(name, path, techStack string) *config.Config {
	cfg := &config.Config{
		Version: "1.0",
		Project: config.ProjectConfig{
			Name:       name,
			Path:       path,
			TechStack:  techStack,
			MainBranch: "main",
		},
		Verification: config.VerificationConfig{
			Command:        detectVerificationCommand(path),
			TimeoutSeconds: 300,
		},
		CLI: config.CLIConfig{
			Agents: map[string]*config.AgentConfig{
				"claude": {
					Tool: "claude",
					Models: map[string]string{
						"simple":   "claude-haiku-4-5-20251001",
						"moderate": "claude-sonnet-4-5-20250929",
						"complex":  "claude-opus-4-6",
					},
				},
				"copilot": {
					Tool: "copilot",
					Models: map[string]string{
						"simple":   "gpt-5-mini",
						"moderate": "gpt-5.1",
						"complex":  "gpt-5.2",
					},
				},
			},
		},
		Execution: config.ExecutionConfig{
			MaxAttempts:   2,
			HaltOnFailure: true,
			AutoCommit:    true,
			CommitFormat:  "task {task_id}: {title}",
		},
		Archival: config.ArchivalConfig{
			AutoArchive: false,
		},
		Prompts: config.PromptsConfig{
			CustomInstructions: "",
		},
	}

	return cfg
}

// mulchProvider maps a devloop agent tool name to a Mulch provider name.
// Returns ("", false) if the tool has no known Mulch provider mapping.
func mulchProvider(tool string) (string, bool) {
	providers := map[string]string{
		"claude":  "claude",
		"copilot": "codex",
	}
	p, ok := providers[tool]
	return p, ok
}

// initMulchBackend checks for the mulch binary, runs mulch init, mulch add for each
// configured domain, and mulch setup for the mapped agent tool provider.
// If mulch is not found, it prints installation instructions.
func initMulchBackend(cwd string, cfg *config.Config) error {
	if _, err := exec.LookPath("mulch"); err != nil {
		fmt.Println("\nMulch knowledge backend selected, but 'mulch' is not installed.")
		fmt.Println("Install mulch using:")
		fmt.Println("  npm install -g mulch-cli")
		fmt.Println("\nAfter installing, run 'mulch init' in your project directory.")
		return fmt.Errorf("mulch not found in PATH")
	}

	fmt.Println("\nInitializing Mulch knowledge backend...")
	cmd := exec.Command("mulch", "init")
	cmd.Dir = cwd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mulch init failed: %w", err)
	}

	for _, domain := range cfg.Knowledge.Domains {
		cmd := exec.Command("mulch", "add", domain)
		cmd.Dir = cwd
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("mulch add %s failed: %w", domain, err)
		}
	}

	// Map each configured agent tool to a Mulch provider and run mulch setup.
	seen := map[string]bool{}
	anyUnmapped := false
	for _, agentCfg := range cfg.CLI.Agents {
		provider, ok := mulchProvider(agentCfg.Tool)
		if !ok {
			anyUnmapped = true
			continue
		}
		if seen[provider] {
			continue
		}
		seen[provider] = true
		cmd := exec.Command("mulch", "setup", provider)
		cmd.Dir = cwd
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("mulch setup %s failed: %w", provider, err)
		}
	}
	if anyUnmapped {
		fmt.Println("Note: one or more agent tools have no Mulch provider mapping; skipping mulch setup for them.")
		fmt.Println("      You can run 'mulch setup <provider>' manually.")
	}

	return nil
}

// initBeadsBackend checks for the bd binary and runs bd init in the project directory.
// If bd is not found, it prints installation instructions.
func initBeadsBackend(cwd string) error {
	if _, err := exec.LookPath("bd"); err != nil {
		fmt.Println("\nBeads storage backend selected, but 'bd' is not installed.")
		fmt.Println("Install bd using one of the following methods:")
		fmt.Println("  go install  github.com/beads-db/bd@latest")
		fmt.Println("  npm install -g @beads-db/bd")
		fmt.Println("  brew install beads-db/tap/bd")
		fmt.Println("\nAfter installing, run 'bd init' in your project directory.")
		fmt.Println("Tip: 'bd init --stealth' hides the .beads directory from git.")
		return fmt.Errorf("bd not found in PATH")
	}

	fmt.Println("\nInitializing Beads storage backend...")
	cmd := exec.Command("bd", "init")
	cmd.Dir = cwd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bd init failed: %w", err)
	}

	fmt.Println("Tip: use 'bd init --stealth' to initialize in stealth mode (hides .beads directory from git)")
	return nil
}

// detectVerificationCommand attempts to detect an appropriate verification command
func detectVerificationCommand(cwd string) string {
	// Check for common project files and suggest verification commands
	checks := []struct {
		file    string
		command string
	}{
		{"package.json", "npm run build && npm test"},
		{"go.mod", "go test ./... && go build ./..."},
		{"Cargo.toml", "cargo test && cargo build"},
		{"pom.xml", "mvn test"},
		{"build.gradle", "./gradlew test"},
		{"requirements.txt", "pytest"},
		{"Makefile", "make test"},
	}

	for _, check := range checks {
		if _, err := os.Stat(filepath.Join(cwd, check.file)); err == nil {
			return check.command
		}
	}

	// Default to empty string - user will need to configure
	return ""
}
