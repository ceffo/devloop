package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestNewTable(t *testing.T) {
	table := NewTable("ID", "Title", "Status")

	if len(table.headers) != 3 {
		t.Errorf("expected 3 headers, got %d", len(table.headers))
	}

	if table.headers[0] != "ID" {
		t.Errorf("expected first header to be 'ID', got '%s'", table.headers[0])
	}

	if len(table.rows) != 0 {
		t.Errorf("expected 0 rows initially, got %d", len(table.rows))
	}
}

func TestTableAppend(t *testing.T) {
	table := NewTable("ID", "Title")
	table.Append("1.1", "Test Task")

	if len(table.rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(table.rows))
	}

	if table.rows[0][0] != "1.1" {
		t.Errorf("expected first cell to be '1.1', got '%s'", table.rows[0][0])
	}

	// Test appending with fewer cells than headers
	table.Append("1.2")
	if len(table.rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(table.rows))
	}
}

func TestTableRender(t *testing.T) {
	table := NewTable("ID", "Title", "Status")
	table.Append("1.1", "Test Task", "pending")
	table.Append("1.2", "Another Task", "completed")

	output := table.Render()

	// Check that output contains headers
	if !strings.Contains(output, "ID") {
		t.Error("output should contain 'ID' header")
	}
	if !strings.Contains(output, "Title") {
		t.Error("output should contain 'Title' header")
	}
	if !strings.Contains(output, "Status") {
		t.Error("output should contain 'Status' header")
	}

	// Check that output contains data
	if !strings.Contains(output, "1.1") {
		t.Error("output should contain '1.1' data")
	}
	if !strings.Contains(output, "Test Task") {
		t.Error("output should contain 'Test Task' data")
	}

	// Check for table structure
	if !strings.Contains(output, "│") {
		t.Error("output should contain table borders")
	}
}

func TestTableSetColumnAlign(t *testing.T) {
	table := NewTable("ID", "Title", "Count")

	// Set right alignment for Count column
	table.SetColumnAlign(2, lipgloss.Right)

	if table.columnAlign[2] != lipgloss.Right {
		t.Error("column 2 should be right-aligned")
	}

	// Test out of bounds
	table.SetColumnAlign(10, lipgloss.Center)
	// Should not panic
}

func TestTableSetHeader(t *testing.T) {
	table := NewTable()
	table.Header("ID", "Title", "Status")

	if len(table.headers) != 3 {
		t.Errorf("expected 3 headers, got %d", len(table.headers))
	}

	if table.headers[1] != "Title" {
		t.Errorf("expected second header to be 'Title', got '%s'", table.headers[1])
	}
}

func TestStatusBadge(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"completed", "completed"},
		{"pending", "pending"},
		{"in_progress", "in_progress"},
		{"failed", "failed"},
		{"blocked", "blocked"},
		{"archived", "archived"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := StatusBadge(tt.status)
			// Check that the status text is present (may be styled)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("StatusBadge(%q) should contain %q, got %q", tt.status, tt.expected, result)
			}
		})
	}
}

func TestComplexityBadge(t *testing.T) {
	tests := []struct {
		complexity string
		expected   string
	}{
		{"simple", "simple"},
		{"moderate", "moderate"},
		{"complex", "complex"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.complexity, func(t *testing.T) {
			result := ComplexityBadge(tt.complexity)
			// Check that the complexity text is present
			if !strings.Contains(result, tt.expected) {
				t.Errorf("ComplexityBadge(%q) should contain %q, got %q", tt.complexity, tt.expected, result)
			}
		})
	}
}

func TestTruncateWithEllipsis(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth int
		contains string
	}{
		{
			name:     "short string",
			input:    "hello",
			maxWidth: 10,
			contains: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			maxWidth: 5,
			contains: "hello",
		},
		{
			name:     "needs truncation",
			input:    "this is a very long string",
			maxWidth: 10,
			contains: "...",
		},
		{
			name:     "very small width",
			input:    "hello",
			maxWidth: 2,
			contains: "..",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateWithEllipsis(tt.input, tt.maxWidth)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("truncateWithEllipsis(%q, %d) should contain %q, got %q",
					tt.input, tt.maxWidth, tt.contains, result)
			}
			// Check that result doesn't exceed maxWidth (accounting for ANSI codes)
			if lipgloss.Width(result) > tt.maxWidth {
				t.Errorf("truncateWithEllipsis(%q, %d) exceeded max width: got width %d",
					tt.input, tt.maxWidth, lipgloss.Width(result))
			}
		})
	}
}

func TestStripAnsi(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no ANSI codes",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "with ANSI codes",
			input:    "\x1b[31mred text\x1b[0m",
			expected: "red text",
		},
		{
			name:     "multiple ANSI codes",
			input:    "\x1b[1m\x1b[31mbold red\x1b[0m",
			expected: "bold red",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripAnsi(tt.input)
			if result != tt.expected {
				t.Errorf("stripAnsi(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTableCalculateWidths(t *testing.T) {
	table := NewTable("ID", "Title", "Status")
	table.Append("1.1", "Short", "pending")
	table.Append("2.1", "This is a much longer title", "completed")

	table.calculateWidths()

	// Check that widths are calculated
	if len(table.maxWidths) != 3 {
		t.Errorf("expected 3 column widths, got %d", len(table.maxWidths))
	}

	// Title column should be wider than "Title" header
	if table.maxWidths[1] < len("Title") {
		t.Errorf("Title column width should be at least %d, got %d", len("Title"), table.maxWidths[1])
	}
}

func TestTableRenderEmpty(t *testing.T) {
	table := NewTable()
	output := table.Render()

	if output != "" {
		t.Errorf("empty table should render empty string, got %q", output)
	}
}

func TestTableRenderNoRows(t *testing.T) {
	table := NewTable("ID", "Title")
	output := table.Render()

	// Should still render headers
	if !strings.Contains(output, "ID") {
		t.Error("output should contain 'ID' header even with no rows")
	}
}
