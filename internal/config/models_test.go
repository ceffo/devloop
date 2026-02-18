package config

import (
	"os/exec"
	"testing"
)

// sampleCopilotHelp is an excerpt of real `copilot --help` output containing the
// --model choices block, used to test parseModelsFromHelp.
const sampleCopilotHelp = `
  --model <model>                     Set the AI model to use (choices:
                                      "claude-sonnet-4.6", "claude-sonnet-4.5",
                                      "claude-haiku-4.5", "claude-opus-4.6",
                                      "claude-opus-4.6-fast", "claude-opus-4.5",
                                      "claude-sonnet-4", "gemini-3-pro-preview",
                                      "gpt-5.3-codex", "gpt-5.2-codex",
                                      "gpt-5.2", "gpt-5.1-codex-max",
                                      "gpt-5.1-codex", "gpt-5.1", "gpt-5",
                                      "gpt-5.1-codex-mini", "gpt-5-mini",
                                      "gpt-4.1")
  --no-alt-screen                     Disable the terminal alternate screen
`

// sampleClaudeHelp is an excerpt of real `claude --help` output; notably the
// --model flag does not enumerate choices, but a later flag does — the parser
// must not bleed across flag boundaries.
const sampleClaudeHelp = `
  --model <model>                     Model for the current session. Provide an
                                      alias for the latest model (e.g. 'sonnet'
                                      or 'opus') or a model's full name (e.g.
                                      'claude-sonnet-4-6').
  --output-format <format>            Output format (choices: "text", "json", "stream-json")
  --permission-mode <mode>            Permission mode (choices: "acceptEdits", "bypassPermissions")
`

func TestParseModelsFromHelp_WithChoices(t *testing.T) {
	models := parseModelsFromHelp(sampleCopilotHelp)
	if models == nil {
		t.Fatal("expected non-nil model map for help text with choices")
	}

	want := []string{
		"claude-sonnet-4.6", "claude-haiku-4.5", "claude-opus-4.6",
		"gpt-5", "gpt-5-mini", "gpt-4.1",
	}
	for _, m := range want {
		if !models[m] {
			t.Errorf("expected model %q to be present", m)
		}
	}
}

func TestParseModelsFromHelp_WithoutChoices(t *testing.T) {
	models := parseModelsFromHelp(sampleClaudeHelp)
	if models != nil {
		t.Errorf("expected nil for help text without --model choices, got %v", models)
	}
}

// TestParseModelsFromHelp_ScopedToModelFlag ensures that choices belonging to a
// *later* flag (e.g. --output-format) are not mistakenly attributed to --model.
func TestParseModelsFromHelp_ScopedToModelFlag(t *testing.T) {
	models := parseModelsFromHelp(sampleClaudeHelp)
	if models != nil {
		// "text", "json", etc. must NOT appear as valid models.
		for _, bad := range []string{"text", "json", "stream-json", "acceptEdits", "bypassPermissions"} {
			if models[bad] {
				t.Errorf("output-format / permission-mode choice %q should not be treated as a model", bad)
			}
		}
	}
}

func TestParseModelsFromHelp_Empty(t *testing.T) {
	if got := parseModelsFromHelp(""); got != nil {
		t.Errorf("expected nil for empty input, got %v", got)
	}
}

func TestParseModelsFromHelp_NoModelFlag(t *testing.T) {
	if got := parseModelsFromHelp("--verbose\n--output\n"); got != nil {
		t.Errorf("expected nil when --model flag absent, got %v", got)
	}
}

func TestFetchModels_BinaryNotFound(t *testing.T) {
	models, err := FetchModels("__nonexistent_binary_xyz__")
	if err != nil {
		t.Fatalf("expected no error for missing binary, got: %v", err)
	}
	if models != nil {
		t.Errorf("expected nil model map for missing binary, got %v", models)
	}
}

func TestFetchModels_CopilotBinary(t *testing.T) {
	if _, err := exec.LookPath("copilot"); err != nil {
		t.Skip("copilot binary not in PATH, skipping")
	}

	models, err := FetchModels("copilot")
	if err != nil {
		t.Fatalf("FetchModels(copilot) returned unexpected error: %v", err)
	}
	if models == nil {
		t.Fatal("expected non-nil model map from copilot --help")
	}
	if len(models) == 0 {
		t.Error("expected at least one model from copilot --help")
	}
}

func TestFetchModels_ClaudeBinary(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude binary not in PATH, skipping")
	}

	// claude does not enumerate model choices — FetchModels must return nil
	// without an error.
	models, err := FetchModels("claude")
	if err != nil {
		t.Fatalf("FetchModels(claude) returned unexpected error: %v", err)
	}
	// nil is expected (no choices block), but a non-nil map is also acceptable
	// if a future version of claude starts listing choices.
	_ = models
}
