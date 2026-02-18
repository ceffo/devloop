package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TaskStatus represents a task's current status in the TUI.
type TaskStatus string

// Task status constants representing the lifecycle states of a task.
const (
	StatusPending    TaskStatus = "pending"
	StatusRunning    TaskStatus = "running"
	StatusCompleted  TaskStatus = "completed"
	StatusFailed     TaskStatus = "failed"
	StatusBlocked    TaskStatus = "blocked"
	StatusInProgress TaskStatus = "in_progress"
)

// TaskItem is a display-friendly representation of a task for the TUI.
type TaskItem struct {
	ID         string
	Title      string
	Status     TaskStatus
	Complexity string
	BlockedBy  []string
	Attempt    int
	MaxAttempt int
}

// StatusIcon returns the icon for a task status.
func StatusIcon(status TaskStatus) string {
	switch status {
	case StatusCompleted:
		return "✓"
	case StatusRunning, StatusInProgress:
		return "▶"
	case StatusFailed:
		return "✗"
	case StatusBlocked:
		return "⊘"
	case StatusPending:
		return "○"
	default:
		return "○"
	}
}

// statusStyle returns the lipgloss style for a task status.
func statusStyle(status TaskStatus) lipgloss.Style {
	switch status {
	case StatusCompleted:
		return lipgloss.NewStyle().Foreground(colorGreen)
	case StatusRunning, StatusInProgress:
		return lipgloss.NewStyle().Foreground(colorYellow).Bold(true)
	case StatusFailed:
		return lipgloss.NewStyle().Foreground(colorRed)
	case StatusBlocked:
		return lipgloss.NewStyle().Foreground(colorMagenta)
	default:
		return lipgloss.NewStyle().Foreground(colorGray)
	}
}

// TUI messages for communication between executor and TUI.

// TaskUpdateMsg updates a task's status in the TUI.
type TaskUpdateMsg struct {
	TaskID  string
	Status  TaskStatus
	Attempt int
}

// LogMsg appends a log line to the TUI.
type LogMsg struct {
	Line string
}

// DoneMsg signals the TUI to exit.
type DoneMsg struct {
	Err error
}

// AgentStatusMsg updates the active agent and model in the TUI header.
type AgentStatusMsg struct {
	AgentName string
	ModelID   string
}

// AgentOutputMsg appends a line of agent stdout/stderr to the output pane.
type AgentOutputMsg struct {
	Line string
}

// UsageStats holds current API usage metrics for the active agent.
type UsageStats struct {
	// TasksCompleted is the number of tasks successfully completed this session.
	TasksCompleted int
	// TasksFailed is the number of tasks that failed this session.
	TasksFailed int
	// TotalDuration is the total wall-clock time spent in agent calls (seconds).
	TotalDuration int
	// LastTaskDuration is the duration of the most recent agent call (seconds).
	LastTaskDuration int
	// AgentCalls is the total number of agent invocations made this session.
	AgentCalls int
}

// UsageStatsMsg updates the usage stats footer in the TUI.
type UsageStatsMsg struct {
	Stats UsageStats
}

// TUIModel is the Bubble Tea model for the task list view.
type TUIModel struct {
	tasks       []TaskItem
	cursor      int // index of the currently highlighted task
	activeIndex int // index of the currently running task (-1 if none)
	logs        []string
	width       int
	height      int
	done        bool
	finalErr    error
	sessionID   string
	agentName   string
	modelID     string // currently active model

	// Agent output pane
	showOutput  bool     // whether the output pane is visible
	outputLines []string // streaming agent stdout/stderr lines

	// Interrupt confirmation
	confirmingQuit bool        // whether the confirmation dialog is shown
	cancelFunc     func()      // called to cancel executor on confirmed quit

	// Usage stats footer
	usage UsageStats
}

// NewTUIModel creates a new TUI model with the given tasks.
func NewTUIModel(tasks []TaskItem, sessionID, agentName string) TUIModel {
	return TUIModel{
		tasks:       tasks,
		cursor:      0,
		activeIndex: -1,
		logs:        []string{},
		width:       80,
		height:      24,
		sessionID:   sessionID,
		agentName:   agentName,
	}
}

// SetCancelFunc registers a function to be called when the user confirms an interrupt.
// The cancel function should stop the executor gracefully.
func (m *TUIModel) SetCancelFunc(cancel func()) {
	m.cancelFunc = cancel
}

