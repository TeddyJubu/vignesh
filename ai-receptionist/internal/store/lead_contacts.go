package store

import (
	"time"

	"github.com/google/uuid"
)

type LeadContact struct {
	ID         string
	JobID      string
	Name       string
	Company    string
	Email      string
	Phone      string
	FitScore   int
	PitchAngle string
	URL        string
	LinkedIn   string
	ICPMatch   string
	CreatedAt  time.Time
}

func (d *DB) InsertLeadContacts(jobID string, rows []LeadContact) error {
	for _, r := range rows {
		if r.ID == "" {
			r.ID = uuid.NewString()
		}
		_, err := d.db.Exec(
			`INSERT INTO lead_contacts (id, job_id, name, company, email, phone, fit_score, pitch_angle, url, linkedin, icp_match, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
			r.ID, jobID, r.Name, r.Company, r.Email, r.Phone, r.FitScore, r.PitchAngle, r.URL, r.LinkedIn, r.ICPMatch,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) CountLeadContactsByJob(jobID string) (int, error) {
	var n int
	err := d.db.QueryRow(`SELECT COUNT(1) FROM lead_contacts WHERE job_id = ?`, jobID).Scan(&n)
	return n, err
}
