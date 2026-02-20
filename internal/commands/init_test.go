package commands

import (
	"os"
	"path/filepath"
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

// Helper function for string contains check
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
