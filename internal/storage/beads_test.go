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

// --- GetTask tests ---

func TestGetTask_ParsesBdShowJSON(t *testing.T) {
	store := newTestStore()

	bdJSON := `[{"id":"bd-x7f3","title":"My Task","description":"Do the thing","status":"open","labels":["backend"],"deps":["bd-abc1"],"metadata":{"complexity":"moderate","max_attempts":3}}]`

	orig := execCommandContextFunc
	callCount := 0
	execCommandContextFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		callCount++
		switch callCount {
		case 1:
			// bd kv get devloop:DEV-5
			return exec.CommandContext(ctx, "sh", "-c", "echo bd-x7f3")
		case 2:
			// bd show bd-x7f3 --json
			return exec.CommandContext(ctx, "sh", "-c", "echo '"+bdJSON+"'")
		}
		t.Errorf("unexpected call %d", callCount)
		return exec.CommandContext(ctx, "sh", "-c", "exit 1")
	}
	defer func() { execCommandContextFunc = orig }()

	task, err := store.GetTask(context.Background(), "DEV-5")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if task.ID != "bd-x7f3" {
		t.Errorf("ID: expected %q, got %q", "bd-x7f3", task.ID)
	}
	if task.Title != "My Task" {
		t.Errorf("Title: expected %q, got %q", "My Task", task.Title)
	}
	if task.Description != "Do the thing" {
		t.Errorf("Description: expected %q, got %q", "Do the thing", task.Description)
	}
	if task.Status != "pending" {
		t.Errorf("Status: expected %q, got %q", "pending", task.Status)
	}
	if task.Complexity != "moderate" {
		t.Errorf("Complexity: expected %q, got %q", "moderate", task.Complexity)
	}
	if !stringSlicesEqual(task.Tags, []string{"backend"}) {
		t.Errorf("Tags: expected [backend], got %v", task.Tags)
	}
	if task.Metadata.MaxAttempts != 3 {
		t.Errorf("MaxAttempts: expected 3, got %d", task.Metadata.MaxAttempts)
	}
	if !stringSlicesEqual(task.BlockedBy, []string{"bd-abc1"}) {
		t.Errorf("BlockedBy: expected [bd-abc1], got %v", task.BlockedBy)
	}
}

func TestGetTask_PopulatesBlockedByFromBdShow(t *testing.T) {
	store := newTestStore()

	bdJSON := `[{"id":"bd-x7f3","title":"Task","description":"desc","status":"blocked","labels":[],"deps":["bd-dep1","bd-dep2"]}]`

	orig := execCommandContextFunc
	execCommandContextFunc = mockExecCommand("echo '"+bdJSON+"'", nil)
	defer func() { execCommandContextFunc = orig }()

	// Already a Beads ID, so no KV lookup
	task, err := store.GetTask(context.Background(), "bd-x7f3")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if len(task.BlockedBy) != 2 || task.BlockedBy[0] != "bd-dep1" || task.BlockedBy[1] != "bd-dep2" {
		t.Errorf("BlockedBy: expected [bd-dep1 bd-dep2], got %v", task.BlockedBy)
	}
}

func TestGetTask_BdShowFails(t *testing.T) {
	store := newTestStore()

	orig := execCommandContextFunc
	execCommandContextFunc = mockExecCommand("exit 1", nil)
	defer func() { execCommandContextFunc = orig }()

	_, err := store.GetTask(context.Background(), "bd-x7f3")
	if err == nil {
		t.Fatal("GetTask should fail when bd show fails")
	}
}

func TestGetTask_InvalidJSON(t *testing.T) {
	store := newTestStore()

	orig := execCommandContextFunc
	execCommandContextFunc = mockExecCommand("echo 'not json'", nil)
	defer func() { execCommandContextFunc = orig }()

	_, err := store.GetTask(context.Background(), "bd-x7f3")
	if err == nil {
		t.Fatal("GetTask should fail on invalid JSON")
	}
}

// --- LoadTasks tests ---

