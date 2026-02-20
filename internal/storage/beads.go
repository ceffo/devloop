package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ceffo/devloop/internal/config"
)

// lookPathFunc is a function variable for looking up executables in PATH
// It's exposed for testing purposes
var lookPathFunc = exec.LookPath

// execCommandContextFunc is a function variable for creating exec.Cmd with context
// It's exposed for testing purposes
var execCommandContextFunc = exec.CommandContext

// BeadsStore manages interactions with the bd binary
type BeadsStore struct {
	cfg    *config.Config
	bdPath string
}

// NewBeadsStore creates a new BeadsStore instance and verifies the bd binary is available
func NewBeadsStore(cfg *config.Config) (*BeadsStore, error) {
	// Check if bd binary is available on $PATH
	bdPath, err := lookPathFunc("bd")
	if err != nil {
		return nil, fmt.Errorf("bd binary not found: %w\n\ninstall options:\n  - go install: go install github.com/midbel/bead@latest\n  - npm install: npm install -g beads\n  - brew install: brew install beads", err)
	}

	return &BeadsStore{
		cfg:    cfg,
		bdPath: bdPath,
	}, nil
}

// isBeadsHashID returns true if id looks like a native Beads hash ID (e.g., "bd-x7f3")
func isBeadsHashID(id string) bool {
	return strings.HasPrefix(id, "bd-")
}

// resolveBeadsID returns the Beads hash ID for the given ID.
// If id is already a Beads hash ID (e.g. "bd-x7f3"), it is returned as-is.
// Otherwise, a KV lookup is performed via `bd kv get "devloop:<id>"`.
func (s *BeadsStore) resolveBeadsID(ctx context.Context, devloopID string) (string, error) {
	if isBeadsHashID(devloopID) {
		return devloopID, nil
	}

	kvCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	key := "devloop:" + devloopID
	cmd := execCommandContextFunc(kvCtx, s.bdPath, "kv", "get", key)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("bd kv get %q failed: %w", key, err)
	}

	beadsID := strings.TrimSpace(string(out))
	if beadsID == "" {
		return "", fmt.Errorf("no Beads ID found for devloop ID %q", devloopID)
	}

	return beadsID, nil
}

// writeIDMapping writes bidirectional KV entries for a devloop ID and a Beads hash ID.
// It writes:
//
//	devloop:<devloopID> → <beadsID>
//	beads:<beadsID>     → <devloopID>
func (s *BeadsStore) writeIDMapping(ctx context.Context, devloopID, beadsID string) error {
	entries := [][2]string{
		{"devloop:" + devloopID, beadsID},
		{"beads:" + beadsID, devloopID},
	}

	for _, entry := range entries {
		kvCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		cmd := execCommandContextFunc(kvCtx, s.bdPath, "kv", "set", entry[0], entry[1])
		out, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			return fmt.Errorf("bd kv set %q failed: %w (output: %s)", entry[0], err, strings.TrimSpace(string(out)))
		}
	}

	return nil
}

// BeadsStatusInfo represents a Beads task status with optional labels
type BeadsStatusInfo struct {
	Status string   // "open", "in_progress", "closed", "blocked"
	Labels []string // optional labels like "failed", "compacted"
}

// devloopStatusToBeads converts a devloop task status to Beads status representation.
// Maps:
//   - pending → open
//   - in_progress → in_progress
//   - completed → closed
//   - failed → closed + "failed" label
//   - blocked → blocked
//   - archived → closed + "compacted" label
// Unknown statuses return "open" with a warning.
func devloopStatusToBeads(devloopStatus string) BeadsStatusInfo {
	switch devloopStatus {
	case "pending":
		return BeadsStatusInfo{Status: "open"}
	case "in_progress":
		return BeadsStatusInfo{Status: "in_progress"}
	case "completed":
		return BeadsStatusInfo{Status: "closed"}
	case "failed":
		return BeadsStatusInfo{Status: "closed", Labels: []string{"failed"}}
	case "blocked":
		return BeadsStatusInfo{Status: "blocked"}
	case "archived":
		return BeadsStatusInfo{Status: "closed", Labels: []string{"compacted"}}
	default:
		fmt.Printf("WARNING: unknown devloop status %q, using default 'open'\n", devloopStatus)
		return BeadsStatusInfo{Status: "open"}
	}
}

