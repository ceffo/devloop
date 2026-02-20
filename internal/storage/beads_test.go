package storage

import (
	"context"
	"errors"
	"os/exec"
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

// newTestStore returns a BeadsStore with a fake bdPath, bypassing LookPath.
func newTestStore() *BeadsStore {
	return &BeadsStore{
		cfg:    &config.Config{},
		bdPath: "/usr/local/bin/bd",
	}
}

// mockExecCommand returns a function that captures args and runs the given shell command.
func mockExecCommand(shellCmd string, capturedArgs *[][]string) func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if capturedArgs != nil {
			*capturedArgs = append(*capturedArgs, append([]string{name}, args...))
		}
		return exec.CommandContext(ctx, "sh", "-c", shellCmd)
	}
}

// --- resolveBeadsID tests ---

func TestResolveBeadsID_AlreadyBeadsID(t *testing.T) {
	store := newTestStore()

	// Override execCommandContextFunc to detect any unexpected call
	orig := execCommandContextFunc
	called := false
	execCommandContextFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		called = true
		return exec.CommandContext(ctx, name, args...)
	}
	defer func() { execCommandContextFunc = orig }()

	got, err := store.resolveBeadsID(context.Background(), "bd-x7f3")
	if err != nil {
		t.Fatalf("resolveBeadsID should not fail for Beads hash ID: %v", err)
	}
	if got != "bd-x7f3" {
		t.Errorf("expected %q, got %q", "bd-x7f3", got)
	}
	if called {
		t.Error("exec should not be called for native Beads hash IDs")
	}
}

func TestResolveBeadsID_DevloopID_Success(t *testing.T) {
	store := newTestStore()

	var capturedArgs [][]string
	orig := execCommandContextFunc
	execCommandContextFunc = mockExecCommand("echo bd-x7f3", &capturedArgs)
	defer func() { execCommandContextFunc = orig }()

	got, err := store.resolveBeadsID(context.Background(), "DEV-5")
	if err != nil {
		t.Fatalf("resolveBeadsID failed: %v", err)
	}
	if got != "bd-x7f3" {
		t.Errorf("expected %q, got %q", "bd-x7f3", got)
	}

	// Verify command construction: bd kv get "devloop:DEV-5"
	if len(capturedArgs) != 1 {
		t.Fatalf("expected 1 exec call, got %d", len(capturedArgs))
	}
	args := capturedArgs[0]
	if args[0] != "/usr/local/bin/bd" {
		t.Errorf("expected bd binary %q, got %q", "/usr/local/bin/bd", args[0])
	}
	if len(args) < 4 || args[1] != "kv" || args[2] != "get" || args[3] != "devloop:DEV-5" {
		t.Errorf("unexpected kv get args: %v", args[1:])
	}
}

func TestResolveBeadsID_DevloopID_EmptyOutput(t *testing.T) {
	store := newTestStore()

	orig := execCommandContextFunc
	execCommandContextFunc = mockExecCommand("echo ''", nil)
	defer func() { execCommandContextFunc = orig }()

	_, err := store.resolveBeadsID(context.Background(), "DEV-5")
	if err == nil {
		t.Fatal("resolveBeadsID should fail when output is empty")
	}
	if !contains(err.Error(), "no Beads ID found") {
		t.Errorf("expected 'no Beads ID found' in error, got: %v", err)
	}
}

func TestResolveBeadsID_DevloopID_ExecFails(t *testing.T) {
	store := newTestStore()

	orig := execCommandContextFunc
	execCommandContextFunc = mockExecCommand("exit 1", nil)
	defer func() { execCommandContextFunc = orig }()

	_, err := store.resolveBeadsID(context.Background(), "DEV-5")
	if err == nil {
		t.Fatal("resolveBeadsID should fail when bd kv get fails")
	}
	if !contains(err.Error(), "bd kv get") {
		t.Errorf("expected 'bd kv get' in error, got: %v", err)
	}
}