// Init implements tea.Model.
func (m TUIModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m TUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If confirmation dialog is active, handle y/n only
		if m.confirmingQuit {
			switch msg.String() {
			case "y", "Y":
				if m.cancelFunc != nil {
					m.cancelFunc()
				}
				m.done = true
				return m, tea.Quit
			case "n", "N", "esc":
				m.confirmingQuit = false
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			m.confirmingQuit = true
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}
		case "o", "tab":
			m.showOutput = !m.showOutput
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case TaskUpdateMsg:
		for i, t := range m.tasks {
			if t.ID == msg.TaskID {
				m.tasks[i].Status = msg.Status
				m.tasks[i].Attempt = msg.Attempt
				if msg.Status == StatusRunning || msg.Status == StatusInProgress {
					m.activeIndex = i
					m.cursor = i
				}
				break
			}
		}

	case LogMsg:
		m.logs = append(m.logs, msg.Line)

	case AgentOutputMsg:
		m.outputLines = append(m.outputLines, msg.Line)

	case AgentStatusMsg:
		m.agentName = msg.AgentName
		m.modelID = msg.ModelID

	case UsageStatsMsg:
		m.usage = msg.Stats

	case DoneMsg:
		m.done = true
		m.finalErr = msg.Err
		return m, tea.Quit
	}

	return m, nil
}

// View implements tea.Model.
func (m TUIModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorCyan).
		Padding(0, 1)
	header := headerStyle.Render("devloop")

	// Build status info - show agent/model or idle
	statusInfo := "Idle"
	if m.modelID != "" {
		statusInfo = fmt.Sprintf("Agent: %s | Model: %s", m.agentName, m.modelID)
	} else if m.agentName != "" {
		statusInfo = fmt.Sprintf("Agent: %s", m.agentName)
	}

	sessionInfo := lipgloss.NewStyle().Foreground(colorGray).Render(
		fmt.Sprintf(" session:%s | %s", truncate(m.sessionID, 8), statusInfo))
	b.WriteString(header + sessionInfo + "\n")
	b.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render(
		strings.Repeat("─", min(m.width, 80))) + "\n")

	// Reserve space for output pane when visible
	outputPaneHeight := 0
	if m.showOutput {
		outputPaneHeight = max(m.height/3, 5)
	}

	// Calculate how many tasks we can show
	// header (2) + footer separator (1) + usage line (1) + hint (1) + padding (1) = 6
	maxTaskLines := max(m.height-6-outputPaneHeight, 5)

	// Determine visible window around cursor for auto-scroll
	startIdx, endIdx := visibleWindow(len(m.tasks), maxTaskLines, m.cursor)

	// Task list
	for i := startIdx; i < endIdx; i++ {
		task := m.tasks[i]
		line := m.renderTaskLine(task, i == m.cursor)
		b.WriteString(line + "\n")
	}

	// Scroll indicator
	if len(m.tasks) > maxTaskLines {
		scrollInfo := lipgloss.NewStyle().Foreground(colorGray).Render(
			fmt.Sprintf("  (%d/%d tasks)", m.cursor+1, len(m.tasks)))
		b.WriteString(scrollInfo + "\n")
	}

	// Recent log lines
	if len(m.logs) > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render(
			strings.Repeat("─", min(m.width, 80))) + "\n")
		logStart := max(len(m.logs)-3, 0)
		for _, line := range m.logs[logStart:] {
			b.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render(
				truncate(line, m.width-2)) + "\n")
		}
	}

	// Agent output pane
	if m.showOutput {
		b.WriteString(lipgloss.NewStyle().Foreground(colorCyan).Render(
			strings.Repeat("─", min(m.width, 80))) + "\n")
		paneTitle := lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("Agent Output")
		b.WriteString(paneTitle + "\n")

		// Show the last outputPaneHeight-2 lines (title + separator take 2 lines)
		visibleLines := max(outputPaneHeight-2, 1)
		lineStart := max(len(m.outputLines)-visibleLines, 0)
		for _, line := range m.outputLines[lineStart:] {
			b.WriteString(truncate(line, m.width-2) + "\n")
		}
		if len(m.outputLines) == 0 {
			b.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render("  (no output yet)") + "\n")
		}
	}

	// Footer separator
	b.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render(
		strings.Repeat("─", min(m.width, 80))) + "\n")

	if m.confirmingQuit {
		// Confirmation dialog replaces footer
		prompt := lipgloss.NewStyle().Foreground(colorYellow).Bold(true).Render(
			"⚠  Stop execution? Session will be saved and can be resumed. (y/n)")
		b.WriteString(prompt)
	} else {
		// Usage stats line
		usageLine := m.renderUsageStats()
		b.WriteString(usageLine + "\n")
		// Key hints
		footerHint := "q: quit • ↑/↓: navigate • o: toggle output"
		footer := lipgloss.NewStyle().Foreground(colorGray).Render(footerHint)
		b.WriteString(footer)
	}

	return b.String()
}