func TestLoadTasks_ParsesJSONList(t *testing.T) {
	store := newTestStore()

	listJSON := `[{"id":"bd-aaa1","title":"Task A","description":"body a","status":"open","labels":[],"deps":[]},{"id":"bd-bbb2","title":"Task B","description":"body b","status":"in_progress","labels":[],"deps":["bd-aaa1"]}]`

	orig := execCommandContextFunc
	var capturedArgs [][]string
	execCommandContextFunc = mockExecCommand("echo '"+listJSON+"'", &capturedArgs)
	defer func() { execCommandContextFunc = orig }()

	tasks, err := store.LoadTasks(context.Background())
	if err != nil {
		t.Fatalf("LoadTasks failed: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != "bd-aaa1" {
		t.Errorf("tasks[0].ID: expected %q, got %q", "bd-aaa1", tasks[0].ID)
	}
	if tasks[0].Status != "pending" {
		t.Errorf("tasks[0].Status: expected %q, got %q", "pending", tasks[0].Status)
	}
	if tasks[1].ID != "bd-bbb2" {
		t.Errorf("tasks[1].ID: expected %q, got %q", "bd-bbb2", tasks[1].ID)
	}
	if tasks[1].Status != "in_progress" {
		t.Errorf("tasks[1].Status: expected %q, got %q", "in_progress", tasks[1].Status)
	}
	if !stringSlicesEqual(tasks[1].BlockedBy, []string{"bd-aaa1"}) {
		t.Errorf("tasks[1].BlockedBy: expected [bd-aaa1], got %v", tasks[1].BlockedBy)
	}

	// Verify --status open was passed
	if len(capturedArgs) != 1 {
		t.Fatalf("expected 1 exec call, got %d", len(capturedArgs))
	}
	args := capturedArgs[0]
	if !contains(joinArgs(args), "--status open") {
		t.Errorf("expected --status open in args: %v", args)
	}
}

func TestLoadTasks_EmptyList(t *testing.T) {
	store := newTestStore()

	orig := execCommandContextFunc
	execCommandContextFunc = mockExecCommand("echo '[]'", nil)
	defer func() { execCommandContextFunc = orig }()

	tasks, err := store.LoadTasks(context.Background())
	if err != nil {
		t.Fatalf("LoadTasks failed: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestLoadTasks_ReadsComplexityFromMetadata(t *testing.T) {
	store := newTestStore()

	listJSON := `[{"id":"bd-aaa1","title":"Task A","description":"body","status":"open","labels":[],"deps":[],"metadata":{"complexity":"simple"}}]`

	orig := execCommandContextFunc
	execCommandContextFunc = mockExecCommand("echo '"+listJSON+"'", nil)
	defer func() { execCommandContextFunc = orig }()

	tasks, err := store.LoadTasks(context.Background())
	if err != nil {
		t.Fatalf("LoadTasks failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Complexity != "simple" {
		t.Errorf("Complexity: expected %q, got %q", "simple", tasks[0].Complexity)
	}
}

// --- QueryTasks tests ---

func TestQueryTasks_MapsStatusFilter(t *testing.T) {
	store := newTestStore()

	orig := execCommandContextFunc
	var capturedArgs [][]string
	execCommandContextFunc = mockExecCommand("echo '[]'", &capturedArgs)
	defer func() { execCommandContextFunc = orig }()

	_, err := store.QueryTasks(context.Background(), Filter{Status: "pending"})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if len(capturedArgs) != 1 {
		t.Fatalf("expected 1 exec call, got %d", len(capturedArgs))
	}
	// pending → open
	if !contains(joinArgs(capturedArgs[0]), "--status open") {
		t.Errorf("expected --status open for pending filter, got: %v", capturedArgs[0])
	}
}

func TestQueryTasks_NoStatusFilter(t *testing.T) {
	store := newTestStore()

	orig := execCommandContextFunc
	var capturedArgs [][]string
	execCommandContextFunc = mockExecCommand("echo '[]'", &capturedArgs)
	defer func() { execCommandContextFunc = orig }()

	_, err := store.QueryTasks(context.Background(), Filter{})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if len(capturedArgs) != 1 {
		t.Fatalf("expected 1 exec call, got %d", len(capturedArgs))
	}
	// No --status flag when filter is empty
	if contains(joinArgs(capturedArgs[0]), "--status") {
		t.Errorf("expected no --status flag for empty filter, got: %v", capturedArgs[0])
	}
}

func TestQueryTasks_CompletedMapsToClosedStatus(t *testing.T) {
	store := newTestStore()

	orig := execCommandContextFunc
	var capturedArgs [][]string
	execCommandContextFunc = mockExecCommand("echo '[]'", &capturedArgs)
	defer func() { execCommandContextFunc = orig }()

	_, err := store.QueryTasks(context.Background(), Filter{Status: "completed"})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	// completed → closed
	if !contains(joinArgs(capturedArgs[0]), "--status closed") {
		t.Errorf("expected --status closed for completed filter, got: %v", capturedArgs[0])
	}
}

// --- QueryReadyTasks tests ---

func TestQueryReadyTasks_CallsBdReady(t *testing.T) {
	store := newTestStore()

	orig := execCommandContextFunc
	var capturedArgs [][]string
	execCommandContextFunc = mockExecCommand("echo '[]'", &capturedArgs)
	defer func() { execCommandContextFunc = orig }()

	_, err := store.QueryReadyTasks(context.Background())
	if err != nil {
		t.Fatalf("QueryReadyTasks failed: %v", err)
	}

	// Verify bd ready --json was called
	if len(capturedArgs) != 1 {
		t.Fatalf("expected 1 exec call, got %d", len(capturedArgs))
	}
	args := capturedArgs[0]
	if args[0] != "/usr/local/bin/bd" {
		t.Errorf("expected bd binary, got %q", args[0])
	}
	if len(args) < 3 || args[1] != "ready" || args[2] != "--json" {
		t.Errorf("expected 'ready --json' args, got: %v", args[1:])
	}
}

func TestQueryReadyTasks_ReadsComplexityFromMetadata(t *testing.T) {
	store := newTestStore()

	readyJSON := `[{"id":"bd-aaa1","title":"Ready Task","description":"description","status":"open","labels":[],"deps":[],"metadata":{"complexity":"simple"}}]`

	orig := execCommandContextFunc
	execCommandContextFunc = mockExecCommand("echo '"+readyJSON+"'", nil)
	defer func() { execCommandContextFunc = orig }()

	tasks, err := store.QueryReadyTasks(context.Background())
	if err != nil {
		t.Fatalf("QueryReadyTasks failed: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != "bd-aaa1" {
		t.Errorf("ID: expected %q, got %q", "bd-aaa1", tasks[0].ID)
	}
	if tasks[0].Complexity != "simple" {
		t.Errorf("Complexity: expected %q, got %q", "simple", tasks[0].Complexity)
	}
}

func TestQueryReadyTasks_EmptyWhenNoTasksReady(t *testing.T) {
	store := newTestStore()

	orig := execCommandContextFunc
	execCommandContextFunc = mockExecCommand("echo '[]'", nil)
	defer func() { execCommandContextFunc = orig }()

	tasks, err := store.QueryReadyTasks(context.Background())
	if err != nil {
		t.Fatalf("QueryReadyTasks failed: %v", err)
	}

	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
	if tasks == nil {
		t.Error("expected empty slice, not nil")
	}
}

func TestQueryReadyTasks_BdReadyFails(t *testing.T) {
	store := newTestStore()

	orig := execCommandContextFunc
	execCommandContextFunc = mockExecCommand("exit 1", nil)
	defer func() { execCommandContextFunc = orig }()

	_, err := store.QueryReadyTasks(context.Background())
	if err == nil {
		t.Fatal("QueryReadyTasks should fail when bd ready fails")
	}
	if !contains(err.Error(), "bd ready") {
		t.Errorf("expected 'bd ready' in error, got: %v", err)
	}
}

func TestQueryReadyTasks_InvalidJSON(t *testing.T) {
	store := newTestStore()

	orig := execCommandContextFunc
	execCommandContextFunc = mockExecCommand("echo 'not json'", nil)
	defer func() { execCommandContextFunc = orig }()

	_, err := store.QueryReadyTasks(context.Background())
	if err == nil {
		t.Fatal("QueryReadyTasks should fail on invalid JSON")
	}
}

// --- UpdateTask tests ---

func TestUpdateTask_InProgress_ClaimsCalled(t *testing.T) {
	store := newTestStore()

	var capturedArgs [][]string
	orig := execCommandContextFunc
	execCommandContextFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		capturedArgs = append(capturedArgs, append([]string{name}, args...))
		return exec.CommandContext(ctx, "sh", "-c", "true")
	}
	defer func() { execCommandContextFunc = orig }()

	task := &Task{
		ID:     "bd-x7f3",
		Status: "in_progress",
		Execution: TaskExecution{
			TotalDuration: 30,
		},
	}

	if err := store.UpdateTask(context.Background(), task); err != nil {
		t.Fatalf("UpdateTask in_progress failed: %v", err)
	}

	// Verify bd update --claim --json was called
	if len(capturedArgs) < 1 {
		t.Fatal("expected at least one exec call")
	}
	first := joinArgs(capturedArgs[0])
	if !contains(first, "update") {
		t.Errorf("expected 'update' in first call args: %v", capturedArgs[0])
	}
	if !contains(first, "--claim") {
		t.Errorf("expected '--claim' in first call args: %v", capturedArgs[0])
	}
	if !contains(first, "--json") {
		t.Errorf("expected '--json' in first call args: %v", capturedArgs[0])
	}
	if !contains(first, "bd-x7f3") {
		t.Errorf("expected beads ID 'bd-x7f3' in first call args: %v", capturedArgs[0])
	}

	// Only one bd call for in_progress (claim)
	if len(capturedArgs) != 1 {
		t.Errorf("expected exactly 1 exec call for in_progress, got %d", len(capturedArgs))
	}
}

func TestUpdateTask_Completed_CloseCalled(t *testing.T) {
	store := newTestStore()

	var capturedArgs [][]string
	orig := execCommandContextFunc
	execCommandContextFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		capturedArgs = append(capturedArgs, append([]string{name}, args...))
		return exec.CommandContext(ctx, "sh", "-c", "true")
	}
	defer func() { execCommandContextFunc = orig }()

	task := &Task{
		ID:     "bd-x7f3",
		Status: "completed",
		Results: &TaskResults{
			VerificationOutput: "all tests passed",
			CommitHash:         "abc123",
		},
	}

	if err := store.UpdateTask(context.Background(), task); err != nil {
		t.Fatalf("UpdateTask completed failed: %v", err)
	}

	// Verify bd close --reason 'Verification passed' was called
	if len(capturedArgs) < 1 {
		t.Fatal("expected at least one exec call")
	}
	first := joinArgs(capturedArgs[0])
	if !contains(first, "close") {
		t.Errorf("expected 'close' in first call args: %v", capturedArgs[0])
	}
	if !contains(first, "--reason") {
		t.Errorf("expected '--reason' in first call args: %v", capturedArgs[0])
	}
	if !contains(first, "Verification passed") {
		t.Errorf("expected 'Verification passed' in first call args: %v", capturedArgs[0])
	}

	// Only one bd call for completed
	if len(capturedArgs) != 1 {
		t.Errorf("expected exactly 1 exec call for completed, got %d", len(capturedArgs))
	}
}

func TestUpdateTask_Failed_CloseAndLabelCalled(t *testing.T) {
	store := newTestStore()

	var capturedArgs [][]string
	orig := execCommandContextFunc
	execCommandContextFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		capturedArgs = append(capturedArgs, append([]string{name}, args...))
		return exec.CommandContext(ctx, "sh", "-c", "true")
	}
	defer func() { execCommandContextFunc = orig }()

	task := &Task{
		ID:     "bd-x7f3",
		Status: "failed",
	}

	if err := store.UpdateTask(context.Background(), task); err != nil {
		t.Fatalf("UpdateTask failed status failed: %v", err)
	}

	// Expect exactly 2 bd calls: close + label add
	if len(capturedArgs) != 2 {
		t.Fatalf("expected 2 exec calls for failed status, got %d: %v", len(capturedArgs), capturedArgs)
	}

	// First call: bd close <id> --reason ...
	first := joinArgs(capturedArgs[0])
	if !contains(first, "close") {
		t.Errorf("expected 'close' in first call: %v", capturedArgs[0])
	}
	if !contains(first, "--reason") {
		t.Errorf("expected '--reason' in first call: %v", capturedArgs[0])
	}

	// Second call: bd label add <id> failed
	second := joinArgs(capturedArgs[1])
	if !contains(second, "label") {
		t.Errorf("expected 'label' in second call: %v", capturedArgs[1])
	}
	if !contains(second, "add") {
		t.Errorf("expected 'add' in second call: %v", capturedArgs[1])
	}
	if !contains(second, "failed") {
		t.Errorf("expected 'failed' in second call: %v", capturedArgs[1])
	}
}

func TestUpdateTask_UnsupportedStatus_ReturnsError(t *testing.T) {
	store := newTestStore()

	orig := execCommandContextFunc
	execCommandContextFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "true")
	}
	defer func() { execCommandContextFunc = orig }()

	task := &Task{
		ID:     "bd-x7f3",
		Status: "pending",
	}

	err := store.UpdateTask(context.Background(), task)
	if err == nil {
		t.Fatal("UpdateTask should fail for unsupported status")
	}
	if !contains(err.Error(), "unsupported status") {
		t.Errorf("expected 'unsupported status' in error, got: %v", err)
	}
}

