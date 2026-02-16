package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Table represents a styled table with headers and rows
type Table struct {
	headers     []string
	rows        [][]string
	columnAlign []lipgloss.Position
	maxWidths   []int
	termWidth   int
}

// NewTable creates a new table with the given headers
func NewTable(headers ...string) *Table {
	// Get terminal width, default to 120 if not available
	width, _, err := term.GetSize(0)
	if err != nil || width == 0 {
		width = 120
	}

	return &Table{
		headers:     headers,
		rows:        [][]string{},
		columnAlign: make([]lipgloss.Position, len(headers)),
		maxWidths:   make([]int, len(headers)),
		termWidth:   width,
	}
}

// SetColumnAlign sets the alignment for a specific column
func (t *Table) SetColumnAlign(col int, align lipgloss.Position) {
	if col >= 0 && col < len(t.columnAlign) {
		t.columnAlign[col] = align
	}
}

// Append adds a new row to the table
func (t *Table) Append(cells ...string) {
	// Pad cells if needed
	row := make([]string, len(t.headers))
	copy(row, cells)
	t.rows = append(t.rows, row)
}

// Render outputs the formatted table as a string
func (t *Table) Render() string {
	if len(t.headers) == 0 {
		return ""
	}

	// Calculate column widths
	t.calculateWidths()

	// Build header
	var sb strings.Builder
	sb.WriteString(t.renderHeader())
	sb.WriteString("\n")
	sb.WriteString(t.renderSeparator())
	sb.WriteString("\n")

	// Build rows
	for _, row := range t.rows {
		sb.WriteString(t.renderRow(row))
		sb.WriteString("\n")
	}

	return sb.String()
}

// calculateWidths determines the maximum width for each column
func (t *Table) calculateWidths() {
	// Initialize with header widths
	for i, header := range t.headers {
		t.maxWidths[i] = lipgloss.Width(header)
	}

	// Check row widths
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(t.maxWidths) {
				cellWidth := lipgloss.Width(cell)
				if cellWidth > t.maxWidths[i] {
					t.maxWidths[i] = cellWidth
				}
			}
		}
	}

	// Apply terminal width constraints
	// Reserve space for borders and padding: 3 chars per column (| + space + space)
	// plus 1 for the final border
	totalPadding := (len(t.headers) * 3) + 1
	availableWidth := t.termWidth - totalPadding

	// If total width exceeds available, proportionally reduce columns
	totalWidth := 0
	for _, w := range t.maxWidths {
		totalWidth += w
	}

	if totalWidth > availableWidth {
		ratio := float64(availableWidth) / float64(totalWidth)
		for i := range t.maxWidths {
			t.maxWidths[i] = int(float64(t.maxWidths[i]) * ratio)
			// Ensure minimum width of 8 chars
			if t.maxWidths[i] < 8 {
				t.maxWidths[i] = 8
			}
		}
	}
}

// renderHeader renders the table header
func (t *Table) renderHeader() string {
	var cells []string
	for i, header := range t.headers {
		style := tableHeaderStyle.
			Width(t.maxWidths[i]).
			Align(lipgloss.Center)
		cells = append(cells, style.Render(header))
	}
	return "│ " + strings.Join(cells, " │ ") + " │"
}

// renderSeparator renders a separator line
func (t *Table) renderSeparator() string {
	var parts []string
	for _, width := range t.maxWidths {
		parts = append(parts, strings.Repeat("─", width))
	}
	return "├─" + strings.Join(parts, "─┼─") + "─┤"
}

// renderRow renders a single data row
func (t *Table) renderRow(row []string) string {
	var cells []string
	for i, cell := range row {
		if i >= len(t.maxWidths) {
			break
		}

		// Truncate if needed
		displayCell := cell
		if lipgloss.Width(cell) > t.maxWidths[i] {
			displayCell = truncateWithEllipsis(cell, t.maxWidths[i])
		}

		// Determine alignment (default to left)
		align := lipgloss.Left
		if i < len(t.columnAlign) && t.columnAlign[i] != 0 {
			align = t.columnAlign[i]
		}

		style := tableCellStyle.
			Width(t.maxWidths[i]).
			Align(align)
		cells = append(cells, style.Render(displayCell))
	}
	return "│ " + strings.Join(cells, " │ ") + " │"
}

// truncateWithEllipsis truncates a string to fit within maxWidth, adding ellipsis
func truncateWithEllipsis(s string, maxWidth int) string {
	if maxWidth < 3 {
		return "..."[:maxWidth]
	}

	// Handle styled strings by removing ANSI codes for width calculation
	plainWidth := lipgloss.Width(s)
	if plainWidth <= maxWidth {
		return s
	}

	// For strings with ANSI codes, we need to be more careful
	// Strip the styling, truncate, then we'll return the truncated plain text
	// This is a simplification - in production you might want to preserve styling
	runes := []rune(stripAnsi(s))
	if len(runes) <= maxWidth-3 {
		return string(runes) + "..."
	}
	return string(runes[:maxWidth-3]) + "..."
}

// stripAnsi removes ANSI escape codes from a string
func stripAnsi(s string) string {
	// Simple ANSI stripping - lipgloss.Width handles this but we need the plain string
	var result strings.Builder
	inEscape := false

	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}

	return result.String()
}

// Header is a convenience method to set headers after creation
func (t *Table) Header(headers ...string) {
	t.headers = headers
	t.columnAlign = make([]lipgloss.Position, len(headers))
	t.maxWidths = make([]int, len(headers))
}

// StatusBadge returns a styled status badge with colored background
func StatusBadge(status string) string {
	var style lipgloss.Style

	switch status {
	case "completed":
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorGreen).
			Padding(0, 1).
			Bold(true)
	case "in_progress":
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(colorYellow).
			Padding(0, 1).
			Bold(true)
	case "failed":
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorRed).
			Padding(0, 1).
			Bold(true)
	case "blocked":
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorMagenta).
			Padding(0, 1).
			Bold(true)
	case "archived":
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorGray).
			Padding(0, 1)
	case "pending":
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#E0E0E0")).
			Padding(0, 1)
	default:
		return status
	}

	return style.Render(status)
}

// ComplexityBadge returns a styled complexity badge
func ComplexityBadge(complexity string) string {
	switch complexity {
	case "simple":
		return lipgloss.NewStyle().Foreground(colorGreen).Render("●") + " " + complexity
	case "moderate":
		return lipgloss.NewStyle().Foreground(colorYellow).Render("●") + " " + complexity
	case "complex":
		return lipgloss.NewStyle().Foreground(colorRed).Render("●") + " " + complexity
	default:
		return complexity
	}
}
