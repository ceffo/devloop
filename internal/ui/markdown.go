// Package ui provides terminal output utilities including markdown rendering, progress indicators, and TUI components.
package ui

import (
	"os"

	"github.com/charmbracelet/glamour"
	"golang.org/x/term"
)

// RenderMarkdown renders markdown content with terminal styling.
// It respects the NO_COLOR environment variable and terminal width.
func RenderMarkdown(content string) (string, error) {
	// Get terminal width, default to 120 if not available
	width, _, err := term.GetSize(0)
	if err != nil || width == 0 {
		width = 120
	}

	// Build renderer options
	var options []glamour.TermRendererOption

	// Determine style based on NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		// Use a simple style without colors
		options = []glamour.TermRendererOption{
			glamour.WithStandardStyle("ascii"),
			glamour.WithWordWrap(width),
		}
	} else {
		// Use a nice dark style for colored output
		options = []glamour.TermRendererOption{
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(width),
		}
	}

	// Create the renderer
	renderer, err := glamour.NewTermRenderer(options...)
	if err != nil {
		return "", err
	}

	// Render the markdown
	rendered, err := renderer.Render(content)
	if err != nil {
		return "", err
	}

	return rendered, nil
}
