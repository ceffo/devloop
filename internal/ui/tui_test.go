package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func sampleTasks() []TaskItem {
	return []TaskItem{
		{ID: "DEV-1", Title: "Setup project", Status: StatusCompleted, Complexity: "simple", Wave: 1},
		{ID: "DEV-2", Title: "Build core", Status: StatusRunning, Complexity: "moderate", Wave: 1, Attempt: 1, MaxAttempt: 3},
		{ID: "DEV-3", Title: "Add tests", Status: StatusPending, Complexity: "simple", Wave: 1},
		{ID: "DEV-4", Title: "Implement feature", Status: StatusBlocked, Complexity: "complex", Wave: 2, BlockedBy: []string{"DEV-2", "DEV-3"}},
		{ID: "DEV-5", Title: "Deploy", Status: StatusFailed, Complexity: "moderate", Wave: 2},
	}
}

func TestNewTUIModel(t *testing.T) {
	tasks := sampleTasks()
	m := NewTUIModel(tasks, "abc12345-session", "claude")

	if len(m.Tasks()) != 5 {
		t.Errorf("expected 5 tasks, got %d", len(m.Tasks()))
	}
	if m.Cursor() != 0 {
		t.Errorf("expected cursor at 0, got %d", m.Cursor())
	}
	if m.ActiveIndex() != -1 {
		t.Errorf("expected activeIndex -1, got %d", m.ActiveIndex())
	}
	if m.Done() {
		t.Error("expected done to be false initially")
	}
}

func TestTUIModelKeyNavigation(t *testing.T) {
	tasks := sampleTasks()
	m := NewTUIModel(tasks, "sess-123", "claude")

	// Move down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(TUIModel)
	if m.Cursor() != 1 {
		t.Errorf("expected cursor 1 after down, got %d", m.Cursor())
	}

	// Move down with 'j'
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(TUIModel)
	if m.Cursor() != 2 {
		t.Errorf("expected cursor 2 after j, got %d", m.Cursor())
	}

	// Move up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(TUIModel)
	if m.Cursor() != 1 {
		t.Errorf("expected cursor 1 after up, got %d", m.Cursor())
	}

	// Move up with 'k'
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(TUIModel)
	if m.Cursor() != 0 {
		t.Errorf("expected cursor 0 after k, got %d", m.Cursor())
	}

	// Should not go below 0
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(TUIModel)
	if m.Cursor() != 0 {
		t.Errorf("expected cursor to stay at 0, got %d", m.Cursor())
	}
}

func TestTUIModelCursorBounds(t *testing.T) {
	tasks := sampleTasks()
	m := NewTUIModel(tasks, "sess-123", "claude")

	// Move to bottom
	for range 10 {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(TUIModel)
	}
	if m.Cursor() != len(tasks)-1 {
		t.Errorf("expected cursor at %d, got %d", len(tasks)-1, m.Cursor())
	}
}

func TestTUIModelTaskUpdate(t *testing.T) {
	tasks := sampleTasks()
	m := NewTUIModel(tasks, "sess-123", "claude")

	// Update DEV-3 to running
	updated, _ := m.Update(TaskUpdateMsg{
		TaskID:  "DEV-3",
		Status:  StatusRunning,
		Attempt: 1,
	})
	m = updated.(TUIModel)

	if m.Tasks()[2].Status != StatusRunning {
		t.Errorf("expected DEV-3 status running, got %s", m.Tasks()[2].Status)
	}
	if m.ActiveIndex() != 2 {
		t.Errorf("expected activeIndex 2, got %d", m.ActiveIndex())
	}
	if m.Cursor() != 2 {
		t.Errorf("expected cursor auto-scrolled to 2, got %d", m.Cursor())
	}

	// Update DEV-3 to completed
	updated, _ = m.Update(TaskUpdateMsg{
		TaskID: "DEV-3",
		Status: StatusCompleted,
	})
	m = updated.(TUIModel)

	if m.Tasks()[2].Status != StatusCompleted {
		t.Errorf("expected DEV-3 status completed, got %s", m.Tasks()[2].Status)
	}
}

