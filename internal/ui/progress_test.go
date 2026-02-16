package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewSpinner(t *testing.T) {
	s := NewSpinner("loading...")
	if s == nil {
		t.Fatal("NewSpinner returned nil")
	}
	if s.message != "loading..." {
		t.Errorf("expected message 'loading...', got %q", s.message)
	}
}

func TestSpinnerStartStop(t *testing.T) {
	s := NewSpinner("test")
	var buf bytes.Buffer
	s.w = &buf

	// Force non-animated mode for predictable test output
	origNoColor := os.Getenv("NO_COLOR")
	os.Setenv("NO_COLOR", "1")
	defer func() {
		if origNoColor == "" {
			os.Unsetenv("NO_COLOR")
		} else {
			os.Setenv("NO_COLOR", origNoColor)
		}
	}()

	s.Start()
	s.Stop()

	output := buf.String()
	if !strings.Contains(output, "test") {
		t.Errorf("expected output to contain 'test', got %q", output)
	}
	// In non-animated mode, should use arrow prefix
	if !strings.Contains(output, "→") {
		t.Errorf("expected plain text fallback with arrow, got %q", output)
	}
}

func TestSpinnerDoubleStart(t *testing.T) {
	s := NewSpinner("test")
	var buf bytes.Buffer
	s.w = &buf

	origNoColor := os.Getenv("NO_COLOR")
	os.Setenv("NO_COLOR", "1")
	defer func() {
		if origNoColor == "" {
			os.Unsetenv("NO_COLOR")
		} else {
			os.Setenv("NO_COLOR", origNoColor)
		}
	}()

	s.Start()
	s.Start() // Should be a no-op
	s.Stop()

	// Should only have printed the message once
	output := buf.String()
	count := strings.Count(output, "test")
	if count != 1 {
		t.Errorf("expected message printed once, got %d times", count)
	}
}

func TestSpinnerDoubleStop(t *testing.T) {
	s := NewSpinner("test")
	var buf bytes.Buffer
	s.w = &buf

	origNoColor := os.Getenv("NO_COLOR")
	os.Setenv("NO_COLOR", "1")
	defer func() {
		if origNoColor == "" {
			os.Unsetenv("NO_COLOR")
		} else {
			os.Setenv("NO_COLOR", origNoColor)
		}
	}()

	s.Start()
	s.Stop()
	s.Stop() // Should be a no-op, no panic
}

