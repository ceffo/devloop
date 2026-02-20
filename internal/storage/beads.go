package storage

import (
	"fmt"
	"os/exec"

	"github.com/ceffo/devloop/internal/config"
)

// lookPathFunc is a function variable for looking up executables in PATH
// It's exposed for testing purposes
var lookPathFunc = exec.LookPath

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
