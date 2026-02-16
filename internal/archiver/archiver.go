package archiver

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ceffo/devloop/internal/config"
	"github.com/ceffo/devloop/internal/storage"
)

// ArchiveEntry represents a single archived task entry.
// Fields are tagged for JSON serialization suitable for JSONL export.
type ArchiveEntry struct {
	TaskID      string     `json:"task_id"`
	Title       string     `json:"title"`
	Status      string     `json:"status"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Wave        int        `json:"wave,omitempty"`
}

// ArchiveResult holds summary statistics and output path for an archive operation.
type ArchiveResult struct {
	Total      int            `json:"total"`
	Completed  int            `json:"completed"`
	ByWave     map[string]int `json:"by_wave,omitempty"`
	OutputPath string         `json:"output_path,omitempty"`
}

// ArchiveOptions provide filters and flags for archive generation.
type ArchiveOptions struct {
	Wave int  `json:"wave,omitempty"`
	Auto bool `json:"auto"`
}

// ArchiveIndexEntry tracks metadata for a single archived wave
type ArchiveIndexEntry struct {
	Wave        int       `json:"wave"`
	ArchivedAt  time.Time `json:"archived_at"`
	TaskCount   int       `json:"task_count"`
	ArchivePath string    `json:"archive_path"`
}

// ArchiveIndex tracks all archived waves
type ArchiveIndex struct {
	Waves []ArchiveIndexEntry `json:"waves"`
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

// ExportWave exports completed tasks for a given wave to .devloop/archive/wave-N.jsonl
// and updates their status to 'archived' in the main storage
func (a *Archiver) ExportWave(wave int) (*ArchiveResult, error) {
	// Query completed tasks for the wave
	tasks, err := a.storage.QueryTasks(storage.Filter{
		Status: "completed",
		Wave:   wave,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query completed tasks for wave %d: %w", wave, err)
	}

	if len(tasks) == 0 {
		return &ArchiveResult{
			Total:     0,
			Completed: 0,
		}, nil
	}

	// Create archive directory if it doesn't exist
	archiveDir := filepath.Join(a.cfg.Project.Path, ".devloop", "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Write tasks to wave-N.jsonl
	archivePath := filepath.Join(archiveDir, fmt.Sprintf("wave-%d.jsonl", wave))
	file, err := os.OpenFile(archivePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create archive file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)

	// Write each task to the archive file
	for _, task := range tasks {
		entry := ArchiveEntry{
			TaskID: task.ID,
			Title:  task.Title,
			Status: task.Status,
			Wave:   task.Wave,
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

	// Update task statuses to 'archived' in main storage
	for _, task := range tasks {
		task.Status = "archived"
		task.Metadata.UpdatedAt = time.Now()
		if err := a.storage.UpdateTask(task); err != nil {
			return nil, fmt.Errorf("failed to update task %s to archived status: %w", task.ID, err)
		}
	}

	// Update archive index
	if err := a.updateArchiveIndex(wave, len(tasks), archivePath); err != nil {
		return nil, fmt.Errorf("failed to update archive index: %w", err)
	}

	return &ArchiveResult{
		Total:      len(tasks),
		Completed:  len(tasks),
		OutputPath: archivePath,
	}, nil
}

// updateArchiveIndex updates the archive index file with wave metadata
func (a *Archiver) updateArchiveIndex(wave, taskCount int, archivePath string) error {
	indexPath := filepath.Join(a.cfg.Project.Path, ".devloop", "archive", "index.json")

	// Load existing index or create new one
	var index ArchiveIndex
	if data, err := os.ReadFile(indexPath); err == nil {
		if err := json.Unmarshal(data, &index); err != nil {
			return fmt.Errorf("failed to parse existing archive index: %w", err)
		}
	}

	// Check if wave already exists in index and update it, or add new entry
	found := false
	for i, entry := range index.Waves {
		if entry.Wave == wave {
			index.Waves[i] = ArchiveIndexEntry{
				Wave:        wave,
				ArchivedAt:  time.Now(),
				TaskCount:   taskCount,
				ArchivePath: archivePath,
			}
			found = true
			break
		}
	}

	if !found {
		index.Waves = append(index.Waves, ArchiveIndexEntry{
			Wave:        wave,
			ArchivedAt:  time.Now(),
			TaskCount:   taskCount,
			ArchivePath: archivePath,
		})
	}

	// Write updated index
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal archive index: %w", err)
	}

	if err := os.WriteFile(indexPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write archive index: %w", err)
	}

	return nil
}