// joinArgs concatenates args with spaces for simple substring checking
func joinArgs(args []string) string {
	result := ""
	for i, a := range args {
		if i > 0 {
			result += " "
		}
		result += a
	}
	return result
}

// --- SaveTask tests ---

// mockExecCommandSequence returns a mock that cycles through shell commands in order.
// Each call consumes the next command; the last command is repeated for any extra calls.
func mockExecCommandSequence(shellCmds []string, capturedArgs *[][]string) func(ctx context.Context, name string, args ...string) *exec.Cmd {
	idx := 0
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if capturedArgs != nil {
			*capturedArgs = append(*capturedArgs, append([]string{name}, args...))
		}
		cmd := shellCmds[idx]
		if idx < len(shellCmds)-1 {
			idx++
		}
		return exec.CommandContext(ctx, "sh", "-c", cmd)
	}
}

func TestSaveTask_BasicCreationFlow(t *testing.T) {
	store := newTestStore()

	var capturedArgs [][]string
	orig := execCommandContextFunc
	// Sequence: bd create → 2x kv set → bd update (metadata/acceptance/labels)
	execCommandContextFunc = mockExecCommandSequence([]string{
		`echo '{"id":"bd-a1b2"}'`, // bd create --json
		"true",                    // bd kv set devloop:DEV-1 bd-a1b2
		"true",                    // bd kv set beads:bd-a1b2 DEV-1
		"true",                    // bd update --metadata --acceptance --labels
	}, &capturedArgs)
	defer func() { execCommandContextFunc = orig }()

	task := &Task{
		ID:                 "DEV-1",
		Title:              "My task",
		Description:        "Do something useful",
		Complexity:         "simple",
		AcceptanceCriteria: []string{"it works"},
		Tags:               []string{"backend"},
	}
	task.Metadata.MaxAttempts = 3

	if err := store.SaveTask(context.Background(), task); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	// Verify bd create was called with --body-file and --json
	if len(capturedArgs) < 1 {
		t.Fatal("expected at least one exec call")
	}
	createCall := joinArgs(capturedArgs[0])
	if !contains(createCall, "create") {
		t.Errorf("expected 'create' in first call: %v", capturedArgs[0])
	}
	if !contains(createCall, "--body-file") {
		t.Errorf("expected '--body-file' in create call: %v", capturedArgs[0])
	}
	if !contains(createCall, "--json") {
		t.Errorf("expected '--json' in create call: %v", capturedArgs[0])
	}
	if !contains(createCall, "My task") {
		t.Errorf("expected title 'My task' in create call: %v", capturedArgs[0])
	}

	// Verify KV mapping calls
	if len(capturedArgs) < 4 {
		t.Fatalf("expected at least 4 exec calls (create + 2 kv set + update), got %d", len(capturedArgs))
	}
	kvCall1 := joinArgs(capturedArgs[1])
	if !contains(kvCall1, "kv") || !contains(kvCall1, "set") {
		t.Errorf("expected 'kv set' in second call: %v", capturedArgs[1])
	}
	if !contains(kvCall1, "devloop:DEV-1") {
		t.Errorf("expected 'devloop:DEV-1' in kv set call: %v", capturedArgs[1])
	}
	if !contains(kvCall1, "bd-a1b2") {
		t.Errorf("expected 'bd-a1b2' in kv set call: %v", capturedArgs[1])
	}

	kvCall2 := joinArgs(capturedArgs[2])
	if !contains(kvCall2, "kv") || !contains(kvCall2, "set") {
		t.Errorf("expected 'kv set' in third call: %v", capturedArgs[2])
	}
	if !contains(kvCall2, "beads:bd-a1b2") {
		t.Errorf("expected 'beads:bd-a1b2' in kv set call: %v", capturedArgs[2])
	}
	if !contains(kvCall2, "DEV-1") {
		t.Errorf("expected 'DEV-1' in kv set call: %v", capturedArgs[2])
	}

	// Verify bd update was called with metadata, acceptance, and labels
	updateCall := joinArgs(capturedArgs[3])
	if !contains(updateCall, "update") {
		t.Errorf("expected 'update' in fourth call: %v", capturedArgs[3])
	}
	if !contains(updateCall, "--metadata") {
		t.Errorf("expected '--metadata' in update call: %v", capturedArgs[3])
	}
	if !contains(updateCall, "simple") {
		t.Errorf("expected complexity 'simple' in metadata: %v", capturedArgs[3])
	}
	if !contains(updateCall, "--acceptance") {
		t.Errorf("expected '--acceptance' in update call: %v", capturedArgs[3])
	}
	if !contains(updateCall, "it works") {
		t.Errorf("expected acceptance criteria in update call: %v", capturedArgs[3])
	}
	if !contains(updateCall, "--labels") {
		t.Errorf("expected '--labels' in update call: %v", capturedArgs[3])
	}
	if !contains(updateCall, "backend") {
		t.Errorf("expected tag 'backend' in update call: %v", capturedArgs[3])
	}
}