// --- writeIDMapping tests ---

func TestWriteIDMapping_Success(t *testing.T) {
	store := newTestStore()

	var capturedArgs [][]string
	orig := execCommandContextFunc
	execCommandContextFunc = mockExecCommand("true", &capturedArgs)
	defer func() { execCommandContextFunc = orig }()

	err := store.writeIDMapping(context.Background(), "DEV-5", "bd-x7f3")
	if err != nil {
		t.Fatalf("writeIDMapping failed: %v", err)
	}

	// Verify two exec calls were made
	if len(capturedArgs) != 2 {
		t.Fatalf("expected 2 exec calls, got %d", len(capturedArgs))
	}

	// First call: bd kv set "devloop:DEV-5" "bd-x7f3"
	first := capturedArgs[0]
	if first[0] != "/usr/local/bin/bd" {
		t.Errorf("expected bd binary, got %q", first[0])
	}
	if len(first) < 5 || first[1] != "kv" || first[2] != "set" || first[3] != "devloop:DEV-5" || first[4] != "bd-x7f3" {
		t.Errorf("unexpected first kv set args: %v", first[1:])
	}

	// Second call: bd kv set "beads:bd-x7f3" "DEV-5"
	second := capturedArgs[1]
	if len(second) < 5 || second[1] != "kv" || second[2] != "set" || second[3] != "beads:bd-x7f3" || second[4] != "DEV-5" {
		t.Errorf("unexpected second kv set args: %v", second[1:])
	}
}

func TestWriteIDMapping_FirstSetFails(t *testing.T) {
	store := newTestStore()

	callCount := 0
	orig := execCommandContextFunc
	execCommandContextFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		callCount++
		return exec.CommandContext(ctx, "sh", "-c", "exit 1")
	}
	defer func() { execCommandContextFunc = orig }()

	err := store.writeIDMapping(context.Background(), "DEV-5", "bd-x7f3")
	if err == nil {
		t.Fatal("writeIDMapping should fail when first kv set fails")
	}
	if !contains(err.Error(), "bd kv set") {
		t.Errorf("expected 'bd kv set' in error, got: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 exec call before failure, got %d", callCount)
	}
}

func TestWriteIDMapping_SecondSetFails(t *testing.T) {
	store := newTestStore()

	callCount := 0
	orig := execCommandContextFunc
	execCommandContextFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			return exec.CommandContext(ctx, "sh", "-c", "true")
		}
		return exec.CommandContext(ctx, "sh", "-c", "exit 1")
	}
	defer func() { execCommandContextFunc = orig }()

	err := store.writeIDMapping(context.Background(), "DEV-5", "bd-x7f3")
	if err == nil {
		t.Fatal("writeIDMapping should fail when second kv set fails")
	}
	if !contains(err.Error(), "bd kv set") {
		t.Errorf("expected 'bd kv set' in error, got: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 exec calls, got %d", callCount)
	}
}

// --- devloopStatusToBeads tests ---

func TestDevloopStatusToBeads(t *testing.T) {
	tests := []struct {
		name             string
		devloopStatus    string
		expectedStatus   string
		expectedLabels   []string
	}{
		{
			name:           "pending maps to open",
			devloopStatus:  "pending",
			expectedStatus: "open",
			expectedLabels: []string{},
		},
		{
			name:           "in_progress maps to in_progress",
			devloopStatus:  "in_progress",
			expectedStatus: "in_progress",
			expectedLabels: []string{},
		},
		{
			name:           "completed maps to closed",
			devloopStatus:  "completed",
			expectedStatus: "closed",
			expectedLabels: []string{},
		},
		{
			name:           "failed maps to closed with failed label",
			devloopStatus:  "failed",
			expectedStatus: "closed",
			expectedLabels: []string{"failed"},
		},
		{
			name:           "blocked maps to blocked",
			devloopStatus:  "blocked",
			expectedStatus: "blocked",
			expectedLabels: []string{},
		},
		{
			name:           "archived maps to closed with compacted label",
			devloopStatus:  "archived",
			expectedStatus: "closed",
			expectedLabels: []string{"compacted"},
		},
		{
			name:           "unknown status defaults to open with warning",
			devloopStatus:  "unknown_status",
			expectedStatus: "open",
			expectedLabels: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := devloopStatusToBeads(tt.devloopStatus)

			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %q, got %q", tt.expectedStatus, result.Status)
			}

			if !stringSlicesEqual(result.Labels, tt.expectedLabels) {
				t.Errorf("expected labels %v, got %v", tt.expectedLabels, result.Labels)
			}
		})
	}
}

