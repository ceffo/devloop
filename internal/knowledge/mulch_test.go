package knowledge

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func TestNewMulchClient_BinaryFound(t *testing.T) {
	orig := mulchLookPathFunc
	defer func() { mulchLookPathFunc = orig }()

	mulchLookPathFunc = func(file string) (string, error) {
		return "/usr/local/bin/mulch", nil
	}

	c := NewMulchClient(0, "")
	if c.mulchPath != "/usr/local/bin/mulch" {
		t.Errorf("expected mulchPath=/usr/local/bin/mulch, got %q", c.mulchPath)
	}
	if !c.IsEnabled() {
		t.Error("expected IsEnabled()=true when binary found")
	}
}

func TestNewMulchClient_BinaryNotFound(t *testing.T) {
	orig := mulchLookPathFunc
	defer func() { mulchLookPathFunc = orig }()

	mulchLookPathFunc = func(file string) (string, error) {
		return "", errors.New("executable file not found in $PATH")
	}

	c := NewMulchClient(0, "")
	if c.mulchPath != "" {
		t.Errorf("expected empty mulchPath when binary not found, got %q", c.mulchPath)
	}
	if c.IsEnabled() {
		t.Error("expected IsEnabled()=false when binary not found")
	}
}

func TestMulchClient_Prime_MissingBinary(t *testing.T) {
	c := &MulchClient{mulchPath: ""}
	result := c.Prime()
	if result != "" {
		t.Errorf("Prime with missing binary should return empty string, got %q", result)
	}
}

func TestMulchClient_Prime_CommandConstruction(t *testing.T) {
	origLook := mulchLookPathFunc
	origExec := mulchExecCommandContextFunc
	defer func() {
		mulchLookPathFunc = origLook
		mulchExecCommandContextFunc = origExec
	}()

	mulchLookPathFunc = func(file string) (string, error) {
		return "/usr/local/bin/mulch", nil
	}

	var capturedName string
	var capturedArgs []string
	mulchExecCommandContextFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		capturedName = name
		capturedArgs = args
		// Return a real command that succeeds with output
		return exec.CommandContext(ctx, "echo", "prime context")
	}

	c := NewMulchClient(4096, "mycontext")
	result := c.Prime()

	if capturedName != "/usr/local/bin/mulch" {
		t.Errorf("expected command name /usr/local/bin/mulch, got %q", capturedName)
	}

	expected := []string{"prime", "--format", "markdown", "--budget", "4096", "--context", "mycontext"}
	if len(capturedArgs) != len(expected) {
		t.Fatalf("expected args %v, got %v", expected, capturedArgs)
	}
	for i, arg := range expected {
		if capturedArgs[i] != arg {
			t.Errorf("arg[%d]: expected %q, got %q", i, arg, capturedArgs[i])
		}
	}

	if result == "" {
		t.Error("Prime should return stdout output")
	}
}

func TestMulchClient_Prime_NoBudgetNoContext(t *testing.T) {
	origExec := mulchExecCommandContextFunc
	defer func() { mulchExecCommandContextFunc = origExec }()

	var capturedArgs []string
	mulchExecCommandContextFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		capturedArgs = args
		return exec.CommandContext(ctx, "echo", "output")
	}

	c := &MulchClient{mulchPath: "/usr/local/bin/mulch", budget: 0, primeContext: ""}
	c.Prime()

	expected := []string{"prime", "--format", "markdown"}
	if len(capturedArgs) != len(expected) {
		t.Fatalf("expected args %v, got %v", expected, capturedArgs)
	}
	for i, arg := range expected {
		if capturedArgs[i] != arg {
			t.Errorf("arg[%d]: expected %q, got %q", i, arg, capturedArgs[i])
		}
	}
}

func TestMulchClient_Prime_OnlyBudget(t *testing.T) {
	origExec := mulchExecCommandContextFunc
	defer func() { mulchExecCommandContextFunc = origExec }()

	var capturedArgs []string
	mulchExecCommandContextFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		capturedArgs = args
		return exec.CommandContext(ctx, "echo", "output")
	}

	c := &MulchClient{mulchPath: "/usr/local/bin/mulch", budget: 2048, primeContext: ""}
	c.Prime()

	expected := []string{"prime", "--format", "markdown", "--budget", "2048"}
	if len(capturedArgs) != len(expected) {
		t.Fatalf("expected args %v, got %v", expected, capturedArgs)
	}
	for i, arg := range expected {
		if capturedArgs[i] != arg {
			t.Errorf("arg[%d]: expected %q, got %q", i, arg, capturedArgs[i])
		}
	}
}

func TestMulchClient_Record_MissingBinary(t *testing.T) {
	c := &MulchClient{mulchPath: ""}
	err := c.Record(RecordData{Domain: "go", Type: "pattern", Content: "use interfaces"})
	if err == nil {
		t.Error("Record with missing binary should return error")
	}
}

func TestMulchClient_Record_CommandConstruction(t *testing.T) {
	origExec := mulchExecCommandContextFunc
	defer func() { mulchExecCommandContextFunc = origExec }()

	var capturedName string
	var capturedArgs []string
	mulchExecCommandContextFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		capturedName = name
		capturedArgs = args
		return exec.CommandContext(ctx, "true") // succeeds with no output
	}

	c := &MulchClient{mulchPath: "/usr/local/bin/mulch"}
	rd := RecordData{Domain: "go", Type: "pattern", Content: "use interfaces"}
	err := c.Record(rd)

	if err != nil {
		t.Errorf("Record should succeed, got error: %v", err)
	}
	if capturedName != "/usr/local/bin/mulch" {
		t.Errorf("expected command /usr/local/bin/mulch, got %q", capturedName)
	}
	expected := []string{"record", "go", "pattern", "use interfaces"}
	if len(capturedArgs) != len(expected) {
		t.Fatalf("expected args %v, got %v", expected, capturedArgs)
	}
	for i, arg := range expected {
		if capturedArgs[i] != arg {
			t.Errorf("arg[%d]: expected %q, got %q", i, arg, capturedArgs[i])
		}
	}
}

func TestMulchClient_Record_PointerRecordData(t *testing.T) {
	origExec := mulchExecCommandContextFunc
	defer func() { mulchExecCommandContextFunc = origExec }()

	mulchExecCommandContextFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "true")
	}

	c := &MulchClient{mulchPath: "/usr/local/bin/mulch"}
	rd := &RecordData{Domain: "go", Type: "pattern", Content: "use interfaces"}
	if err := c.Record(rd); err != nil {
		t.Errorf("Record with *RecordData should succeed, got: %v", err)
	}
}

func TestMulchClient_Record_UnsupportedType(t *testing.T) {
	c := &MulchClient{mulchPath: "/usr/local/bin/mulch"}
	err := c.Record("invalid string type")
	if err == nil {
		t.Error("Record with unsupported type should return error")
	}
}

func TestMulchClient_Record_NilPointer(t *testing.T) {
	c := &MulchClient{mulchPath: "/usr/local/bin/mulch"}
	var rd *RecordData
	err := c.Record(rd)
	if err == nil {
		t.Error("Record with nil *RecordData should return error")
	}
}
