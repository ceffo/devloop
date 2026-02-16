package archiver

import "time"

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
	Total      int               `json:"total"`
	Completed  int               `json:"completed"`
	ByWave     map[string]int    `json:"by_wave,omitempty"`
	OutputPath string            `json:"output_path,omitempty"`
}

// ArchiveOptions provide filters and flags for archive generation.
type ArchiveOptions struct {
	Wave int  `json:"wave,omitempty"`
	Auto bool `json:"auto"`
}