// renderTaskLine renders a single task line.
func (m TUIModel) renderTaskLine(task TaskItem, selected bool) string {
	icon := StatusIcon(task.Status)
	style := statusStyle(task.Status)

	// Build the task line
	prefix := "  "
	if selected {
		prefix = "▸ "
	}

	idStr := lipgloss.NewStyle().Bold(true).Render(task.ID)
	iconStr := style.Render(icon)
	titleStr := task.Title

	// Attempt info for running tasks
	attemptStr := ""
	if (task.Status == StatusRunning || task.Status == StatusInProgress) && task.MaxAttempt > 0 {
		attemptStr = lipgloss.NewStyle().Foreground(colorGray).Render(
			fmt.Sprintf(" [%d/%d]", task.Attempt, task.MaxAttempt))
	}

	// Dependency info for blocked tasks
	depStr := ""
	if task.Status == StatusBlocked && len(task.BlockedBy) > 0 {
		depStr = lipgloss.NewStyle().Foreground(colorMagenta).Render(
			fmt.Sprintf(" blocked by %s", strings.Join(task.BlockedBy, ", ")))
	}

	line := fmt.Sprintf("%s%s %s %s%s%s", prefix, iconStr, idStr, titleStr, attemptStr, depStr)

	if selected {
		line = lipgloss.NewStyle().Bold(true).Render(line)
	}

	return line
}

// renderUsageStats renders the usage stats line for the footer.
func (m TUIModel) renderUsageStats() string {
	u := m.usage
	if u.AgentCalls == 0 && u.TasksCompleted == 0 && u.TasksFailed == 0 {
		return lipgloss.NewStyle().Foreground(colorGray).Render("Usage: no agent calls yet")
	}

	parts := []string{}

	// Task counts
	if u.TasksCompleted > 0 || u.TasksFailed > 0 {
		total := u.TasksCompleted + u.TasksFailed
		parts = append(parts, fmt.Sprintf("tasks: %d/%d done", u.TasksCompleted, total))
	}

	// Agent calls
	if u.AgentCalls > 0 {
		parts = append(parts, fmt.Sprintf("calls: %d", u.AgentCalls))
	}

	// Duration
	if u.TotalDuration > 0 {
		parts = append(parts, fmt.Sprintf("time: %ds", u.TotalDuration))
	}

	line := "Usage: " + strings.Join(parts, " • ")
	return lipgloss.NewStyle().Foreground(colorCyan).Render(line)
}

// visibleWindow calculates the start/end indices for the visible task window.
func visibleWindow(total, maxVisible, cursor int) (start, end int) {
	if total <= maxVisible {
		return 0, total
	}

	// Keep cursor roughly centered
	half := maxVisible / 2
	start = max(cursor-half, 0)
	end = start + maxVisible
	if end > total {
		end = total
		start = end - maxVisible
	}
	return start, end
}

// truncate shortens a string to the given length.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// Tasks returns the current task items (for testing).
func (m TUIModel) Tasks() []TaskItem {
	return m.tasks
}

// Cursor returns the current cursor position (for testing).
func (m TUIModel) Cursor() int {
	return m.cursor
}

// ActiveIndex returns the index of the currently running task (for testing).
func (m TUIModel) ActiveIndex() int {
	return m.activeIndex
}

// Done returns whether the TUI has finished (for testing).
func (m TUIModel) Done() bool {
	return m.done
}

// ShowOutput returns whether the output pane is currently visible (for testing).
func (m TUIModel) ShowOutput() bool {
	return m.showOutput
}

// OutputLines returns the accumulated agent output lines (for testing).
func (m TUIModel) OutputLines() []string {
	return m.outputLines
}

// ConfirmingQuit returns whether the quit confirmation dialog is active (for testing).
func (m TUIModel) ConfirmingQuit() bool {
	return m.confirmingQuit
}

// Usage returns the current usage stats (for testing).
func (m TUIModel) Usage() UsageStats {
	return m.usage
}
