package commands

import (
	"testing"
)

func TestCheckMulch_NotFound(t *testing.T) {
	t.Setenv("PATH", "")

	err := checkMulch()
	if err == nil {
		t.Fatal("expected error when mulch is not in PATH, got nil")
	}
	if err.Error() != "mulch not found in PATH" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestKnowledgeCmd_Structure(t *testing.T) {
	cmd := KnowledgeCmd()

	if cmd.Name() != "knowledge" {
		t.Errorf("Name() = %q, want %q", cmd.Name(), "knowledge")
	}

	subCmdNames := map[string]bool{}
	for _, sub := range cmd.Commands() {
		subCmdNames[sub.Name()] = true
	}

	expected := []string{"status", "query", "search", "compact", "doctor", "diff", "prime"}
	for _, name := range expected {
		if !subCmdNames[name] {
			t.Errorf("subcommand %q not found in knowledge command", name)
		}
	}
}

func TestKnowledgeSearchCmd_RequiresArg(t *testing.T) {
	cmd := knowledgeSearchCmd()
	err := cmd.Args(cmd, []string{})
	if err == nil {
		t.Error("expected error when no args provided to search, got nil")
	}
}

func TestKnowledgeSearchCmd_AcceptsArg(t *testing.T) {
	cmd := knowledgeSearchCmd()
	err := cmd.Args(cmd, []string{"my query"})
	if err != nil {
		t.Errorf("unexpected error with valid arg: %v", err)
	}
}

func TestRunMulch_NotFound(t *testing.T) {
	t.Setenv("PATH", "")

	err := runMulch("status")
	if err == nil {
		t.Fatal("expected error when mulch is not in PATH, got nil")
	}
	if err.Error() != "mulch not found in PATH" {
		t.Errorf("unexpected error message: %v", err)
	}
}
