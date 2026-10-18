package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/davaba86/daily-pulse/internal/model"
)

func TestSaveSnapshot_AndReadBack(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "pulse.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	takenAt := time.Date(2026, 10, 18, 8, 2, 0, 0, time.UTC)
	prs := []model.PullRequest{
		{Source: "github", Repo: "davaba86/mc-platform-cloud", Number: 482, Title: "fix-auth token refresh", UpdatedAt: takenAt},
	}
	pipelines := []model.PipelineRun{
		{Repo: "davaba86/mc-platform-cloud", Status: model.PipelinePass, Branch: "main", Duration: 192 * time.Second, UpdatedAt: takenAt},
	}

	snapshotID, err := SaveSnapshot(db, prs, pipelines, takenAt)
	if err != nil {
		t.Fatalf("SaveSnapshot() error = %v", err)
	}
	if snapshotID == 0 {
		t.Fatal("SaveSnapshot() returned snapshot id 0, want nonzero")
	}

	var prCount, pipelineCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM pull_requests WHERE snapshot_id = ?", snapshotID).Scan(&prCount); err != nil {
		t.Fatalf("count pull_requests: %v", err)
	}
	if prCount != 1 {
		t.Errorf("pull_requests count = %d, want 1", prCount)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM pipeline_runs WHERE snapshot_id = ?", snapshotID).Scan(&pipelineCount); err != nil {
		t.Fatalf("count pipeline_runs: %v", err)
	}
	if pipelineCount != 1 {
		t.Errorf("pipeline_runs count = %d, want 1", pipelineCount)
	}
}