func TestSaveTask_WithDependencies(t *testing.T) {
	store := newTestStore()

	var capturedArgs [][]string
	orig := execCommandContextFunc
	// Sequence: bd create → 2x kv set → bd update (metadata) → bd kv get DEV-2 → bd dep add
	execCommandContextFunc = mockExecCommandSequence([]string{
		`echo '{"id":"bd-c3d4"}'`, // bd create
		"true",                    // bd kv set devloop:DEV-3
		"true",                    // bd kv set beads:bd-c3d4
		"true",                    // bd update --metadata
		"echo bd-e5f6",            // bd kv get devloop:DEV-2
		"true",                    // bd dep add
	}, &capturedArgs)
	defer func() { execCommandContextFunc = orig }()

	task := &Task{
		ID:          "DEV-3",
		Title:       "Dependent task",
		Description: "Depends on DEV-2",
		BlockedBy:   []string{"DEV-2"},
	}

	if err := store.SaveTask(context.Background(), task); err != nil {
		t.Fatalf("SaveTask with dependency failed: %v", err)
	}

	// Should have 6 calls: create, 2 kv set, update (metadata), kv get (resolve dep), dep add
	if len(capturedArgs) != 6 {
		t.Fatalf("expected 6 exec calls, got %d: %v", len(capturedArgs), capturedArgs)
	}

	// Verify dep add call
	depCall := joinArgs(capturedArgs[5])
	if !contains(depCall, "dep") || !contains(depCall, "add") {
		t.Errorf("expected 'dep add' in last call: %v", capturedArgs[4])
	}
	if !contains(depCall, "bd-c3d4") {
		t.Errorf("expected task ID 'bd-c3d4' in dep add call: %v", capturedArgs[5])
	}
	if !contains(depCall, "bd-e5f6") {
		t.Errorf("expected dep ID 'bd-e5f6' in dep add call: %v", capturedArgs[5])
	}
}