func TestTUIModelLogMsg(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")

	updated, _ := m.Update(LogMsg{Line: "Starting task DEV-1"})
	m = updated.(TUIModel)

	updated, _ = m.Update(LogMsg{Line: "Agent running..."})
	m = updated.(TUIModel)

	view := m.View()
	if !strings.Contains(view, "Agent running...") {
		t.Error("expected log message to appear in view")
	}
}

func TestTUIModelDoneMsg(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")

	updated, cmd := m.Update(DoneMsg{Err: nil})
	m = updated.(TUIModel)

	if !m.Done() {
		t.Error("expected done to be true after DoneMsg")
	}
	if cmd == nil {
		t.Error("expected quit command after DoneMsg")
	}
}

func TestTUIModelQuit(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")

	// Pressing 'q' should show the confirmation dialog, not quit immediately
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(TUIModel)

	if m.Done() {
		t.Error("expected done to be false after 'q' (confirmation required)")
	}
	if cmd != nil {
		t.Error("expected no quit command after 'q' (confirmation required)")
	}
	if !m.ConfirmingQuit() {
		t.Error("expected confirmingQuit to be true after 'q'")
	}
}

func TestTUIModelQuitConfirmYes(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")

	// Trigger confirmation dialog
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(TUIModel)

	// Confirm with 'y'
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(TUIModel)

	if !m.Done() {
		t.Error("expected done to be true after confirming quit with 'y'")
	}
	if cmd == nil {
		t.Error("expected quit command after confirming with 'y'")
	}
}

func TestTUIModelQuitConfirmNo(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")

	// Trigger confirmation dialog
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(TUIModel)

	// Decline with 'n'
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(TUIModel)

	if m.Done() {
		t.Error("expected done to be false after declining quit with 'n'")
	}
	if cmd != nil {
		t.Error("expected no quit command after declining with 'n'")
	}
	if m.ConfirmingQuit() {
		t.Error("expected confirmingQuit to be false after declining")
	}
}

func TestTUIModelQuitConfirmEscape(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")

	// Trigger confirmation dialog
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(TUIModel)

	// Dismiss with Escape
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(TUIModel)

	if m.Done() {
		t.Error("expected done to be false after pressing Escape")
	}
	if m.ConfirmingQuit() {
		t.Error("expected confirmingQuit to be false after pressing Escape")
	}
}

func TestTUIModelQuitWithCancelFunc(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")

	cancelCalled := false
	m.SetCancelFunc(func() { cancelCalled = true })

	// Trigger confirmation dialog
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(TUIModel)

	// Confirm with 'y'
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if !cancelCalled {
		t.Error("expected cancel function to be called when confirming quit")
	}
}

func TestTUIModelConfirmDialogView(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")

	// Trigger confirmation dialog
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(TUIModel)

	view := m.View()
	if !strings.Contains(view, "Stop execution?") {
		t.Error("expected confirmation prompt to appear in view when confirmingQuit is true")
	}
	// Normal footer should not appear when confirming
	if strings.Contains(view, "q: quit") {
		t.Error("expected normal footer to be replaced by confirmation dialog")
	}
}

func TestTUIModelNoNavigationDuringConfirm(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")

	// Trigger confirmation dialog
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(TUIModel)

	initialCursor := m.Cursor()

	// Navigation keys should be ignored during confirmation
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(TUIModel)

	if m.Cursor() != initialCursor {
		t.Error("expected cursor not to move while confirmation dialog is active")
	}
	if !m.ConfirmingQuit() {
		t.Error("expected confirmingQuit to remain true after ignored navigation key")
	}
}