// --- beadsStatusToDevloop tests ---

func TestBeadsStatusToDevloop(t *testing.T) {
	tests := []struct {
		name            string
		beadsStatus     BeadsStatusInfo
		expectedStatus  string
	}{
		{
			name:           "open maps to pending",
			beadsStatus:    BeadsStatusInfo{Status: "open"},
			expectedStatus: "pending",
		},
		{
			name:           "in_progress maps to in_progress",
			beadsStatus:    BeadsStatusInfo{Status: "in_progress"},
			expectedStatus: "in_progress",
		},
		{
			name:           "closed without labels maps to completed",
			beadsStatus:    BeadsStatusInfo{Status: "closed"},
			expectedStatus: "completed",
		},
		{
			name:           "closed with failed label maps to failed",
			beadsStatus:    BeadsStatusInfo{Status: "closed", Labels: []string{"failed"}},
			expectedStatus: "failed",
		},
		{
			name:           "closed with compacted label maps to archived",
			beadsStatus:    BeadsStatusInfo{Status: "closed", Labels: []string{"compacted"}},
			expectedStatus: "archived",
		},
		{
			name:           "closed with multiple labels including failed maps to failed",
			beadsStatus:    BeadsStatusInfo{Status: "closed", Labels: []string{"compacted", "failed"}},
			expectedStatus: "failed",
		},
		{
			name:           "closed with multiple labels including compacted maps to archived",
			beadsStatus:    BeadsStatusInfo{Status: "closed", Labels: []string{"other", "compacted"}},
			expectedStatus: "archived",
		},
		{
			name:           "blocked maps to blocked",
			beadsStatus:    BeadsStatusInfo{Status: "blocked"},
			expectedStatus: "blocked",
		},
		{
			name:           "unknown Beads status defaults to pending with warning",
			beadsStatus:    BeadsStatusInfo{Status: "unknown"},
			expectedStatus: "pending",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := beadsStatusToDevloop(tt.beadsStatus)

			if result != tt.expectedStatus {
				t.Errorf("expected status %q, got %q", tt.expectedStatus, result)
			}
		})
	}
}

// --- Roundtrip tests ---

func TestStatusMappingRoundtrip(t *testing.T) {
	tests := []struct {
		name          string
		devloopStatus string
		expectedRoundtrip string
	}{
		{"pending roundtrips", "pending", "pending"},
		{"in_progress roundtrips", "in_progress", "in_progress"},
		{"completed roundtrips", "completed", "completed"},
		{"failed roundtrips", "failed", "failed"},
		{"blocked roundtrips", "blocked", "blocked"},
		{"archived roundtrips", "archived", "archived"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// devloop → beads → devloop
			beadsStatus := devloopStatusToBeads(tt.devloopStatus)
			devloopStatus := beadsStatusToDevloop(beadsStatus)

			if devloopStatus != tt.expectedRoundtrip {
				t.Errorf("roundtrip failed: %q → %+v → %q, expected %q",
					tt.devloopStatus, beadsStatus, devloopStatus, tt.expectedRoundtrip)
			}
		})
	}
}

// Helper function to compare string slices
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