func TestSaveTask_TempFileCleanedUp(t *testing.T) {
	store := newTestStore()

	var tempFilePath string
	orig := execCommandContextFunc
	execCommandContextFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// Capture the --body-file path from the create call
		for i, arg := range args {
			if arg == "--body-file" && i+1 < len(args) {
				tempFilePath = args[i+1]
			}
		}
		if len(args) > 0 && args[0] == "create" {
			return exec.CommandContext(ctx, "sh", "-c", `echo '{"id":"bd-g7h8"}'`)
		}
		return exec.CommandContext(ctx, "sh", "-c", "true")
	}
	defer func() { execCommandContextFunc = orig }()

	task := &Task{
		ID:          "DEV-4",
		Title:       "Cleanup test",
		Description: "Check temp file removal",
	}

	if err := store.SaveTask(context.Background(), task); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	if tempFilePath == "" {
		t.Fatal("expected --body-file path to be captured")
	}

	// Temp file should be removed after SaveTask
	fileExistsCmd := exec.Command("sh", "-c", "test -f "+tempFilePath)
	if fileExistsCmd.Run() == nil {
		t.Errorf("temp file %q should have been removed after SaveTask", tempFilePath)
	}
}

func TestSaveTask_BdCreateFails(t *testing.T) {
	store := newTestStore()

	orig := execCommandContextFunc
	execCommandContextFunc = mockExecCommand("exit 1", nil)
	defer func() { execCommandContextFunc = orig }()

	task := &Task{
		ID:          "DEV-5",
		Title:       "Failing task",
		Description: "This should fail",
	}

	err := store.SaveTask(context.Background(), task)
	if err == nil {
		t.Fatal("SaveTask should fail when bd create fails")
	}
	if !contains(err.Error(), "bd create failed") {
		t.Errorf("expected 'bd create failed' in error, got: %v", err)
	}
}

