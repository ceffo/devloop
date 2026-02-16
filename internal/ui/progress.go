package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// isTTY reports whether stdout is a terminal.
func isTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

// useAnimation reports whether animated output should be used.
// Returns false if stdout is not a TTY or NO_COLOR is set.
func useAnimation() bool {
	return isTTY() && os.Getenv("NO_COLOR") == ""
}

// Spinner wraps a blocking operation with an animated spinner.
// Falls back to plain text when stdout is not a TTY or NO_COLOR is set.
type Spinner struct {
	message string
	style   lipgloss.Style
	w       io.Writer

	mu      sync.Mutex
	running bool
	done    chan struct{}
}

// NewSpinner creates a spinner that displays the given message.
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		style:   lipgloss.NewStyle().Foreground(colorCyan),
		w:       os.Stdout,
	}
}

// Start begins the spinner animation. If the terminal does not support
// animation, it prints the message once with a static indicator.
func (s *Spinner) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}
	s.running = true

	if !useAnimation() {
		fmt.Fprintf(s.w, "  → %s\n", s.message)
		return
	}

	s.done = make(chan struct{})

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = s.style

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.done:
				// Clear the spinner line
				fmt.Fprintf(s.w, "\r\033[K")
				return
			case <-ticker.C:
				sp, _ = sp.Update(sp.Tick())
				fmt.Fprintf(s.w, "\r\033[K  %s %s", sp.View(), s.message)
			}
		}
	}()
}

// Stop halts the spinner animation and clears the line.
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}
	s.running = false

	if s.done != nil {
		close(s.done)
		// Give goroutine time to clear the line
		time.Sleep(50 * time.Millisecond)
	}
}

// SpinWhile runs fn while displaying a spinner with the given message.
// Returns whatever fn returns. Falls back to plain text in non-TTY mode.
func SpinWhile(message string, fn func() error) error {
	s := NewSpinner(message)
	s.Start()
	err := fn()
	s.Stop()
	return err
}

// TaskProgress tracks progress across multiple tasks with a visual progress bar.
type TaskProgress struct {
	total     int
	current   int
	w         io.Writer
	bar       progress.Model
	animate   bool
	taskLabel string
}

// NewTaskProgress creates a progress tracker for the given total number of tasks.
func NewTaskProgress(total int) *TaskProgress {
	bar := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	return &TaskProgress{
		total:   total,
		current: 0,
		w:       os.Stdout,
		bar:     bar,
		animate: useAnimation(),
	}
}

// Update sets the current task index and label, then renders the progress bar.
func (tp *TaskProgress) Update(current int, label string) {
	tp.current = current
	tp.taskLabel = label

	if !tp.animate {
		// Plain text fallback
		fmt.Fprintf(tp.w, "[%d/%d] %s\n", current, tp.total, label)
		return
	}

	pct := float64(current) / float64(tp.total)
	barView := tp.bar.ViewAs(pct)
	line := fmt.Sprintf("  %s %d/%d %s", barView, current, tp.total, label)
	fmt.Fprintf(tp.w, "\r\033[K%s", line)
}

// Complete clears the progress bar line (for animated mode).
func (tp *TaskProgress) Complete() {
	if tp.animate {
		fmt.Fprintf(tp.w, "\r\033[K")
	}
}

// StatusLine displays a real-time status line that can be updated.
// Used during the dev loop to show current operation status.
type StatusLine struct {
	w       io.Writer
	animate bool
	mu      sync.Mutex
}

// NewStatusLine creates a new updatable status line.
func NewStatusLine() *StatusLine {
	return &StatusLine{
		w:       os.Stdout,
		animate: useAnimation(),
	}
}

// Update replaces the current status line with new content.
func (sl *StatusLine) Update(format string, args ...any) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	msg := fmt.Sprintf(format, args...)

	if !sl.animate {
		fmt.Fprintf(sl.w, "%s\n", msg)
		return
	}

	fmt.Fprintf(sl.w, "\r\033[K%s", msg)
}

// Clear removes the status line.
func (sl *StatusLine) Clear() {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	if sl.animate {
		fmt.Fprintf(sl.w, "\r\033[K")
	}
}

// Println prints a line, clearing any active status line first.
func (sl *StatusLine) Println(format string, args ...any) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	msg := fmt.Sprintf(format, args...)

	if sl.animate {
		fmt.Fprintf(sl.w, "\r\033[K%s\n", msg)
	} else {
		fmt.Fprintf(sl.w, "%s\n", msg)
	}
}

// TaskHeader prints a formatted task header with progress info.
func TaskHeader(current, total int, taskID, title, complexity, model, agentName string) string {
	var b strings.Builder
	b.WriteString(Separator())
	b.WriteString("\n")
	fmt.Fprintf(&b, "Task %d/%d: %s - %s\n", current, total, taskID, title)
	fmt.Fprintf(&b, "Complexity: %s | Model: %s | Agent: %s\n", complexity, model, agentName)
	b.WriteString(Separator())
	return b.String()
}