// beadsStatusToDevloop converts a Beads task status to a devloop task status.
// Maps:
//   - open → pending
//   - in_progress → in_progress
//   - closed (no labels) → completed
//   - closed + "failed" label → failed
//   - closed + "compacted" label → archived
//   - blocked → blocked
// Unknown Beads statuses return "pending" with a warning.
// When multiple labels are present, "failed" takes priority over "compacted".
func beadsStatusToDevloop(beadsStatus BeadsStatusInfo) string {
	switch beadsStatus.Status {
	case "open":
		return "pending"
	case "in_progress":
		return "in_progress"
	case "blocked":
		return "blocked"
	case "closed":
		// Check labels to determine exact devloop status
		// Priority: "failed" > "compacted" > no label
		hasFailed := false
		hasCompacted := false
		for _, label := range beadsStatus.Labels {
			if label == "failed" {
				hasFailed = true
			}
			if label == "compacted" {
				hasCompacted = true
			}
		}
		if hasFailed {
			return "failed"
		}
		if hasCompacted {
			return "archived"
		}
		// closed without special labels means completed
		return "completed"
	default:
		fmt.Printf("WARNING: unknown Beads status %q, using default 'pending'\n", beadsStatus.Status)
		return "pending"
	}
}

// bdCreateOutput represents the JSON output of `bd create --json`.
type bdCreateOutput struct {
	ID string `json:"id"`
}

// bdShowTask represents the JSON output of `bd show --json` and each element of `bd list --json`.
type bdShowTask struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Status string   `json:"status"`
	Labels []string `json:"labels"`
	Deps   []string `json:"deps"`
}

// bdShowTaskToBeadsData converts a bdShowTask to beadsTaskData with status mapped to devloop.
func bdShowTaskToBeadsData(t *bdShowTask) beadsTaskData {
	status := beadsStatusToDevloop(BeadsStatusInfo{Status: t.Status, Labels: t.Labels})
	return beadsTaskData{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Body,
		Status:      status,
	}
}

// parseBdListOutput parses the JSON array output of `bd list --json` and merges each task
// with its sidecar file to return a slice of *Task.
func (s *BeadsStore) parseBdListOutput(data []byte) ([]*Task, error) {
	var bdTasks []bdShowTask
	if err := json.Unmarshal(data, &bdTasks); err != nil {
		return nil, fmt.Errorf("failed to parse bd list JSON: %w", err)
	}

	tasks := make([]*Task, 0, len(bdTasks))
	for i := range bdTasks {
		bdTask := &bdTasks[i]
		sidecar, err := s.readSidecar(bdTask.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to read sidecar for %q: %w", bdTask.ID, err)
		}
		task := mergeSidecar(bdShowTaskToBeadsData(bdTask), sidecar)
		task.BlockedBy = bdTask.Deps
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetTask resolves the devloop ID to a Beads ID via KV lookup, fetches the task via
// `bd show --json`, reads the sidecar, and returns the merged *Task.
// BlockedBy is populated from the Beads deps field, not the sidecar.
func (s *BeadsStore) GetTask(ctx context.Context, id string) (*Task, error) {
	beadsID, err := s.resolveBeadsID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("GetTask: failed to resolve ID %q: %w", id, err)
	}

	showCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := execCommandContextFunc(showCtx, s.bdPath, "show", beadsID, "--json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd show %q failed: %w", beadsID, err)
	}

	var bdTask bdShowTask
	if err := json.Unmarshal(out, &bdTask); err != nil {
		return nil, fmt.Errorf("failed to parse bd show JSON: %w (output: %s)", err, strings.TrimSpace(string(out)))
	}

	sidecar, err := s.readSidecar(beadsID)
	if err != nil {
		return nil, fmt.Errorf("failed to read sidecar for %q: %w", beadsID, err)
	}

	task := mergeSidecar(bdShowTaskToBeadsData(&bdTask), sidecar)
	task.BlockedBy = bdTask.Deps
	return task, nil
}

