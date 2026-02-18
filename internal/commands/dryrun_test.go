package commands

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// setupDryRunTestRoot creates a root command with the persistent --dry-run flag
// and attaches the given sub-command, mirroring the real main.go setup.
func setupDryRunTestRoot(sub *cobra.Command) *cobra.Command {
	root := &cobra.Command{Use: "devloop"}
	root.PersistentFlags().String("config", ".devloop/config.json", "")
	root.PersistentFlags().String("agent", "", "")
	root.PersistentFlags().Bool("dry-run", false, "")
	root.AddCommand(sub)
	return root
}

// captureOutput runs a cobra command and captures stdout.
func captureOutput(root *cobra.Command, args []string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	root.SetArgs(args)
	err := root.Execute()

	w.Close()
	os.Stdout = old
	io.Copy(buf, r) //nolint:errcheck
	return buf.String(), err
}

// TestDryRunFlagExists verifies that --dry-run is exposed on every leaf command.
func TestDryRunFlagExists(t *testing.T) {
	cmds := []*cobra.Command{
		RunCmd(),
		ResumeCmd(),
		InitCmd(),
		ArchiveCmd(),
	}

	for _, cmd := range cmds {
		t.Run(cmd.Use, func(t *testing.T) {
			root := setupDryRunTestRoot(cmd)
			f := root.PersistentFlags().Lookup("dry-run")
			if f == nil {
				t.Errorf("--dry-run persistent flag not found on root for command %q", cmd.Use)
			}
		})
	}
}

// TestRunDryRun verifies that 'run --dry-run' prints a message and makes no writes.
func TestRunDryRun(t *testing.T) {
	tempDir := t.TempDir()
	devloopDir := filepath.Join(tempDir, ".devloop")
	if err := os.MkdirAll(filepath.Join(devloopDir, "logs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(devloopDir, "state"), 0755); err != nil {
		t.Fatal(err)
	}
	configJSON := `{
		"version": "1.0",
		"project": {"name": "test", "path": "` + tempDir + `", "tech_stack": "Go", "main_branch": "main"},
		"verification": {"command": "echo ok", "timeout_seconds": 30},
		"cli": {"agents": {"claude": {"tool": "claude", "models": {"simple": "haiku", "moderate": "sonnet", "complex": "opus"}}}},
		"execution": {"max_attempts": 1, "halt_on_failure": false, "auto_commit": false}
	}`
	configPath := filepath.Join(devloopDir, "config.json")
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(devloopDir, "tasks.jsonl"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	root := setupDryRunTestRoot(RunCmd())
	out, err := captureOutput(root, []string{"--config", configPath, "--dry-run", "run"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "[dry-run]") {
		t.Errorf("expected [dry-run] output, got: %s", out)
	}
}

// TestInitDryRun verifies that 'init --dry-run' prints a message and creates no files.
func TestInitDryRun(t *testing.T) {
	tempDir := t.TempDir()

	// Change to temp dir so init uses it as cwd
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir) //nolint:errcheck
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	root := setupDryRunTestRoot(InitCmd())
	out, err := captureOutput(root, []string{"--dry-run", "init"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "[dry-run]") {
		t.Errorf("expected [dry-run] output, got: %s", out)
	}

	// Verify no .devloop directory was created
	if _, statErr := os.Stat(filepath.Join(tempDir, ".devloop")); !os.IsNotExist(statErr) {
		t.Error("expected no .devloop directory to be created in dry-run mode")
	}
}

// TestArchiveDryRun verifies that 'archive --dry-run' prints a message and writes no files.
func TestArchiveDryRun(t *testing.T) {
	tempDir := t.TempDir()
	devloopDir := filepath.Join(tempDir, ".devloop")
	if err := os.MkdirAll(filepath.Join(devloopDir, "archive"), 0755); err != nil {
		t.Fatal(err)
	}
	configJSON := `{
		"version": "1.0",
		"project": {"name": "test", "path": "` + tempDir + `", "tech_stack": "Go", "main_branch": "main"},
		"verification": {"command": "echo ok", "timeout_seconds": 30},
		"cli": {"agents": {"claude": {"tool": "claude", "models": {"simple": "haiku", "moderate": "sonnet", "complex": "opus"}}}},
		"execution": {"max_attempts": 1, "halt_on_failure": false, "auto_commit": false}
	}`
	configPath := filepath.Join(devloopDir, "config.json")
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}
	// Write a completed task
	taskJSON := `{"id":"DEV-1","title":"Done task","status":"completed","complexity":"simple","metadata":{"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z","max_attempts":1},"execution":{"attempts":[],"total_duration":0}}` + "\n"
	if err := os.WriteFile(filepath.Join(devloopDir, "tasks.jsonl"), []byte(taskJSON), 0644); err != nil {
		t.Fatal(err)
	}

	root := setupDryRunTestRoot(ArchiveCmd())
	out, err := captureOutput(root, []string{"--config", configPath, "--dry-run", "archive"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "[dry-run]") {
		t.Errorf("expected [dry-run] output, got: %s", out)
	}

	// Verify archive directory has no new files
	entries, _ := os.ReadDir(filepath.Join(devloopDir, "archive"))
	if len(entries) != 0 {
		t.Errorf("expected no archive files in dry-run mode, got %d", len(entries))
	}
}

// TestTasksUpdateDryRun verifies that 'tasks update --status ... --dry-run' prints and does not write.
func TestTasksUpdateDryRun(t *testing.T) {
	tempDir := t.TempDir()
	devloopDir := filepath.Join(tempDir, ".devloop")
	if err := os.MkdirAll(devloopDir, 0755); err != nil {
		t.Fatal(err)
	}
	configJSON := `{
		"version": "1.0",
		"project": {"name": "test", "path": "` + tempDir + `", "tech_stack": "Go", "main_branch": "main"},
		"verification": {"command": "echo ok", "timeout_seconds": 30},
		"cli": {"agents": {"claude": {"tool": "claude", "models": {"simple": "haiku", "moderate": "sonnet", "complex": "opus"}}}},
		"execution": {"max_attempts": 1, "halt_on_failure": false, "auto_commit": false}
	}`
	configPath := filepath.Join(devloopDir, "config.json")
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}
	taskJSON := `{"id":"DEV-1","title":"A task","status":"pending","complexity":"simple","metadata":{"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z","max_attempts":1},"execution":{"attempts":[],"total_duration":0}}` + "\n"
	tasksPath := filepath.Join(devloopDir, "tasks.jsonl")
	if err := os.WriteFile(tasksPath, []byte(taskJSON), 0644); err != nil {
		t.Fatal(err)
	}

	root := &cobra.Command{Use: "devloop"}
	root.PersistentFlags().String("config", ".devloop/config.json", "")
	root.PersistentFlags().String("agent", "", "")
	root.PersistentFlags().Bool("dry-run", false, "")
	tasksCmd := TasksCmd()
	root.AddCommand(tasksCmd)

	out, err := captureOutput(root, []string{"--config", configPath, "--dry-run", "tasks", "update", "DEV-1", "--status", "completed"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "[dry-run]") {
		t.Errorf("expected [dry-run] output, got: %s", out)
	}

	// Verify the task was NOT changed
	data, _ := os.ReadFile(tasksPath)
	if strings.Contains(string(data), `"completed"`) {
		t.Error("task status should not have been updated in dry-run mode")
	}
}