func TestSpinWhile(t *testing.T) {
	origNoColor := os.Getenv("NO_COLOR")
	os.Setenv("NO_COLOR", "1")
	defer func() {
		if origNoColor == "" {
			os.Unsetenv("NO_COLOR")
		} else {
			os.Setenv("NO_COLOR", origNoColor)
		}
	}()

	called := false
	err := SpinWhile("working...", func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !called {
		t.Error("expected fn to be called")
	}
}

func TestSpinWhileError(t *testing.T) {
	origNoColor := os.Getenv("NO_COLOR")
	os.Setenv("NO_COLOR", "1")
	defer func() {
		if origNoColor == "" {
			os.Unsetenv("NO_COLOR")
		} else {
			os.Setenv("NO_COLOR", origNoColor)
		}
	}()

	expectedErr := os.ErrNotExist
	err := SpinWhile("working...", func() error {
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestNewTaskProgress(t *testing.T) {
	tp := NewTaskProgress(10)
	if tp == nil {
		t.Fatal("NewTaskProgress returned nil")
	}
	if tp.total != 10 {
		t.Errorf("expected total 10, got %d", tp.total)
	}
}

func TestTaskProgressUpdate(t *testing.T) {
	origNoColor := os.Getenv("NO_COLOR")
	os.Setenv("NO_COLOR", "1")
	defer func() {
		if origNoColor == "" {
			os.Unsetenv("NO_COLOR")
		} else {
			os.Setenv("NO_COLOR", origNoColor)
		}
	}()

	tp := NewTaskProgress(3)
	var buf bytes.Buffer
	tp.w = &buf
	tp.animate = false // Force plain text

	tp.Update(1, "first task")
	tp.Update(2, "second task")
	tp.Complete()

	output := buf.String()
	if !strings.Contains(output, "[1/3] first task") {
		t.Errorf("expected '[1/3] first task' in output, got %q", output)
	}
	if !strings.Contains(output, "[2/3] second task") {
		t.Errorf("expected '[2/3] second task' in output, got %q", output)
	}
}

func TestNewStatusLine(t *testing.T) {
	sl := NewStatusLine()
	if sl == nil {
		t.Fatal("NewStatusLine returned nil")
	}
}

func TestStatusLineUpdate(t *testing.T) {
	sl := NewStatusLine()
	var buf bytes.Buffer
	sl.w = &buf
	sl.animate = false // Force plain text

	sl.Update("step %d of %d", 1, 3)
	sl.Update("step %d of %d", 2, 3)
	sl.Clear()

	output := buf.String()
	if !strings.Contains(output, "step 1 of 3") {
		t.Errorf("expected 'step 1 of 3' in output, got %q", output)
	}
	if !strings.Contains(output, "step 2 of 3") {
		t.Errorf("expected 'step 2 of 3' in output, got %q", output)
	}
}

func TestStatusLinePrintln(t *testing.T) {
	sl := NewStatusLine()
	var buf bytes.Buffer
	sl.w = &buf
	sl.animate = false

	sl.Println("result: %s", "done")

	output := buf.String()
	if !strings.Contains(output, "result: done") {
		t.Errorf("expected 'result: done' in output, got %q", output)
	}
	if !strings.HasSuffix(output, "\n") {
		t.Error("expected output to end with newline")
	}
}

func TestTaskHeader(t *testing.T) {
	header := TaskHeader(1, 5, "1.1", "Add feature", "moderate", "sonnet", "claude")

	if !strings.Contains(header, "Task 1/5") {
		t.Error("expected 'Task 1/5' in header")
	}
	if !strings.Contains(header, "1.1") {
		t.Error("expected task ID in header")
	}
	if !strings.Contains(header, "Add feature") {
		t.Error("expected title in header")
	}
	if !strings.Contains(header, "moderate") {
		t.Error("expected complexity in header")
	}
	if !strings.Contains(header, "sonnet") {
		t.Error("expected model in header")
	}
	if !strings.Contains(header, "claude") {
		t.Error("expected agent name in header")
	}
	if !strings.Contains(header, "═") {
		t.Error("expected separator in header")
	}
}

func TestIsTTY(t *testing.T) {
	// Just verify the function doesn't panic
	_ = isTTY()
}

func TestUseAnimation(t *testing.T) {
	// With NO_COLOR set, should return false
	origNoColor := os.Getenv("NO_COLOR")
	os.Setenv("NO_COLOR", "1")

	if useAnimation() {
		t.Error("expected useAnimation() to return false when NO_COLOR is set")
	}

	// Restore
	if origNoColor == "" {
		os.Unsetenv("NO_COLOR")
	} else {
		os.Setenv("NO_COLOR", origNoColor)
	}
}

func TestSpinnerAnimated(t *testing.T) {
	// Test that animated spinner goroutine works without deadlock
	// We can't easily test actual animation in CI, but we can verify
	// the start/stop cycle completes without hanging.
	s := NewSpinner("animated test")
	var buf bytes.Buffer
	s.w = &buf

	// Temporarily allow animation (might not actually animate in test,
	// but we can at least exercise the code path)
	origNoColor := os.Getenv("NO_COLOR")
	os.Unsetenv("NO_COLOR")
	defer func() {
		if origNoColor == "" {
			os.Unsetenv("NO_COLOR")
		} else {
			os.Setenv("NO_COLOR", origNoColor)
		}
	}()

	s.Start()
	time.Sleep(250 * time.Millisecond)
	s.Stop()
	// Just verify it doesn't hang or panic
}