// LoadTasks returns all open tasks via `bd list --json --status open`, merging each with its sidecar.
func (s *BeadsStore) LoadTasks(ctx context.Context) ([]*Task, error) {
	listCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := execCommandContextFunc(listCtx, s.bdPath, "list", "--json", "--status", "open")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd list failed: %w", err)
	}

	return s.parseBdListOutput(out)
}

// QueryTasks applies the devloop→Beads status mapping (when filter.Status is set) and calls
// `bd list --json [--status <mapped>]`, merging each result with its sidecar.
func (s *BeadsStore) QueryTasks(ctx context.Context, filter Filter) ([]*Task, error) {
	args := []string{"list", "--json"}
	if filter.Status != "" {
		beadsStatus := devloopStatusToBeads(filter.Status)
		args = append(args, "--status", beadsStatus.Status)
	}

	listCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := execCommandContextFunc(listCtx, s.bdPath, args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd list failed: %w", err)
	}

	return s.parseBdListOutput(out)
}

// QueryReadyTasks calls `bd ready --json`, parses the output, and merges each result
// with its sidecar data. Returns an empty slice (not error) when no tasks are ready.
func (s *BeadsStore) QueryReadyTasks(ctx context.Context) ([]*Task, error) {
	readyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := execCommandContextFunc(readyCtx, s.bdPath, "ready", "--json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd ready failed: %w", err)
	}

	return s.parseBdListOutput(out)
}

// UpdateTask dispatches the status transition to the appropriate bd command(s)
// and updates the sidecar file with the latest execution and results data.
//   - in_progress: bd update <id> --claim --json (atomic claim)
//   - completed:   bd close <id> --reason 'Verification passed'
//   - failed:      bd close <id> --reason 'Verification failed' + bd label add <id> failed
//
// All bd calls use a 10-second timeout.
func (s *BeadsStore) UpdateTask(ctx context.Context, task *Task) error {
	beadsID, err := s.resolveBeadsID(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("UpdateTask: failed to resolve ID %q: %w", task.ID, err)
	}

	switch task.Status {
	case "in_progress":
		claimCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		cmd := execCommandContextFunc(claimCtx, s.bdPath, "update", beadsID, "--claim", "--json")
		out, cmdErr := cmd.CombinedOutput()
		cancel()
		if cmdErr != nil {
			return fmt.Errorf("bd update --claim %q failed: %w (output: %s)", beadsID, cmdErr, strings.TrimSpace(string(out)))
		}

	case "completed":
		closeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		cmd := execCommandContextFunc(closeCtx, s.bdPath, "close", beadsID, "--reason", "Verification passed")
		out, cmdErr := cmd.CombinedOutput()
		cancel()
		if cmdErr != nil {
			return fmt.Errorf("bd close %q failed: %w (output: %s)", beadsID, cmdErr, strings.TrimSpace(string(out)))
		}

	case "failed":
		closeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		cmd := execCommandContextFunc(closeCtx, s.bdPath, "close", beadsID, "--reason", "Verification failed")
		out, cmdErr := cmd.CombinedOutput()
		cancel()
		if cmdErr != nil {
			return fmt.Errorf("bd close %q failed: %w (output: %s)", beadsID, cmdErr, strings.TrimSpace(string(out)))
		}

		labelCtx, cancelLabel := context.WithTimeout(ctx, 10*time.Second)
		cmd = execCommandContextFunc(labelCtx, s.bdPath, "label", "add", beadsID, "failed")
		out, cmdErr = cmd.CombinedOutput()
		cancelLabel()
		if cmdErr != nil {
			return fmt.Errorf("bd label add failed %q failed: %w (output: %s)", beadsID, cmdErr, strings.TrimSpace(string(out)))
		}

	default:
		return fmt.Errorf("UpdateTask: unsupported status %q for task %q", task.Status, task.ID)
	}

	// Update sidecar with latest execution and results data
	sidecar, err := s.readSidecar(beadsID)
	if err != nil {
		return fmt.Errorf("UpdateTask: failed to read sidecar for %q: %w", beadsID, err)
	}
	if sidecar == nil {
		sidecar = &TaskSidecar{}
	}
	sidecar.Execution = task.Execution
	sidecar.Results = task.Results
	sidecar.DecomposedInto = task.DecomposedInto
	if err := s.writeSidecar(beadsID, sidecar); err != nil {
		return fmt.Errorf("UpdateTask: failed to write sidecar for %q: %w", beadsID, err)
	}

	return nil
}

