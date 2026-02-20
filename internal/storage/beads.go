package storage

import (
	"context"
	"fmt"
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
