package store

import (
	"database/sql"
	"strings"
	"time"
)

type DreamProposal struct {
	ID        string
	CreatedAt time.Time
	Status    string
	Title     string
	Patch     string
	Rationale string
}

func (d *DB) ListDreamProposals(limit int) ([]DreamProposal, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := d.db.Query(
		`SELECT id, created_at, status, title, patch, rationale
		   FROM dream_proposals
		  ORDER BY datetime(created_at) DESC
		  LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DreamProposal
	for rows.Next() {
		var p DreamProposal
		var created string
		if err := rows.Scan(&p.ID, &created, &p.Status, &p.Title, &p.Patch, &p.Rationale); err != nil {
			return nil, err
		}
		if t, err := parseSQLiteTime(created); err == nil {
			p.CreatedAt = t
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (d *DB) GetDreamProposal(id string) (*DreamProposal, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, nil
	}
	row := d.db.QueryRow(
		`SELECT id, created_at, status, title, patch, rationale
		   FROM dream_proposals
		  WHERE id = ?`,
		id,
	)
	var p DreamProposal
	var created string
	if err := row.Scan(&p.ID, &created, &p.Status, &p.Title, &p.Patch, &p.Rationale); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if t, err := parseSQLiteTime(created); err == nil {
		p.CreatedAt = t
	}
	return &p, nil
}

func (d *DB) InsertDreamProposal(p DreamProposal) error {
	if strings.TrimSpace(p.ID) == "" {
		return nil
	}
	if strings.TrimSpace(p.Status) == "" {
		p.Status = "pending"
	}
	_, err := d.db.Exec(
		`INSERT INTO dream_proposals (id, created_at, status, title, patch, rationale)
		 VALUES (?, datetime('now'), ?, ?, ?, ?)
		 ON CONFLICT(id) DO NOTHING`,
		p.ID, p.Status, p.Title, p.Patch, p.Rationale,
	)
	return err
}

func (d *DB) UpdateDreamProposalStatus(id, status string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	status = strings.TrimSpace(status)
	if status == "" {
		return nil
	}
	_, err := d.db.Exec(`UPDATE dream_proposals SET status = ? WHERE id = ?`, status, id)
	return err
}
