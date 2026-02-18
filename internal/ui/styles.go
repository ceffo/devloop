package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Color palette
	colorGreen   = lipgloss.Color("#00C853")
	colorRed     = lipgloss.Color("#D50000")
	colorYellow  = lipgloss.Color("#FFD600")
	colorBlue    = lipgloss.Color("#2962FF")
	colorMagenta = lipgloss.Color("#AA00FF")
	colorCyan    = lipgloss.Color("#00B8D4")
	colorGray    = lipgloss.Color("#616161")

	// Base text styles
	baseStyle = lipgloss.NewStyle()

	// Heading styles
	h1Style = baseStyle.Bold(true).Foreground(colorCyan).MarginBottom(1)
	h2Style = baseStyle.Bold(true).Foreground(colorBlue)
	h3Style = baseStyle.Bold(true)

	// Message styles
	successStyle = baseStyle.Foreground(colorGreen).Bold(true)
	errorStyle   = baseStyle.Foreground(colorRed).Bold(true)
	warningStyle = baseStyle.Foreground(colorYellow).Bold(true)
	infoStyle    = baseStyle.Foreground(colorBlue)

	// Label and value styles
	labelStyle = baseStyle.Foreground(colorCyan).Bold(true)
	valueStyle = baseStyle

	// Status styles
	statusCompletedStyle  = baseStyle.Foreground(colorGreen)
	statusInProgressStyle = baseStyle.Foreground(colorYellow)
	statusFailedStyle     = baseStyle.Foreground(colorRed)
	statusBlockedStyle    = baseStyle.Foreground(colorMagenta)
	statusArchivedStyle   = baseStyle.Foreground(colorGray)
	statusPendingStyle    = baseStyle

	// Border styles
	borderStyle = baseStyle.
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorCyan).
			Padding(1, 2)

	sectionBorderStyle = baseStyle.
				Border(lipgloss.DoubleBorder()).
				BorderForeground(colorBlue).
				Padding(0, 2)

	// Table-related styles
	tableHeaderStyle = baseStyle.Bold(true).Foreground(colorCyan)
	tableCellStyle   = baseStyle

	// Result styles
	resultPassedStyle = baseStyle.Foreground(colorGreen)
	resultFailedStyle = baseStyle.Foreground(colorRed)
)

func init() {
	// Check for NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		disableColors()
	}
}

// disableColors removes all color and styling
func disableColors() {
	baseStyle = lipgloss.NewStyle()
	h1Style = baseStyle
	h2Style = baseStyle
	h3Style = baseStyle
	successStyle = baseStyle
	errorStyle = baseStyle
	warningStyle = baseStyle
	infoStyle = baseStyle
	labelStyle = baseStyle
	valueStyle = baseStyle
	statusCompletedStyle = baseStyle
	statusInProgressStyle = baseStyle
	statusFailedStyle = baseStyle
	statusBlockedStyle = baseStyle
	statusArchivedStyle = baseStyle
	statusPendingStyle = baseStyle
	borderStyle = baseStyle
	sectionBorderStyle = baseStyle
	tableHeaderStyle = baseStyle
	tableCellStyle = baseStyle
	resultPassedStyle = baseStyle
	resultFailedStyle = baseStyle
}

// Success returns a formatted success message
func Success(msg string) string {
	return successStyle.Render("✓ " + msg)
}

// Error returns a formatted error message
func Error(msg string) string {
	return errorStyle.Render("✗ " + msg)
}

// Warning returns a formatted warning message
func Warning(msg string) string {
	return warningStyle.Render("⚠ " + msg)
}

// Info returns a formatted info message
func Info(msg string) string {
	return infoStyle.Render("ℹ " + msg)
}

// Section returns a text with section styling (heading + border)
func Section(title string) string {
	return sectionBorderStyle.Render(h2Style.Render(title))
}

// Heading1 returns a top-level heading
func Heading1(text string) string {
	return h1Style.Render(text)
}

// Heading2 returns a second-level heading
func Heading2(text string) string {
	return h2Style.Render(text)
}

// Heading3 returns a third-level heading
func Heading3(text string) string {
	return h3Style.Render(text)
}

// Label returns a formatted label
func Label(text string) string {
	return labelStyle.Render(text + ":")
}

// Value returns a formatted value
func Value(text string) string {
	return valueStyle.Render(text)
}

// LabelValue returns a formatted label-value pair
func LabelValue(label, value string) string {
	return Label(label) + " " + Value(value)
}

// Status returns a color-coded status string
func Status(status string) string {
	switch status {
	case "completed":
		return statusCompletedStyle.Render(status)
	case "in_progress":
		return statusInProgressStyle.Render(status)
	case "failed":
		return statusFailedStyle.Render(status)
	case "blocked":
		return statusBlockedStyle.Render(status)
	case "archived":
		return statusArchivedStyle.Render(status)
	case "pending":
		return statusPendingStyle.Render(status)
	default:
		return status
	}
}

// Result returns a color-coded result string
func Result(result string) string {
	if result == "passed" {
		return resultPassedStyle.Render(result)
	}
	return resultFailedStyle.Render(result)
}

// Border wraps text in a bordered box
func Border(text string) string {
	return borderStyle.Render(text)
}

// TableHeader returns styled table header text
func TableHeader(text string) string {
	return tableHeaderStyle.Render(text)
}

// TableCell returns styled table cell text
func TableCell(text string) string {
	return tableCellStyle.Render(text)
}

// Separator returns a visual separator line
func Separator() string {
	return "═══════════════════════════════════════════════════════════"
}

// StatusIconStr returns a plain-text icon for the given status string.
func StatusIconStr(status string) string {
	switch status {
	case "completed":
		return "✓"
	case "in_progress":
		return "▶"
	case "failed":
		return "✗"
	case "blocked":
		return "⊘"
	case "pending":
		return "○"
	case "archived":
		return "▪"
	default:
		return "○"
	}
}
