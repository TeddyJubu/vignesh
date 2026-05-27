package store

import (
	"database/sql"
	"time"
)

// GetLastAckAt returns persisted ack cooldown timestamp for a conversation.
func (d *DB) GetLastAckAt(convID string) (*time.Time, error) {
	var raw string
	err := d.db.QueryRow(`SELECT COALESCE(last_ack_at, '') FROM conv_meta WHERE conv_id = ?`, convID).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}
	t, err := parseSQLiteTime(raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TouchLastAckAt persists ack send time (survives restarts).
func (d *DB) TouchLastAckAt(convID string, at time.Time) error {
	_, err := d.db.Exec(
		`INSERT INTO conv_meta (conv_id, last_ack_at) VALUES (?, ?)
		 ON CONFLICT(conv_id) DO UPDATE SET last_ack_at = excluded.last_ack_at`,
		convID, at.UTC().Format(time.RFC3339),
	)
	return err
}

func (d *DB) ClearConvMeta(convID string) error {
	_, err := d.db.Exec(`DELETE FROM conv_meta WHERE conv_id = ?`, convID)
	return err
}
