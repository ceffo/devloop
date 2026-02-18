package ui

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// stripANSI removes ANSI escape codes from a string for testing
func stripANSI(s string) string {
	ansiRE := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRE.ReplaceAllString(s, "")
}

func TestRenderMarkdown_Simple(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(string) bool
	}{
		{
			name:    "simple text",
			input:   "Hello, world!",
			wantErr: false,
			check: func(output string) bool {
				return strings.Contains(stripANSI(output), "Hello")
			},
		},
		{
			name:    "heading",
			input:   "# Main Title",
			wantErr: false,
			check: func(output string) bool {
				return strings.Contains(stripANSI(output), "Main") && strings.Contains(stripANSI(output), "Title")
			},
		},
		{
			name:    "bold text",
			input:   "**bold text**",
			wantErr: false,
			check: func(output string) bool {
				return strings.Contains(stripANSI(output), "bold")
			},
		},
		{
			name:    "bullet list",
			input:   "- Item 1\n- Item 2\n- Item 3",
			wantErr: false,
			check: func(output string) bool {
				plain := stripANSI(output)
				return strings.Contains(plain, "Item") && strings.Contains(plain, "1")
			},
		},
		{
			name:    "code block",
			input:   "```\ncode here\n```",
			wantErr: false,
			check: func(output string) bool {
				return strings.Contains(stripANSI(output), "code")
			},
		},
		{
			name:    "inline code",
			input:   "Use `variable` in code",
			wantErr: false,
			check: func(output string) bool {
				return strings.Contains(stripANSI(output), "variable")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := RenderMarkdown(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderMarkdown() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !tt.check(output) {
				t.Errorf("RenderMarkdown() output does not contain expected content.\nPlain: %q", stripANSI(output))
			}
		})
	}
}

func TestRenderMarkdown_NoColor(t *testing.T) {
	// Save original NO_COLOR value
	originalNoColor, noColorSet := os.LookupEnv("NO_COLOR")
	defer func() {
		if noColorSet {
			os.Setenv("NO_COLOR", originalNoColor)
		} else {
			os.Unsetenv("NO_COLOR")
		}
	}()

	// Set NO_COLOR
	os.Setenv("NO_COLOR", "1")

	input := "# Heading\n\n**Bold** and *italic* text"
	output, err := RenderMarkdown(input)
	if err != nil {
		t.Fatalf("RenderMarkdown() unexpected error: %v", err)
	}

	// Check that output is still valid
	plain := stripANSI(output)
	if !strings.Contains(plain, "Heading") {
		t.Errorf("RenderMarkdown() with NO_COLOR failed to render heading.\nGot: %q", plain)
	}
}

func TestRenderMarkdown_EmptyString(t *testing.T) {
	_, err := RenderMarkdown("")
	if err != nil {
		t.Errorf("RenderMarkdown() unexpected error for empty string: %v", err)
	}

	// Empty input should produce empty or minimal output (acceptable)
}

func TestRenderMarkdown_LongContent(t *testing.T) {
	// Create a long markdown content
	var builder strings.Builder
	builder.WriteString("# Documentation\n\n")
	for i := 1; i <= 5; i++ {
		builder.WriteString("## Section " + string(rune(48+i)) + "\n\n")
		builder.WriteString("This is a paragraph with some content.\n\n")
		builder.WriteString("- Point 1\n")
		builder.WriteString("- Point 2\n")
		builder.WriteString("- Point 3\n\n")
	}

	input := builder.String()
	output, err := RenderMarkdown(input)
	if err != nil {
		t.Fatalf("RenderMarkdown() unexpected error: %v", err)
	}

	// Verify content is rendered
	plain := stripANSI(output)
	if !strings.Contains(plain, "Documentation") {
		t.Errorf("RenderMarkdown() failed to render title")
	}

	if !strings.Contains(plain, "Point") {
		t.Errorf("RenderMarkdown() failed to render list items.\nGot: %q", plain)
	}
}

func TestRenderMarkdown_TerminalWidth(t *testing.T) {
	// Test that rendering respects content wrapping
	// Create content longer than typical terminal width
	longText := "This is a very long line of text that should be wrapped by the markdown renderer to fit within the terminal width. " +
		"It contains multiple sentences to ensure proper wrapping behavior. " +
		"The renderer should handle this gracefully without breaking."

	input := longText + "\n\nAnother paragraph here."
	output, err := RenderMarkdown(input)
	if err != nil {
		t.Fatalf("RenderMarkdown() unexpected error: %v", err)
	}

	// Just verify it renders without error and contains the content
	plain := stripANSI(output)
	if !strings.Contains(plain, "long") && !strings.Contains(plain, "wrapped") {
		t.Errorf("RenderMarkdown() failed to render long content.\nGot: %q", plain)
	}
}