func TestTUIModelWindowResize(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(TUIModel)

	if m.width != 120 {
		t.Errorf("expected width 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("expected height 40, got %d", m.height)
	}
}

func TestTUIModelViewContent(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")

	view := m.View()

	// Check task IDs appear
	for _, task := range sampleTasks() {
		if !strings.Contains(view, task.ID) {
			t.Errorf("expected view to contain task ID %s", task.ID)
		}
	}

	// Check blocked task shows dependency info
	if !strings.Contains(view, "blocked by DEV-2, DEV-3") {
		t.Error("expected blocked task to show dependency info")
	}

	// Check footer
	if !strings.Contains(view, "q: quit") {
		t.Error("expected footer with quit instruction")
	}

	// Check header
	if !strings.Contains(view, "devloop") {
		t.Error("expected header with devloop name")
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status TaskStatus
		icon   string
	}{
		{StatusPending, "○"},
		{StatusRunning, "▶"},
		{StatusInProgress, "▶"},
		{StatusCompleted, "✓"},
		{StatusFailed, "✗"},
		{StatusBlocked, "⊘"},
	}

	for _, tt := range tests {
		got := StatusIcon(tt.status)
		if got != tt.icon {
			t.Errorf("StatusIcon(%s) = %s, want %s", tt.status, got, tt.icon)
		}
	}
}

func TestVisibleWindow(t *testing.T) {
	tests := []struct {
		name       string
		total      int
		maxVisible int
		cursor     int
		wantStart  int
		wantEnd    int
	}{
		{"all visible", 5, 10, 2, 0, 5},
		{"scroll to start", 20, 5, 0, 0, 5},
		{"scroll to middle", 20, 5, 10, 8, 13},
		{"scroll to end", 20, 5, 19, 15, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := visibleWindow(tt.total, tt.maxVisible, tt.cursor)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("visibleWindow(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.total, tt.maxVisible, tt.cursor, start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"hi", 2, "hi"},
		{"hello", 3, "hel"},
		{"hello", 0, ""},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestTUIModelInit(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected Init() to return nil cmd")
	}
}

func TestTUIModelViewWhenDone(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")
	updated, _ := m.Update(DoneMsg{Err: nil})
	m = updated.(TUIModel)

	view := m.View()
	if view != "" {
		t.Errorf("expected empty view when done, got %q", view)
	}
}

func TestTUIModelOutputPaneToggle(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")

	// Output pane starts hidden
	if m.ShowOutput() {
		t.Error("expected output pane to be hidden initially")
	}

	// Toggle on with 'o'
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = updated.(TUIModel)
	if !m.ShowOutput() {
		t.Error("expected output pane to be visible after pressing 'o'")
	}

	// Toggle off with 'o' again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = updated.(TUIModel)
	if m.ShowOutput() {
		t.Error("expected output pane to be hidden after pressing 'o' again")
	}

	// Toggle on with Tab
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(TUIModel)
	if !m.ShowOutput() {
		t.Error("expected output pane to be visible after pressing Tab")
	}
}

func TestTUIModelAgentOutputMsg(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")

	// Send some agent output lines
	updated, _ := m.Update(AgentOutputMsg{Line: "line one"})
	m = updated.(TUIModel)
	updated, _ = m.Update(AgentOutputMsg{Line: "line two"})
	m = updated.(TUIModel)

	lines := m.OutputLines()
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d", len(lines))
	}
	if lines[0] != "line one" {
		t.Errorf("expected first line 'line one', got %q", lines[0])
	}
	if lines[1] != "line two" {
		t.Errorf("expected second line 'line two', got %q", lines[1])
	}
}

func TestTUIModelOutputPaneVisible(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")
	updated0, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated0.(TUIModel)

	// Add agent output
	updated, _ := m.Update(AgentOutputMsg{Line: "agent stdout line"})
	m = updated.(TUIModel)

	// Enable output pane
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = updated.(TUIModel)

	view := m.View()
	if !strings.Contains(view, "Agent Output") {
		t.Error("expected 'Agent Output' header in view when pane is visible")
	}
	if !strings.Contains(view, "agent stdout line") {
		t.Error("expected agent output line to appear in view when pane is visible")
	}

	// Footer should mention the toggle key
	if !strings.Contains(view, "o: toggle output") {
		t.Error("expected footer to show toggle key hint")
	}
}

func TestTUIModelOutputPaneHidden(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")

	// Add agent output but keep pane hidden
	updated, _ := m.Update(AgentOutputMsg{Line: "hidden output"})
	m = updated.(TUIModel)

	view := m.View()
	if strings.Contains(view, "hidden output") {
		t.Error("expected agent output to be hidden when pane is not shown")
	}
}

func TestTUIModelOutputPaneTaskListRemains(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")
	updated0, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated0.(TUIModel)

	// Enable output pane
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = updated.(TUIModel)

	view := m.View()

	// Task list should still be visible
	for _, task := range sampleTasks() {
		if !strings.Contains(view, task.ID) {
			t.Errorf("expected task %s to remain visible when output pane is open", task.ID)
		}
	}
}

func TestTUIModelUsageStatsMsg(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")

	// No stats yet: footer shows "no agent calls yet"
	view := m.View()
	if !strings.Contains(view, "no agent calls yet") {
		t.Error("expected 'no agent calls yet' in footer before any stats")
	}

	// Send usage stats
	updated, _ := m.Update(UsageStatsMsg{
		Stats: UsageStats{
			TasksCompleted:   2,
			TasksFailed:      1,
			TotalDuration:    45,
			LastTaskDuration: 15,
			AgentCalls:       3,
		},
	})
	m = updated.(TUIModel)

	// Verify stored
	u := m.Usage()
	if u.TasksCompleted != 2 {
		t.Errorf("expected TasksCompleted 2, got %d", u.TasksCompleted)
	}
	if u.AgentCalls != 3 {
		t.Errorf("expected AgentCalls 3, got %d", u.AgentCalls)
	}
	if u.TotalDuration != 45 {
		t.Errorf("expected TotalDuration 45, got %d", u.TotalDuration)
	}

	// Verify footer content
	view = m.View()
	if !strings.Contains(view, "Usage:") {
		t.Error("expected 'Usage:' in footer after stats update")
	}
	if !strings.Contains(view, "tasks: 2/3 done") {
		t.Errorf("expected 'tasks: 2/3 done' in footer, view: %q", view)
	}
	if !strings.Contains(view, "calls: 3") {
		t.Error("expected 'calls: 3' in footer")
	}
	if !strings.Contains(view, "time: 45s") {
		t.Error("expected 'time: 45s' in footer")
	}
}

func TestTUIModelUsageStatsAlwaysVisible(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")
	updated0, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated0.(TUIModel)

	// Set stats
	updated, _ := m.Update(UsageStatsMsg{
		Stats: UsageStats{AgentCalls: 1, TasksCompleted: 1, TotalDuration: 10},
	})
	m = updated.(TUIModel)

	view := m.View()

	// Footer usage line should always appear (key hint and usage)
	if !strings.Contains(view, "Usage:") {
		t.Error("expected usage footer to be always visible")
	}
	if !strings.Contains(view, "q: quit") {
		t.Error("expected key hint footer to still be present")
	}
}

func TestTUIModelUsageStatsHiddenDuringConfirm(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")

	// Set some stats
	updated, _ := m.Update(UsageStatsMsg{
		Stats: UsageStats{AgentCalls: 2, TasksCompleted: 1, TotalDuration: 20},
	})
	m = updated.(TUIModel)

	// Trigger confirmation dialog
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(TUIModel)

	view := m.View()
	// Confirmation dialog replaces footer
	if !strings.Contains(view, "Stop execution?") {
		t.Error("expected confirmation dialog when confirmingQuit is true")
	}
}

func TestTUIModelAgentStatusUpdate(t *testing.T) {
	m := NewTUIModel(sampleTasks(), "sess-123", "claude")

	// Update with agent and model
	updated, _ := m.Update(AgentStatusMsg{
		AgentName: "claude",
		ModelID:   "opus-4",
	})
	m = updated.(TUIModel)

	view := m.View()
	if !strings.Contains(view, "Agent: claude") {
		t.Error("expected view to contain agent name after update")
	}
	if !strings.Contains(view, "Model: opus-4") {
		t.Error("expected view to contain model ID after update")
	}

	// Update to idle (empty agent/model)
	updated, _ = m.Update(AgentStatusMsg{
		AgentName: "",
		ModelID:   "",
	})
	m = updated.(TUIModel)

	view = m.View()
	if !strings.Contains(view, "Idle") {
		t.Error("expected view to show Idle when agent/model are empty")
	}
}
