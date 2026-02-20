package knowledge

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// mulchLookPathFunc is a function variable for looking up executables in PATH.
// Exposed for testing.
var mulchLookPathFunc = exec.LookPath

// mulchExecCommandContextFunc is a function variable for creating exec.Cmd with context.
// Exposed for testing.
var mulchExecCommandContextFunc = exec.CommandContext

// RecordData contains the data to record in the knowledge base.
type RecordData struct {
	Domain  string
	Type    string
	Content string
}

// MulchClient implements Client using the mulch binary.
// If the mulch binary is not found, Prime returns an empty string (non-fatal)
// and Record returns an error.
type MulchClient struct {
	mulchPath    string
	budget       int    // token budget for prime; 0 means flag is omitted
	primeContext string // context name for prime; empty means flag is omitted
}

// NewMulchClient creates a new MulchClient with the given budget and context.
// If the mulch binary is not found, the client is still created but IsEnabled
// returns false, Prime returns empty string, and Record returns an error.
func NewMulchClient(budget int, primeContext string) *MulchClient {
	path, _ := mulchLookPathFunc("mulch")
	return &MulchClient{
		mulchPath:    path,
		budget:       budget,
		primeContext: primeContext,
	}
}

// IsEnabled returns true if the mulch binary is available.
func (m *MulchClient) IsEnabled() bool {
	return m.mulchPath != ""
}

// Prime runs `mulch prime --format markdown [--budget N] [--context C]`
// and returns the stdout output. If mulch is not available, returns empty string (non-fatal).
func (m *MulchClient) Prime() string {
	if m.mulchPath == "" {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	args := []string{"prime", "--format", "markdown"}
	if m.budget > 0 {
		args = append(args, "--budget", strconv.Itoa(m.budget))
	}
	if m.primeContext != "" {
		args = append(args, "--context", m.primeContext)
	}

	cmd := mulchExecCommandContextFunc(ctx, m.mulchPath, args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// Record runs `mulch record <domain> <type> <content>`.
// data must be of type RecordData or *RecordData.
// Returns an error if mulch is not available or if the command fails.
func (m *MulchClient) Record(data any) error {
	if m.mulchPath == "" {
		return errors.New("mulch binary not found")
	}

	var rd RecordData
	switch v := data.(type) {
	case RecordData:
		rd = v
	case *RecordData:
		if v == nil {
			return errors.New("record data is nil")
		}
		rd = *v
	default:
		return fmt.Errorf("unsupported record data type: %T", data)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := mulchExecCommandContextFunc(ctx, m.mulchPath, "record", rd.Domain, rd.Type, rd.Content)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mulch record failed: %w (output: %s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
