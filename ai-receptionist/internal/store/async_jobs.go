package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type AsyncJob struct {
	ID          string
	ConvID      string
	JobType     string
	Status      string
	Payload     string
	Result      string
	Error       string
	NotifyOwner bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (d *DB) InsertAsyncJob(job AsyncJob) error {
	if job.ID == "" {
		job.ID = uuid.NewString()
	}
	if job.Status == "" {
		job.Status = "pending"
	}
	notify := 0
	if job.NotifyOwner {
		notify = 1
	}
	_, err := d.db.Exec(
		`INSERT INTO async_jobs (id, conv_id, job_type, status, payload, result, error, notify_owner, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		job.ID, job.ConvID, job.JobType, job.Status, job.Payload, job.Result, job.Error, notify,
	)
	return err
}

func (d *DB) ListPendingAsyncJobs(limit int) ([]AsyncJob, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := d.db.Query(
		`SELECT id, conv_id, job_type, status, payload, result, error, notify_owner, created_at, updated_at
		 FROM async_jobs WHERE status = 'pending' ORDER BY created_at ASC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAsyncJobs(rows)
}

func (d *DB) UpdateAsyncJobStatus(id, status, result, errMsg string) error {
	_, err := d.db.Exec(
		`UPDATE async_jobs SET status = ?, result = ?, error = ?, updated_at = datetime('now') WHERE id = ?`,
		status, result, errMsg, id,
	)
	return err
}

func scanAsyncJobs(rows *sql.Rows) ([]AsyncJob, error) {
	var out []AsyncJob
	for rows.Next() {
		var j AsyncJob
		var created, updated string
		var notify int
		if err := rows.Scan(&j.ID, &j.ConvID, &j.JobType, &j.Status, &j.Payload, &j.Result, &j.Error, &notify, &created, &updated); err != nil {
			return nil, err
		}
		j.NotifyOwner = notify != 0
		if t, err := parseSQLiteTime(created); err == nil {
			j.CreatedAt = t
		}
		if t, err := parseSQLiteTime(updated); err == nil {
			j.UpdatedAt = t
		}
		out = append(out, j)
	}
	return out, rows.Err()
}
