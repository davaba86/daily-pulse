package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/davaba86/daily-pulse/internal/model"
)

const schema = `
CREATE TABLE IF NOT EXISTS snapshots (
	id        INTEGER PRIMARY KEY AUTOINCREMENT,
	taken_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS pull_requests (
	snapshot_id INTEGER NOT NULL REFERENCES snapshots(id),
	source      TEXT NOT NULL,
	repo        TEXT NOT NULL,
	number      INTEGER NOT NULL,
	title       TEXT NOT NULL,
	author      TEXT NOT NULL,
	updated_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS pipeline_runs (
	snapshot_id  INTEGER NOT NULL REFERENCES snapshots(id),
	repo         TEXT NOT NULL,
	status       TEXT NOT NULL,
	branch       TEXT NOT NULL,
	duration_sec INTEGER NOT NULL,
	updated_at   TEXT NOT NULL
);
`

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}
	return db, nil
}

func SaveSnapshot(db *sql.DB, prs []model.PullRequest, pipelines []model.PipelineRun, takenAt time.Time) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec("INSERT INTO snapshots (taken_at) VALUES (?)", takenAt.Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("insert snapshot: %w", err)
	}
	snapshotID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read snapshot id: %w", err)
	}

	for _, pr := range prs {
		_, err := tx.Exec(
			"INSERT INTO pull_requests (snapshot_id, source, repo, number, title, author, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			snapshotID, pr.Source, pr.Repo, pr.Number, pr.Title, pr.Author, pr.UpdatedAt.Format(time.RFC3339),
		)
		if err != nil {
			return 0, fmt.Errorf("insert pull_request %s#%d: %w", pr.Repo, pr.Number, err)
		}
	}

	for _, p := range pipelines {
		_, err := tx.Exec(
			"INSERT INTO pipeline_runs (snapshot_id, repo, status, branch, duration_sec, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			snapshotID, p.Repo, string(p.Status), p.Branch, int(p.Duration.Seconds()), p.UpdatedAt.Format(time.RFC3339),
		)
		if err != nil {
			return 0, fmt.Errorf("insert pipeline_run %s: %w", p.Repo, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}
	return snapshotID, nil
}
