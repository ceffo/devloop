package ui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// RunTUI starts the Bubble Tea TUI program with the given task items.
// It returns the program so the caller can send messages to update task states.
// Returns nil if stdout is not a terminal (caller should use plain output).
func RunTUI(tasks []TaskItem, sessionID, agentName string) (*tea.Program, error) {
	if !IsTTY() {
		return nil, nil
	}

	model := NewTUIModel(tasks, sessionID, agentName)
	p := tea.NewProgram(model, tea.WithAltScreen())
	return p, nil
}

// PlainTaskList prints the task list to stdout in plain text (non-TTY fallback).
func PlainTaskList(tasks []TaskItem, sessionID, agentName string) {
	fmt.Printf("devloop session:%s agent:%s\n", truncate(sessionID, 8), agentName)
	fmt.Println(strings.Repeat("─", 60))
	for _, t := range tasks {
		icon := StatusIcon(t.Status)
		dep := ""
		if t.Status == StatusBlocked && len(t.BlockedBy) > 0 {
			dep = fmt.Sprintf(" (blocked by %s)", strings.Join(t.BlockedBy, ", "))
		}
		fmt.Printf("  %s %s %s%s\n", icon, t.ID, t.Title, dep)
	}
	fmt.Println(strings.Repeat("─", 60))
}

// PlainTaskUpdate prints a task status update in plain text (non-TTY fallback).
func PlainTaskUpdate(taskID string, status TaskStatus, attempt int) {
	icon := StatusIcon(status)
	attemptStr := ""
	if (status == StatusRunning || status == StatusInProgress) && attempt > 0 {
		attemptStr = fmt.Sprintf(" [attempt %d]", attempt)
	}
	fmt.Fprintf(os.Stdout, "  %s %s %s%s\n", icon, taskID, status, attemptStr)
}
