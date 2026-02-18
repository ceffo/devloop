package processor

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/storage"
)

var (
	taskHeaderRe   = regexp.MustCompile(`^###\s+(\S+):\s+(.+)$`)
	complexityRe   = regexp.MustCompile(`\*\*Complexity:\*\*\s+` + "`" + `([^` + "`" + `]+)` + "`")
	backtickWordRe = regexp.MustCompile("`([^`]+)`")
)

// ParseTasksMarkdown reads a TASKS.md file written in devloop's markdown format
// and converts each task section into a *storage.Task ready for saving.
//
// Expected section format (separated by "---"):
//
//	### ID: Title
//
//	**Model:** `…` | **Complexity:** `simple|moderate|complex`
//
//	Description text…
//
//	**Acceptance Criteria:**
//	- …
//
//	**Blocked by:** `ID1`, `ID2` (or — / none)
//	**Tags:** `tag1`, `tag2`
func ParseTasksMarkdown(filePath string, cfg *config.Config) ([]*storage.Task, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks markdown file: %w", err)
	}

	// Split on horizontal rules — each task lives between two "---" lines.
	sections := strings.Split(string(data), "\n---\n")

	var tasks []*storage.Task
	for _, section := range sections {
		task := parseTaskSection(strings.TrimSpace(section), cfg)
		if task != nil {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// parseTaskSection parses a single markdown task block.
// Returns nil when the section does not contain a valid task header.
func parseTaskSection(section string, cfg *config.Config) *storage.Task {
	lines := strings.Split(section, "\n")

	// Find the ### header line.
	var headerIdx int = -1
	var taskID, title string
	for i, line := range lines {
		if m := taskHeaderRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			headerIdx = i
			taskID = m[1]
			title = strings.TrimSpace(m[2])
			break
		}
	}
	if headerIdx < 0 {
		return nil
	}

	// Everything after the header.
	body := lines[headerIdx+1:]

	complexity := parseComplexity(body)
	description := parseDescription(body)
	criteria := parseAcceptanceCriteria(body)
	blockedBy := parseBlockedBy(body)
	tags := parseTags(body)

	now := time.Now()
	task := &storage.Task{
		ID:                 taskID,
		Title:              title,
		Status:             "pending",
		Complexity:         complexity,
		Description:        description,
		AcceptanceCriteria: criteria,
		BlockedBy:          blockedBy,
		Tags:               tags,
		Metadata: storage.TaskMetadata{
			CreatedAt:   now,
			UpdatedAt:   now,
			SourceType:  "prd",
			MaxAttempts: cfg.Execution.MaxAttempts,
		},
		Execution: storage.TaskExecution{},
	}

	return task
}

// parseComplexity extracts the value from "**Complexity:** `value`".
func parseComplexity(lines []string) string {
	for _, line := range lines {
		if m := complexityRe.FindStringSubmatch(line); m != nil {
			return m[1]
		}
	}
	return "moderate" // sensible default
}

// parseDescription extracts the plain-text description: the block of text
// between the model/complexity line and the first "**Acceptance Criteria:**" line.
func parseDescription(lines []string) string {
	// Skip the Model/Complexity line.
	start := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "**Model:**") {
			start = i + 1
			break
		}
	}
	if start < 0 {
		start = 0
	}

	var descParts []string
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "**Acceptance Criteria:**") ||
			strings.HasPrefix(trimmed, "**Blocked by:**") ||
			strings.HasPrefix(trimmed, "**Tags:**") {
			break
		}
		descParts = append(descParts, trimmed)
	}

	// Collapse multiple blank lines and trim.
	desc := strings.Join(descParts, "\n")
	// Normalise repeated newlines.
	for strings.Contains(desc, "\n\n\n") {
		desc = strings.ReplaceAll(desc, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(desc)
}

// parseAcceptanceCriteria returns the bullet-point lines after
// "**Acceptance Criteria:**" until the next bold field.
func parseAcceptanceCriteria(lines []string) []string {
	var criteria []string
	collecting := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "**Acceptance Criteria:**") {
			collecting = true
			continue
		}
		if collecting {
			if strings.HasPrefix(trimmed, "**") {
				break
			}
			if strings.HasPrefix(trimmed, "- ") {
				criteria = append(criteria, strings.TrimPrefix(trimmed, "- "))
			}
		}
	}
	return criteria
}

// parseBlockedBy extracts task IDs from "**Blocked by:** `ID1`, `ID2`".
// Returns nil (no dependencies) when the value is "—" or empty.
func parseBlockedBy(lines []string) []string {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "**Blocked by:**") {
			return extractBacktickWords(trimmed)
		}
	}
	return nil
}

// parseTags extracts tag names from "**Tags:** `tag1`, `tag2`".
func parseTags(lines []string) []string {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "**Tags:**") {
			return extractBacktickWords(trimmed)
		}
	}
	return nil
}

// extractBacktickWords returns all backtick-quoted words in a line.
func extractBacktickWords(line string) []string {
	matches := backtickWordRe.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return nil
	}
	result := make([]string, 0, len(matches))
	for _, m := range matches {
		result = append(result, m[1])
	}
	return result
}
