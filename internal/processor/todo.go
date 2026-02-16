package processor

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// TodoItem represents a single TODO item parsed from a markdown file.
type TodoItem struct {
	ID       string
	Category string
	Content  string
	Priority string
}

// ParseTodoFile parses a markdown TODO file and returns structured TodoItem entries.
// It extracts categories from headings, items from list entries, and priorities from markers.
//
// Priority markers:
//   - !! or !!! = high
//   - ! = medium
//   - - or no marker = low
//
// Completed items (- [x]) are skipped.
func ParseTodoFile(path string) ([]TodoItem, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open TODO file: %w", err)
	}
	defer file.Close()

	var items []TodoItem
	var currentCategory string
	itemCounter := 1

	// Regular expressions for parsing
	headingRegex := regexp.MustCompile(`^#+\s+(.+)$`)
	completedRegex := regexp.MustCompile(`^\s*[-*]\s+\[[xX]\]`)
	checkboxItemRegex := regexp.MustCompile(`^\s*[-*]\s+\[\s\]\s+(.+)$`)
	plainListItemRegex := regexp.MustCompile(`^\s*[-*]\s+(.+)$`)
	priorityRegex := regexp.MustCompile(`^(!!+|!)\s+`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Check for heading (category)
		if matches := headingRegex.FindStringSubmatch(trimmed); matches != nil {
			currentCategory = strings.TrimSpace(matches[1])
			continue
		}

		// Check for completed items - skip them
		if completedRegex.MatchString(line) {
			continue
		}

		// Check for unchecked checkbox items first
		var content string
		if matches := checkboxItemRegex.FindStringSubmatch(line); matches != nil {
			content = matches[1]
		} else if matches := plainListItemRegex.FindStringSubmatch(line); matches != nil {
			// Then check for plain list items
			content = matches[1]
		}

		if content != "" {
			// Extract priority
			priority := "low"
			if priorityMatches := priorityRegex.FindStringSubmatch(content); priorityMatches != nil {
				marker := priorityMatches[1]
				if strings.HasPrefix(marker, "!!") {
					priority = "high"
				} else if marker == "!" {
					priority = "medium"
				}
				// Remove priority marker from content
				content = priorityRegex.ReplaceAllString(content, "")
			}

			item := TodoItem{
				ID:       fmt.Sprintf("TODO-%d", itemCounter),
				Category: currentCategory,
				Content:  strings.TrimSpace(content),
				Priority: priority,
			}
			items = append(items, item)
			itemCounter++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading TODO file: %w", err)
	}

	return items, nil
}
