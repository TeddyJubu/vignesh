package store

import (
	"database/sql"
	"fmt"
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

func (d *DB) InsertAsyncJob(job AsyncJob) (string, error) {
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
	return job.ID, err
}

// ListPendingAsyncJobs atomically claims pending jobs by marking them running.
func (d *DB) ListPendingAsyncJobs(limit int) ([]AsyncJob, error) {
	if limit <= 0 {
		limit = 10
	}
	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.Query(
		`SELECT id, conv_id, job_type, status, payload, result, error, notify_owner, created_at, updated_at
		 FROM async_jobs WHERE status = 'pending' ORDER BY created_at ASC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	jobs, err := scanAsyncJobs(rows)
	rows.Close()
	if err != nil {
		return nil, err
	}
	var claimed []AsyncJob
	for _, j := range jobs {
		res, err := tx.Exec(
			`UPDATE async_jobs SET status = 'running', updated_at = datetime('now') WHERE id = ? AND status = 'pending'`,
			j.ID,
		)
		if err != nil {
			return nil, err
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			j.Status = "running"
			claimed = append(claimed, j)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return claimed, nil
}

func (d *DB) UpdateAsyncJobStatus(id, status, result, errMsg string) error {
	_, err := d.db.Exec(
		`UPDATE async_jobs SET status = ?, result = ?, error = ?, updated_at = datetime('now') WHERE id = ?`,
		status, result, errMsg, id,
	)
	return err
}

// GetAsyncJob returns a job by id (nil if missing).
func (d *DB) GetAsyncJob(id string) (*AsyncJob, error) {
	row := d.db.QueryRow(
		`SELECT id, conv_id, job_type, status, payload, result, error, notify_owner, created_at, updated_at
		 FROM async_jobs WHERE id = ?`, id,
	)
	var j AsyncJob
	var errMsg sql.NullString
	var created, updated string
	if err := row.Scan(&j.ID, &j.ConvID, &j.JobType, &j.Status, &j.Payload, &j.Result, &errMsg, &j.NotifyOwner, &created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if errMsg.Valid {
		j.Error = errMsg.String
	}
	if t, err := parseSQLiteTime(created); err == nil {
		j.CreatedAt = t
	}
	if t, err := parseSQLiteTime(updated); err == nil {
		j.UpdatedAt = t
	}
	return &j, nil
}

// ResetStaleRunningJobs requeues jobs stuck in running (e.g. after process crash).
func (d *DB) ResetStaleRunningJobs(maxAge time.Duration) (int, error) {
	if maxAge <= 0 {
		maxAge = 15 * time.Minute
	}
	cutoff := time.Now().Add(-maxAge).UTC().Format("2006-01-02 15:04:05")
	res, err := d.db.Exec(
		`UPDATE async_jobs SET status = 'pending', updated_at = datetime('now')
		 WHERE status = 'running' AND updated_at < ?`,
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("reset stale running jobs: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
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
