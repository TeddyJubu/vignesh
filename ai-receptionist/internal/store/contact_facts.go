package store

import "time"

type ContactFact struct {
	ConvID    string
	Key       string
	Value     string
	UpdatedAt time.Time
}

func (d *DB) ListContactFacts(convID string) ([]ContactFact, error) {
	rows, err := d.db.Query(
		`SELECT conv_id, fact_key, fact_value, updated_at FROM contact_facts
		 WHERE conv_id = ?
		   AND (expires_at IS NULL OR expires_at = '' OR datetime(expires_at) > datetime('now'))
		 ORDER BY fact_key`,
		convID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ContactFact
	for rows.Next() {
		var f ContactFact
		var updated string
		if err := rows.Scan(&f.ConvID, &f.Key, &f.Value, &updated); err != nil {
			return nil, err
		}
		if t, err := parseSQLiteTime(updated); err == nil {
			f.UpdatedAt = t
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (d *DB) UpsertContactFact(convID, key, value string) error {
	_, err := d.db.Exec(
		`INSERT INTO contact_facts (conv_id, fact_key, fact_value, updated_at) VALUES (?, ?, ?, datetime('now'))
		 ON CONFLICT(conv_id, fact_key) DO UPDATE SET fact_value = excluded.fact_value, updated_at = datetime('now')`,
		convID, key, value,
	)
	return err
}
