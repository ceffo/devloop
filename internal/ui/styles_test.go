package ui

import (
	"os"
	"strings"
	"testing"
)

func TestSuccess(t *testing.T) {
	result := Success("task completed")
	if !strings.Contains(result, "✓") {
		t.Errorf("Success should include checkmark symbol")
	}
	if !strings.Contains(result, "task completed") {
		t.Errorf("Success should include message text")
	}
}

func TestError(t *testing.T) {
	result := Error("task failed")
	if !strings.Contains(result, "✗") {
		t.Errorf("Error should include X symbol")
	}
	if !strings.Contains(result, "task failed") {
		t.Errorf("Error should include message text")
	}
}

func TestWarning(t *testing.T) {
	result := Warning("potential issue")
	if !strings.Contains(result, "⚠") {
		t.Errorf("Warning should include warning symbol")
	}
	if !strings.Contains(result, "potential issue") {
		t.Errorf("Warning should include message text")
	}
}

func TestInfo(t *testing.T) {
	result := Info("information")
	if !strings.Contains(result, "ℹ") {
		t.Errorf("Info should include info symbol")
	}
	if !strings.Contains(result, "information") {
		t.Errorf("Info should include message text")
	}
}

func TestSection(t *testing.T) {
	result := Section("My Section")
	if !strings.Contains(result, "My Section") {
		t.Errorf("Section should include title text")
	}
}

func TestHeadings(t *testing.T) {
	tests := []struct {
		name string
		fn   func(string) string
		text string
	}{
		{"Heading1", Heading1, "Main Title"},
		{"Heading2", Heading2, "Subtitle"},
		{"Heading3", Heading3, "Section"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(tt.text)
			if !strings.Contains(result, tt.text) {
				t.Errorf("%s should include text: %s", tt.name, tt.text)
			}
		})
	}
}

func TestLabel(t *testing.T) {
	result := Label("Status")
	if !strings.Contains(result, "Status:") {
		t.Errorf("Label should include text with colon")
	}
}

func TestValue(t *testing.T) {
	result := Value("completed")
	if !strings.Contains(result, "completed") {
		t.Errorf("Value should include text")
	}
}

func TestLabelValue(t *testing.T) {
	result := LabelValue("Status", "completed")
	if !strings.Contains(result, "Status:") {
		t.Errorf("LabelValue should include label with colon")
	}
	if !strings.Contains(result, "completed") {
		t.Errorf("LabelValue should include value")
	}
}

func TestStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"completed", "completed"},
		{"in_progress", "in_progress"},
		{"failed", "failed"},
		{"blocked", "blocked"},
		{"archived", "archived"},
		{"pending", "pending"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := Status(tt.status)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Status(%s) should contain %s, got %s", tt.status, tt.expected, result)
			}
		})
	}
}

func TestResult(t *testing.T) {
	tests := []struct {
		result   string
		expected string
	}{
		{"passed", "passed"},
		{"failed", "failed"},
		{"error", "error"},
	}

	for _, tt := range tests {
		t.Run(tt.result, func(t *testing.T) {
			result := Result(tt.result)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Result(%s) should contain %s", tt.result, tt.expected)
			}
		})
	}
}

func TestBorder(t *testing.T) {
	result := Border("content")
	if !strings.Contains(result, "content") {
		t.Errorf("Border should include content text")
	}
}

func TestTableHeader(t *testing.T) {
	result := TableHeader("Column")
	if !strings.Contains(result, "Column") {
		t.Errorf("TableHeader should include text")
	}
}

func TestTableCell(t *testing.T) {
	result := TableCell("value")
	if !strings.Contains(result, "value") {
		t.Errorf("TableCell should include text")
	}
}

func TestSeparator(t *testing.T) {
	result := Separator()
	if len(result) == 0 {
		t.Errorf("Separator should return non-empty string")
	}
	if !strings.Contains(result, "═") {
		t.Errorf("Separator should contain double line characters")
	}
}

func TestNO_COLOR(t *testing.T) {
	// Save original env
	originalValue := os.Getenv("NO_COLOR")
	defer func() {
		if originalValue == "" {
			os.Unsetenv("NO_COLOR")
		} else {
			os.Setenv("NO_COLOR", originalValue)
		}
		// Re-initialize styles with original setting
		if originalValue == "" {
			// Colors should be enabled
			initializeColors()
		} else {
			disableColors()
		}
	}()

	// Test with NO_COLOR set
	os.Setenv("NO_COLOR", "1")
	disableColors()

	// Test that functions still work but without ANSI codes
	successMsg := Success("test")
	// Should contain the text but be plain (no ANSI escape sequences)
	if !strings.Contains(successMsg, "✓") {
		t.Errorf("Success should include checkmark even with NO_COLOR")
	}
	if !strings.Contains(successMsg, "test") {
		t.Errorf("Success should include message text with NO_COLOR")
	}

	// Check that no ANSI codes are present (basic check)
	// ANSI codes typically start with ESC character (\x1b or \033)
	if strings.Contains(successMsg, "\x1b") || strings.Contains(successMsg, "\033") {
		t.Errorf("Success should not contain ANSI codes when NO_COLOR is set")
	}

	errorMsg := Error("test")
	if !strings.Contains(errorMsg, "✗") {
		t.Errorf("Error should include X symbol even with NO_COLOR")
	}
	if strings.Contains(errorMsg, "\x1b") || strings.Contains(errorMsg, "\033") {
		t.Errorf("Error should not contain ANSI codes when NO_COLOR is set")
	}

	statusMsg := Status("completed")
	if !strings.Contains(statusMsg, "completed") {
		t.Errorf("Status should include status text with NO_COLOR")
	}
	if strings.Contains(statusMsg, "\x1b") || strings.Contains(statusMsg, "\033") {
		t.Errorf("Status should not contain ANSI codes when NO_COLOR is set")
	}
}

func TestStatusIconStr(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"completed", "✓"},
		{"in_progress", "▶"},
		{"failed", "✗"},
		{"blocked", "⊘"},
		{"pending", "○"},
		{"archived", "▪"},
		{"unknown", "○"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := StatusIconStr(tt.status)
			if result != tt.expected {
				t.Errorf("StatusIconStr(%q) = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}

// Helper to re-initialize colors (for testing)
func initializeColors() {
	successStyle = baseStyle.Foreground(colorGreen).Bold(true)
	errorStyle = baseStyle.Foreground(colorRed).Bold(true)
	warningStyle = baseStyle.Foreground(colorYellow).Bold(true)
	infoStyle = baseStyle.Foreground(colorBlue)
	labelStyle = baseStyle.Foreground(colorCyan).Bold(true)
	valueStyle = baseStyle
	statusCompletedStyle = baseStyle.Foreground(colorGreen)
	statusInProgressStyle = baseStyle.Foreground(colorYellow)
	statusFailedStyle = baseStyle.Foreground(colorRed)
	statusBlockedStyle = baseStyle.Foreground(colorMagenta)
	statusArchivedStyle = baseStyle.Foreground(colorGray)
	statusPendingStyle = baseStyle
}