func TestSaveTask_BdCreateReturnsInvalidJSON(t *testing.T) {
	store := newTestStore()

	orig := execCommandContextFunc
	execCommandContextFunc = mockExecCommand("echo 'not json'", nil)
	defer func() { execCommandContextFunc = orig }()

	task := &Task{
		ID:          "DEV-6",
		Title:       "JSON test",
		Description: "Test invalid JSON response",
	}

	err := store.SaveTask(context.Background(), task)
	if err == nil {
		t.Fatal("SaveTask should fail when bd create returns invalid JSON")
	}
	if !contains(err.Error(), "failed to parse bd create JSON output") {
		t.Errorf("expected JSON parse error, got: %v", err)
	}
}

func TestSaveTask_BdCreateReturnsEmptyID(t *testing.T) {
	store := newTestStore()

	orig := execCommandContextFunc
	execCommandContextFunc = mockExecCommand(`echo '{"id":""}'`, nil)
	defer func() { execCommandContextFunc = orig }()

	task := &Task{
		ID:          "DEV-7",
		Title:       "Empty ID test",
		Description: "Test empty ID response",
	}

	err := store.SaveTask(context.Background(), task)
	if err == nil {
		t.Fatal("SaveTask should fail when bd create returns empty ID")
	}
	if !contains(err.Error(), "bd create returned empty ID") {
		t.Errorf("expected 'bd create returned empty ID' in error, got: %v", err)
	}
}

