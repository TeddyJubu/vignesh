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
