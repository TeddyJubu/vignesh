package store

import (
	"database/sql"
	"time"
)

type AgentNote struct {
	Key       string
	Content   string
	UpdatedAt time.Time
}

func (d *DB) ListAgentNotes() ([]AgentNote, error) {
	rows, err := d.db.Query(`SELECT key, content, updated_at FROM agent_notes ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AgentNote
	for rows.Next() {
		var n AgentNote
		var updated string
		if err := rows.Scan(&n.Key, &n.Content, &updated); err != nil {
			return nil, err
		}
		if t, err := parseSQLiteTime(updated); err == nil {
			n.UpdatedAt = t
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (d *DB) GetAgentNote(key string) (string, error) {
	row := d.db.QueryRow(`SELECT content FROM agent_notes WHERE key = ?`, key)
	var content string
	if err := row.Scan(&content); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return content, nil
}

func (d *DB) UpsertAgentNote(key, content string) error {
	_, err := d.db.Exec(
		`INSERT INTO agent_notes (key, content, updated_at) VALUES (?, ?, datetime('now'))
		 ON CONFLICT(key) DO UPDATE SET content = excluded.content, updated_at = datetime('now')`,
		key, content,
	)
	return err
}
