package archiver

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/storage"
)

// ArchiveEntry represents a single archived task entry.
type ArchiveEntry struct {
	TaskID      string     `json:"task_id"`
	Title       string     `json:"title"`
	Status      string     `json:"status"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// ArchiveResult holds summary statistics and output path for an archive operation.
type ArchiveResult struct {
	Total      int    `json:"total"`
	OutputPath string `json:"output_path,omitempty"`
}

// Archiver handles archival operations for completed tasks
type Archiver struct {
	cfg     *config.Config
	storage *storage.Storage
}

// NewArchiver creates a new Archiver instance
func NewArchiver(cfg *config.Config, store *storage.Storage) *Archiver {
	return &Archiver{
		cfg:     cfg,
		storage: store,
	}
}

// ExportCompleted exports all completed tasks to .devloop/archive/archive-<timestamp>.jsonl
// and updates their status to 'archived' in the main storage.
func (a *Archiver) ExportCompleted() (*ArchiveResult, error) {
	tasks, err := a.storage.QueryTasks(storage.Filter{Status: "completed"})
	if err != nil {
		return nil, fmt.Errorf("failed to query completed tasks: %w", err)
	}

	if len(tasks) == 0 {
		return &ArchiveResult{Total: 0}, nil
	}

	archiveDir := filepath.Join(a.cfg.Project.Path, ".devloop", "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create archive directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	archivePath := filepath.Join(archiveDir, fmt.Sprintf("archive-%s.jsonl", timestamp))
	file, err := os.OpenFile(archivePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create archive file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)

	for _, task := range tasks {
		entry := ArchiveEntry{
			TaskID: task.ID,
			Title:  task.Title,
			Status: task.Status,
		}
		if task.Results != nil {
			entry.CompletedAt = &task.Results.CompletedAt
		}
		if err := encoder.Encode(entry); err != nil {
			return nil, fmt.Errorf("failed to encode archive entry for task %s: %w", task.ID, err)
		}
	}

	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush archive file: %w", err)
	}

	for _, task := range tasks {
		task.Status = "archived"
		task.Metadata.UpdatedAt = time.Now()
		if err := a.storage.UpdateTask(task); err != nil {
			return nil, fmt.Errorf("failed to update task %s to archived status: %w", task.ID, err)
		}
	}

	return &ArchiveResult{
		Total:      len(tasks),
		OutputPath: archivePath,
	}, nil
}

// GenerateMarkdownSummary creates a human-readable markdown summary for archived tasks.
func (a *Archiver) GenerateMarkdownSummary(tasks []*storage.Task) (string, error) {
	if len(tasks) == 0 {
		return "", nil
	}

	var buf strings.Builder

	buf.WriteString("# Archive Summary\n\n")
	buf.WriteString(fmt.Sprintf("**Archived:** %s\n\n", time.Now().Format("2006-01-02 15:04:05 MST")))

	buf.WriteString("## Tasks\n\n")
	for _, task := range tasks {
		buf.WriteString(fmt.Sprintf("- [x] **%s**: %s\n", task.ID, task.Title))
	}

	totalDuration := 0
	for _, task := range tasks {
		totalDuration += task.Execution.TotalDuration
	}

	buf.WriteString("\n## Statistics\n\n")
	buf.WriteString(fmt.Sprintf("- **Total Tasks:** %d\n", len(tasks)))
	buf.WriteString(fmt.Sprintf("- **Total Duration:** %d seconds\n", totalDuration))

	return buf.String(), nil
}

// WriteMarkdownSummary writes the markdown summary to a timestamped file in .devloop/archive/.
func (a *Archiver) WriteMarkdownSummary(markdown string) (string, error) {
	archiveDir := filepath.Join(a.cfg.Project.Path, ".devloop", "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create archive directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	mdPath := filepath.Join(archiveDir, fmt.Sprintf("archive-%s.md", timestamp))
	if err := os.WriteFile(mdPath, []byte(markdown), 0644); err != nil {
		return "", fmt.Errorf("failed to write markdown summary: %w", err)
	}

	return mdPath, nil
}