// Compact calls `bd admin compact --auto --all --tier 1` for memory decay.
// It is not part of the TaskStore interface; callers use a type assertion.
func (s *BeadsStore) Compact() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := execCommandContextFunc(ctx, s.bdPath, "admin", "compact", "--auto", "--all", "--tier", "1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd admin compact failed: %w (output: %s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Sync calls `bd sync` to synchronize the Beads store with the remote.
// It is not part of the TaskStore interface; callers use a type assertion.
// Sync failures should be treated as warnings, not fatal errors.
func (s *BeadsStore) Sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := execCommandContextFunc(ctx, s.bdPath, "sync")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd sync failed: %w (output: %s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// SetMigrationStatus updates a Beads task's status after creation during migration.
// It handles: in_progress (claim), completed (close), failed (close + label),
// archived (close + compacted label), and blocked (no-op, deps handle blocking).
// For pending status, this is a no-op since tasks are created as open by default.
func (s *BeadsStore) SetMigrationStatus(ctx context.Context, beadsID string, devloopStatus string) error {
	switch devloopStatus {
	case "pending", "blocked":
		// pending: created as "open" by default, no action needed
		// blocked: deps already set via SaveTask; beads handles blocking via deps
		return nil

	case "in_progress":
		claimCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		cmd := execCommandContextFunc(claimCtx, s.bdPath, "update", beadsID, "--claim", "--json")
		out, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			return fmt.Errorf("bd update --claim %q failed: %w (output: %s)", beadsID, err, strings.TrimSpace(string(out)))
		}
		return nil

	case "completed":
		closeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		cmd := execCommandContextFunc(closeCtx, s.bdPath, "close", beadsID, "--reason", "migrated")
		out, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			return fmt.Errorf("bd close %q failed: %w (output: %s)", beadsID, err, strings.TrimSpace(string(out)))
		}
		return nil

	case "failed":
		closeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		cmd := execCommandContextFunc(closeCtx, s.bdPath, "close", beadsID, "--reason", "migrated")
		out, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			return fmt.Errorf("bd close %q failed: %w (output: %s)", beadsID, err, strings.TrimSpace(string(out)))
		}
		labelCtx, cancelLabel := context.WithTimeout(ctx, 10*time.Second)
		cmd = execCommandContextFunc(labelCtx, s.bdPath, "label", "add", beadsID, "failed")
		out, err = cmd.CombinedOutput()
		cancelLabel()
		if err != nil {
			return fmt.Errorf("bd label add failed %q failed: %w (output: %s)", beadsID, err, strings.TrimSpace(string(out)))
		}
		return nil

	case "archived":
		closeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		cmd := execCommandContextFunc(closeCtx, s.bdPath, "close", beadsID, "--reason", "migrated")
		out, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			return fmt.Errorf("bd close %q failed: %w (output: %s)", beadsID, err, strings.TrimSpace(string(out)))
		}
		labelCtx, cancelLabel := context.WithTimeout(ctx, 10*time.Second)
		cmd = execCommandContextFunc(labelCtx, s.bdPath, "label", "add", beadsID, "compacted")
		out, err = cmd.CombinedOutput()
		cancelLabel()
		if err != nil {
			return fmt.Errorf("bd label add compacted %q failed: %w (output: %s)", beadsID, err, strings.TrimSpace(string(out)))
		}
		return nil

	default:
		return fmt.Errorf("SetMigrationStatus: unsupported status %q for task %q", devloopStatus, beadsID)
	}
}

