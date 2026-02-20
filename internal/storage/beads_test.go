package storage

import (
	"errors"
	"testing"

	"github.com/ceffo/devloop/internal/config"
)

func TestNewBeadsStore_Success(t *testing.T) {
	cfg := &config.Config{
		Version: "1.0",
		Project: config.ProjectConfig{
			Name:       "test-project",
			Path:       "/tmp/test",
			TechStack:  "Go",
			MainBranch: "main",
		},
	}

	// Mock lookPathFunc to simulate bd being found
	originalLookPath := lookPathFunc
	lookPathFunc = func(file string) (string, error) {
		return "/usr/local/bin/bd", nil
	}
	defer func() {
		lookPathFunc = originalLookPath
	}()

	store, err := NewBeadsStore(cfg)
	if err != nil {
		t.Fatalf("NewBeadsStore should succeed when bd is available: %v", err)
	}

	if store == nil {
		t.Fatal("NewBeadsStore returned nil store")
	}

	if store.cfg != cfg {
		t.Error("Config not properly stored in BeadsStore struct")
	}

	if store.bdPath != "/usr/local/bin/bd" {
		t.Errorf("bdPath should be set to resolved binary path, got %s", store.bdPath)
	}
}

func TestNewBeadsStore_BdNotFound(t *testing.T) {
	cfg := &config.Config{
		Version: "1.0",
		Project: config.ProjectConfig{
			Name:       "test-project",
			Path:       "/tmp/test",
			TechStack:  "Go",
			MainBranch: "main",
		},
	}

	// Mock lookPathFunc to simulate bd not being found
	originalLookPath := lookPathFunc
	lookPathFunc = func(file string) (string, error) {
		return "", errors.New("executable file not found in $PATH")
	}
	defer func() {
		lookPathFunc = originalLookPath
	}()

	store, err := NewBeadsStore(cfg)
	if err == nil {
		t.Fatal("NewBeadsStore should fail when bd is not found")
	}

	if store != nil {
		t.Error("NewBeadsStore should return nil store when bd is not found")
	}

	// Verify error message includes all three install options
	errMsg := err.Error()
	if !contains(errMsg, "go install") {
		t.Error("Error message should include go install option")
	}
	if !contains(errMsg, "npm") {
		t.Error("Error message should include npm option")
	}
	if !contains(errMsg, "brew") {
		t.Error("Error message should include brew option")
	}
}

// Helper function to check if string contains substring
func contains(str, substr string) bool {
	for i := 0; i < len(str)-len(substr)+1; i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