func TestSaveTask_DepAddFails(t *testing.T) {
	store := newTestStore()

	orig := execCommandContextFunc
	callCount := 0
	execCommandContextFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		callCount++
		switch callCount {
		case 1: // bd create
			return exec.CommandContext(ctx, "sh", "-c", `echo '{"id":"bd-i9j0"}'`)
		case 2, 3: // bd kv set
			return exec.CommandContext(ctx, "sh", "-c", "true")
		case 4: // bd update --metadata
			return exec.CommandContext(ctx, "sh", "-c", "true")
		case 5: // bd kv get (resolve dep)
			return exec.CommandContext(ctx, "sh", "-c", "echo bd-k1l2")
		default: // bd dep add - fail
			return exec.CommandContext(ctx, "sh", "-c", "exit 1")
		}
	}
	defer func() { execCommandContextFunc = orig }()

	task := &Task{
		ID:          "DEV-8",
		Title:       "Dep fail test",
		Description: "Test dep add failure",
		BlockedBy:   []string{"DEV-9"},
	}

	err := store.SaveTask(context.Background(), task)
	if err == nil {
		t.Fatal("SaveTask should fail when bd dep add fails")
	}
	if !contains(err.Error(), "bd dep add") {
		t.Errorf("expected 'bd dep add' in error, got: %v", err)
	}
}

// --- Sync tests ---

func TestSync_Success(t *testing.T) {
store := newTestStore()

var capturedArgs [][]string
orig := execCommandContextFunc
execCommandContextFunc = mockExecCommand("echo synced", &capturedArgs)
defer func() { execCommandContextFunc = orig }()

err := store.Sync()
if err != nil {
t.Fatalf("Sync() should succeed: %v", err)
}

if len(capturedArgs) != 1 {
t.Fatalf("expected 1 exec call, got %d", len(capturedArgs))
}
args := capturedArgs[0]
if args[0] != "/usr/local/bin/bd" {
t.Errorf("expected bd binary, got %q", args[0])
}
if len(args) != 2 || args[1] != "sync" {
t.Errorf("expected 'bd sync', got %v", args[1:])
}
}

func TestSync_Failure(t *testing.T) {
store := newTestStore()

orig := execCommandContextFunc
execCommandContextFunc = mockExecCommand("exit 1", nil)
defer func() { execCommandContextFunc = orig }()

err := store.Sync()
if err == nil {
t.Fatal("Sync() should fail when bd sync fails")
}
if !contains(err.Error(), "bd sync failed") {
t.Errorf("expected 'bd sync failed' in error, got: %v", err)
}
}
