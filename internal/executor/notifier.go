package executor

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ceffo/devloop/internal/storage"
	"github.com/ceffo/devloop/internal/ui"
)

// Notifier abstracts how the executor communicates task progress.
// It has two implementations: TUI-based (interactive terminal) and plain text.
type Notifier interface {
	// Init sets up the notifier with the full task list and session info.
	Init(tasks []*storage.Task, sessionID, agentName string)
	// TaskStarted signals that a task has begun execution.
	TaskStarted(taskID string, attempt, maxAttempts int)
	// TaskCompleted signals that a task finished successfully.
	TaskCompleted(taskID string)
	// TaskFailed signals that a task failed.
	TaskFailed(taskID string)
	// AgentStatusUpdate updates the active agent and model information.
	AgentStatusUpdate(agentName, modelID string)
	// Log emits a log message.
	Log(msg string)
	// Done signals that execution is complete. Called before summary.
	Done(err error)
	// Wait blocks until the notifier has finished (TUI closed, etc).
	Wait() error
}

// tuiNotifier drives the Bubble Tea TUI.
type tuiNotifier struct {
	program *tea.Program
	errCh   chan error
}

func newTUINotifier() *tuiNotifier {
	return &tuiNotifier{
		errCh: make(chan error, 1),
	}
}

func (n *tuiNotifier) Init(tasks []*storage.Task, sessionID, agentName string) {
	items := storageTasks2TUIItems(tasks)
	model := ui.NewTUIModel(items, sessionID, agentName)
	n.program = tea.NewProgram(model, tea.WithAltScreen())

	// Run the TUI program in a background goroutine
	go func() {
		_, err := n.program.Run()
		n.errCh <- err
	}()
}

func (n *tuiNotifier) TaskStarted(taskID string, attempt, maxAttempts int) {
	if n.program != nil {
		n.program.Send(ui.TaskUpdateMsg{
			TaskID:  taskID,
			Status:  ui.StatusRunning,
			Attempt: attempt,
		})
		n.program.Send(ui.LogMsg{
			Line: fmt.Sprintf("Starting %s (attempt %d/%d)", taskID, attempt, maxAttempts),
		})
	}
}

func (n *tuiNotifier) TaskCompleted(taskID string) {
	if n.program != nil {
		n.program.Send(ui.TaskUpdateMsg{
			TaskID: taskID,
			Status: ui.StatusCompleted,
		})
		n.program.Send(ui.LogMsg{
			Line: fmt.Sprintf("Completed %s", taskID),
		})
	}
}

func (n *tuiNotifier) TaskFailed(taskID string) {
	if n.program != nil {
		n.program.Send(ui.TaskUpdateMsg{
			TaskID: taskID,
			Status: ui.StatusFailed,
		})
		n.program.Send(ui.LogMsg{
			Line: fmt.Sprintf("Failed %s", taskID),
		})
	}
}

func (n *tuiNotifier) AgentStatusUpdate(agentName, modelID string) {
	if n.program != nil {
		n.program.Send(ui.AgentStatusMsg{
			AgentName: agentName,
			ModelID:   modelID,
		})
	}
}

func (n *tuiNotifier) Log(msg string) {
	if n.program != nil {
		n.program.Send(ui.LogMsg{Line: msg})
	}
}

func (n *tuiNotifier) Done(err error) {
	if n.program != nil {
		n.program.Send(ui.DoneMsg{Err: err})
	}
}

func (n *tuiNotifier) Wait() error {
	return <-n.errCh
}

// plainNotifier prints plain text output (non-TTY fallback).
type plainNotifier struct{}

func newPlainNotifier() *plainNotifier {
	return &plainNotifier{}
}

func (n *plainNotifier) Init(tasks []*storage.Task, sessionID, agentName string) {
	items := storageTasks2TUIItems(tasks)
	ui.PlainTaskList(items, sessionID, agentName)
}

func (n *plainNotifier) TaskStarted(taskID string, attempt, maxAttempts int) {
	ui.PlainTaskUpdate(taskID, ui.StatusRunning, attempt)
}

func (n *plainNotifier) TaskCompleted(taskID string) {
	ui.PlainTaskUpdate(taskID, ui.StatusCompleted, 0)
}

func (n *plainNotifier) TaskFailed(taskID string) {
	ui.PlainTaskUpdate(taskID, ui.StatusFailed, 0)
}

func (n *plainNotifier) AgentStatusUpdate(agentName, modelID string) {
	// Plain notifier doesn't need to update anything since status is shown inline
}

func (n *plainNotifier) Log(msg string) {
	fmt.Println(msg)
}

func (n *plainNotifier) Done(_ error) {}

func (n *plainNotifier) Wait() error { return nil }

// NewNotifier returns the appropriate notifier based on whether stdout is a TTY.
func NewNotifier() Notifier {
	if ui.IsTTY() {
		return newTUINotifier()
	}
	return newPlainNotifier()
}

// storageTasks2TUIItems converts storage tasks to TUI task items.
func storageTasks2TUIItems(tasks []*storage.Task) []ui.TaskItem {
	items := make([]ui.TaskItem, len(tasks))
	for i, t := range tasks {
		status := ui.StatusPending
		switch t.Status {
		case "completed":
			status = ui.StatusCompleted
		case "in_progress":
			status = ui.StatusRunning
		case "failed":
			status = ui.StatusFailed
		case "blocked":
			status = ui.StatusBlocked
		}
		items[i] = ui.TaskItem{
			ID:         t.ID,
			Title:      t.Title,
			Status:     status,
			Complexity: t.Complexity,
			Wave:       t.Wave,
			BlockedBy:  t.BlockedBy,
			MaxAttempt: t.Metadata.MaxAttempts,
		}
	}
	return items
}
