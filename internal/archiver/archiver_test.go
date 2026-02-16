package archiver

import (
	"encoding/json"
	"testing"
	"time"
)

func TestArchiveEntryJSONRoundtrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	e := ArchiveEntry{
		TaskID:      "task-123",
		Title:       "Test Task",
		Status:      "completed",
		CompletedAt: &now,
		Wave:        2,
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var e2 ArchiveEntry
	if err := json.Unmarshal(b, &e2); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if e2.TaskID != e.TaskID || e2.Title != e.Title || e2.Status != e.Status || e2.Wave != e.Wave {
		t.Fatalf("mismatch: got %+v want %+v", e2, e)
	}
	if e2.CompletedAt == nil || !e2.CompletedAt.Equal(now) {
		t.Fatalf("completed_at mismatch: got %v want %v", e2.CompletedAt, now)
	}
}

func TestArchiveResultAndOptionsJSON(t *testing.T) {
	r := ArchiveResult{
		Total:      10,
		Completed:  7,
		ByWave:     map[string]int{"1": 3, "2": 4},
		OutputPath: "/tmp/out.md",
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal result failed: %v", err)
	}
	var r2 ArchiveResult
	if err := json.Unmarshal(b, &r2); err != nil {
		t.Fatalf("unmarshal result failed: %v", err)
	}
	if r2.Total != r.Total || r2.Completed != r.Completed || r2.OutputPath != r.OutputPath {
		t.Fatalf("mismatch: got %+v want %+v", r2, r)
	}
	o := ArchiveOptions{Wave: 2, Auto: true}
	bo, err := json.Marshal(o)
	if err != nil {
		t.Fatalf("marshal options failed: %v", err)
	}
	var o2 ArchiveOptions
	if err := json.Unmarshal(bo, &o2); err != nil {
		t.Fatalf("unmarshal options failed: %v", err)
	}
	if o2.Wave != o.Wave || o2.Auto != o.Auto {
		t.Fatalf("mismatch options: got %+v want %+v", o2, o)
	}
}