// LoadAllTasks returns all tasks from Beads regardless of status via `bd list --json`
// (no status filter), merging each with its sidecar.
func (s *BeadsStore) LoadAllTasks(ctx context.Context) ([]*Task, error) {
	listCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := execCommandContextFunc(listCtx, s.bdPath, "list", "--json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd list failed: %w", err)
	}

	return s.parseBdListOutput(out)
}

// SaveTask implements the six-step creation flow:
//  1. Write task.Description to a temp file
//  2. Call `bd create <title> --body-file <tempfile> --json`
//  3. Parse the returned JSON to capture the Beads hash ID
//  4. Write bidirectional KV mapping (devloop:<ID> ↔ beads:<beadsID>)
//  5. Write sidecar file with devloop-specific metadata
//  6. Link dependencies via `bd dep add` for each BlockedBy entry
//  7. Remove the temp file
func (s *BeadsStore) SaveTask(ctx context.Context, task *Task) error {
	// Step 1: write description to temp file
	tmpFile, err := os.CreateTemp("", "devloop-task-*.md")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	// Ensure cleanup regardless of outcome
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(task.Description); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write description to temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Step 2: call bd create --body-file <path> --json
	createCtx, cancelCreate := context.WithTimeout(ctx, 10*time.Second)
	defer cancelCreate()

	cmd := execCommandContextFunc(createCtx, s.bdPath, "create", task.Title, "--body-file", tmpPath, "--json")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("bd create failed: %w", err)
	}

	// Step 3: parse hash ID from JSON output
	var result bdCreateOutput
	if err := json.Unmarshal(out, &result); err != nil {
		return fmt.Errorf("failed to parse bd create JSON output: %w (output: %s)", err, strings.TrimSpace(string(out)))
	}
	beadsID := result.ID
	if beadsID == "" {
		return fmt.Errorf("bd create returned empty ID (output: %s)", strings.TrimSpace(string(out)))
	}

	// Step 4: write bidirectional KV mapping
	if err := s.writeIDMapping(ctx, task.ID, beadsID); err != nil {
		return fmt.Errorf("failed to write ID mapping: %w", err)
	}

	// Step 5: write sidecar file
	sidecar := &TaskSidecar{
		DevloopID:          task.ID,
		Complexity:         task.Complexity,
		AcceptanceCriteria: task.AcceptanceCriteria,
		Tags:               task.Tags,
		MaxAttempts:        task.Metadata.MaxAttempts,
		Execution:          task.Execution,
		Results:            task.Results,
	}
	if err := s.writeSidecar(beadsID, sidecar); err != nil {
		return fmt.Errorf("failed to write sidecar: %w", err)
	}

	// Step 6: link dependencies
	for _, depID := range task.BlockedBy {
		resolvedDep, err := s.resolveBeadsID(ctx, depID)
		if err != nil {
			return fmt.Errorf("failed to resolve dependency %q: %w", depID, err)
		}

		depCtx, cancelDep := context.WithTimeout(ctx, 10*time.Second)
		depCmd := execCommandContextFunc(depCtx, s.bdPath, "dep", "add", beadsID, resolvedDep)
		depOut, depErr := depCmd.CombinedOutput()
		cancelDep()
		if depErr != nil {
			return fmt.Errorf("bd dep add %q → %q failed: %w (output: %s)", beadsID, resolvedDep, depErr, strings.TrimSpace(string(depOut)))
		}
	}

	return nil
}
