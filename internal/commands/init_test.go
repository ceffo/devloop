package commands

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectProjectName(t *testing.T) {
	tests := []struct {
		name     string
		cwd      string
		expected string
	}{
		{
			name:     "simple path",
			cwd:      "/home/user/myproject",
			expected: "myproject",
		},
		{
			name:     "nested path",
			cwd:      "/home/user/projects/awesome-app",
			expected: "awesome-app",
		},
		{
			name:     "root directory",
			cwd:      "/",
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectProjectName(tt.cwd)
			if result != tt.expected {
				t.Errorf("detectProjectName(%q) = %q, want %q", tt.cwd, result, tt.expected)
			}
		})
	}
}

func TestDetectTechStack(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{
			name:     "Go project",
			files:    []string{"go.mod"},
			expected: "Go",
		},
		{
			name:     "Node.js project",
			files:    []string{"package.json"},
			expected: "Node.js",
		},
		{
			name:     "Python project",
			files:    []string{"requirements.txt"},
			expected: "Python",
		},
		{
			name:     "mixed project",
			files:    []string{"go.mod", "package.json"},
			expected: "Go + Node.js",
		},
		{
			name:     "no tech stack detected",
			files:    []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tempDir, err := os.MkdirTemp("", "devloop-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create test files
			for _, file := range tt.files {
				path := filepath.Join(tempDir, file)
				if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to create test file %s: %v", file, err)
				}
			}

			result := detectTechStack(tempDir)

			// For multiple stacks, order may vary, so we check if both are present
			if tt.name == "mixed project" {
				if !contains(result, "Go") || !contains(result, "Node.js") {
					t.Errorf("detectTechStack() = %q, want to contain both 'Go' and 'Node.js'", result)
				}
			} else if result != tt.expected {
				t.Errorf("detectTechStack() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDetectVerificationCommand(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{
			name:     "Go project",
			files:    []string{"go.mod"},
			expected: "go test ./... && go build ./...",
		},
		{
			name:     "Node.js project",
			files:    []string{"package.json"},
			expected: "npm run build && npm test",
		},
		{
			name:     "Makefile project",
			files:    []string{"Makefile"},
			expected: "make test",
		},
		{
			name:     "no known files",
			files:    []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tempDir, err := os.MkdirTemp("", "devloop-test-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create test files
			for _, file := range tt.files {
				path := filepath.Join(tempDir, file)
				if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to create test file %s: %v", file, err)
				}
			}

			result := detectVerificationCommand(tempDir)
			if result != tt.expected {
				t.Errorf("detectVerificationCommand() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCreateDirectoryStructure(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "devloop-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	devloopDir := filepath.Join(tempDir, ".devloop")

	// Create directory structure
	if err := createDirectoryStructure(devloopDir); err != nil {
		t.Fatalf("createDirectoryStructure() failed: %v", err)
	}

	// Verify main directory exists
	if _, err := os.Stat(devloopDir); os.IsNotExist(err) {
		t.Error(".devloop directory was not created")
	}

	// Verify subdirectories exist
	subdirs := []string{"logs", "archive", "state"}
	for _, subdir := range subdirs {
		path := filepath.Join(devloopDir, subdir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("subdirectory %s was not created", subdir)
		}
	}
}

func TestCreateInitialConfig(t *testing.T) {
	name := "testproject"
	path := "/home/user/testproject"
	techStack := "Go"

	cfg := createInitialConfig(name, path, techStack)

	if cfg.Project.Name != name {
		t.Errorf("Project.Name = %q, want %q", cfg.Project.Name, name)
	}

	if cfg.Project.Path != path {
		t.Errorf("Project.Path = %q, want %q", cfg.Project.Path, path)
	}

	if cfg.Project.TechStack != techStack {
		t.Errorf("Project.TechStack = %q, want %q", cfg.Project.TechStack, techStack)
	}

	if cfg.Version != "1.0" {
		t.Errorf("Version = %q, want %q", cfg.Version, "1.0")
	}

	if cfg.Project.MainBranch != "main" {
		t.Errorf("Project.MainBranch = %q, want %q", cfg.Project.MainBranch, "main")
	}

	if cfg.Execution.MaxAttempts != 2 {
		t.Errorf("Execution.MaxAttempts = %d, want %d", cfg.Execution.MaxAttempts, 2)
	}

	if len(cfg.CLI.Agents) != 2 {
		t.Errorf("CLI.Agents length = %d, want 2", len(cfg.CLI.Agents))
	}

	if _, exists := cfg.CLI.Agents["claude"]; !exists {
		t.Error("claude agent not found in CLI.Agents")
	}

	if _, exists := cfg.CLI.Agents["copilot"]; !exists {
		t.Error("copilot agent not found in CLI.Agents")
	}
}

func TestInitBeadsBackend_BdNotFound(t *testing.T) {
	// Override PATH so bd is not found
	t.Setenv("PATH", "")

	tempDir, err := os.MkdirTemp("", "devloop-beads-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	err = initBeadsBackend(tempDir)
	if err == nil {
		t.Fatal("expected error when bd is not in PATH, got nil")
	}
	if err.Error() != "bd not found in PATH" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInitMulchBackend_MulchNotFound(t *testing.T) {
	// Override PATH so mulch is not found
	t.Setenv("PATH", "")

	tempDir, err := os.MkdirTemp("", "devloop-mulch-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	err = initMulchBackend(tempDir, nil)
	if err == nil {
		t.Fatal("expected error when mulch is not in PATH, got nil")
	}
	if err.Error() != "mulch not found in PATH" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMulchProvider(t *testing.T) {
	tests := []struct {
		tool     string
		provider string
		ok       bool
	}{
		{"claude", "claude", true},
		{"copilot", "codex", true},
		{"cursor", "", false},
		{"unknown", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		p, ok := mulchProvider(tt.tool)
		if ok != tt.ok || p != tt.provider {
			t.Errorf("mulchProvider(%q) = (%q, %v), want (%q, %v)", tt.tool, p, ok, tt.provider, tt.ok)
		}
	}
}

func TestPromptExtendedSetupSkip(t *testing.T) {
	for _, input := range []string{"n\n", "N\n", "\n", "no\n"} {
		reader := bufio.NewReader(strings.NewReader(input))
		opts, err := promptExtendedSetup(reader)
		if err != nil {
			t.Fatalf("input %q: unexpected error: %v", input, err)
		}
		if opts != nil {
			t.Errorf("input %q: expected nil opts, got %+v", input, opts)
		}
	}
}

func TestPromptExtendedSetupDefaults(t *testing.T) {
	// y → enter (keep jsonl) → enter (keep none) → enter (no coordinator)
	reader := bufio.NewReader(strings.NewReader("y\n\n\n\n"))
	opts, err := promptExtendedSetup(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts == nil {
		t.Fatal("expected non-nil opts")
	}
	if opts.StorageBackend != "jsonl" {
		t.Errorf("StorageBackend = %q, want %q", opts.StorageBackend, "jsonl")
	}
	if opts.KnowledgeBackend != "none" {
		t.Errorf("KnowledgeBackend = %q, want %q", opts.KnowledgeBackend, "none")
	}
	if opts.EnableCoordinator {
		t.Error("expected EnableCoordinator to be false")
	}
	if len(opts.Domains) != 0 {
		t.Errorf("Domains = %v, want empty", opts.Domains)
	}
}

func TestPromptExtendedSetupBeads(t *testing.T) {
	// y → beads → none → y (enable coordinator)
	reader := bufio.NewReader(strings.NewReader("y\nbeads\nnone\ny\n"))
	opts, err := promptExtendedSetup(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.StorageBackend != "beads" {
		t.Errorf("StorageBackend = %q, want %q", opts.StorageBackend, "beads")
	}
	if opts.KnowledgeBackend != "none" {
		t.Errorf("KnowledgeBackend = %q, want %q", opts.KnowledgeBackend, "none")
	}
	if !opts.EnableCoordinator {
		t.Error("expected EnableCoordinator to be true")
	}
}

func TestPromptExtendedSetupMulch(t *testing.T) {
	// y → jsonl → mulch → go,testing → n (no coordinator)
	reader := bufio.NewReader(strings.NewReader("y\njsonl\nmulch\ngo,testing\nn\n"))
	opts, err := promptExtendedSetup(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.KnowledgeBackend != "mulch" {
		t.Errorf("KnowledgeBackend = %q, want %q", opts.KnowledgeBackend, "mulch")
	}
	if len(opts.Domains) != 2 || opts.Domains[0] != "go" || opts.Domains[1] != "testing" {
		t.Errorf("Domains = %v, want [go testing]", opts.Domains)
	}
	if opts.EnableCoordinator {
		t.Error("expected EnableCoordinator to be false")
	}
}

func TestPromptExtendedSetupMulchNoDomains(t *testing.T) {
	// y → jsonl → mulch → (empty domains) → n
	reader := bufio.NewReader(strings.NewReader("y\njsonl\nmulch\n\nn\n"))
	opts, err := promptExtendedSetup(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.KnowledgeBackend != "mulch" {
		t.Errorf("KnowledgeBackend = %q, want %q", opts.KnowledgeBackend, "mulch")
	}
	if len(opts.Domains) != 0 {
		t.Errorf("Domains = %v, want empty", opts.Domains)
	}
}

func TestApplyExtendedSetupNil(t *testing.T) {
	cfg := createInitialConfig("test", "/test", "Go")
	// Should be a no-op
	applyExtendedSetup(cfg, nil)
	// Existing agents should remain unchanged
	if len(cfg.CLI.Agents) == 0 {
		t.Error("CLI.Agents should not be empty after nil apply")
	}
}

func TestApplyExtendedSetupBeadsCoordinator(t *testing.T) {
	cfg := createInitialConfig("test", "/test", "Go")
	opts := &extendedSetupOptions{
		StorageBackend:    "beads",
		KnowledgeBackend:  "none",
		EnableCoordinator: true,
	}
	applyExtendedSetup(cfg, opts)
	if cfg.Storage.Backend != "beads" {
		t.Errorf("Storage.Backend = %q, want %q", cfg.Storage.Backend, "beads")
	}
	if cfg.Knowledge.Backend != "none" {
		t.Errorf("Knowledge.Backend = %q, want %q", cfg.Knowledge.Backend, "none")
	}
	if !cfg.Execution.Coordinator.Enabled {
		t.Error("expected Coordinator.Enabled to be true")
	}
}

func TestApplyExtendedSetupMulchDomains(t *testing.T) {
	cfg := createInitialConfig("test", "/test", "Go")
	opts := &extendedSetupOptions{
		StorageBackend:   "jsonl",
		KnowledgeBackend: "mulch",
		Domains:          []string{"go", "testing"},
	}
	applyExtendedSetup(cfg, opts)
	if cfg.Knowledge.Backend != "mulch" {
		t.Errorf("Knowledge.Backend = %q, want %q", cfg.Knowledge.Backend, "mulch")
	}
	if len(cfg.Knowledge.Domains) != 2 {
		t.Errorf("Knowledge.Domains = %v, want [go testing]", cfg.Knowledge.Domains)
	}
	if !cfg.Knowledge.InjectOnExecute {
		t.Error("expected Knowledge.InjectOnExecute to be true")
	}
	if !cfg.Knowledge.RecordOnComplete {
		t.Error("expected Knowledge.RecordOnComplete to be true")
	}
	if cfg.Knowledge.PrimeBudget != 2000 {
		t.Errorf("Knowledge.PrimeBudget = %d, want 2000", cfg.Knowledge.PrimeBudget)
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
