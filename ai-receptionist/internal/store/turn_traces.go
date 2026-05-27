package store

import "time"

// InsertTurnTrace records a phase timing for post-mortems.
func (d *DB) InsertTurnTrace(convID, phase string, latencyMS int64, errMsg string) error {
	_, err := d.db.Exec(
		`INSERT INTO turn_traces (conv_id, phase, latency_ms, error, created_at) VALUES (?, ?, ?, ?, datetime('now'))`,
		convID, phase, latencyMS, errMsg,
	)
	return err
}

// RecentTurnTraces returns the latest traces for a conversation.
func (d *DB) RecentTurnTraces(convID string, limit int) ([]TurnTrace, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := d.db.Query(
		`SELECT id, conv_id, phase, latency_ms, COALESCE(error, ''), created_at FROM turn_traces
		 WHERE conv_id = ? ORDER BY id DESC LIMIT ?`,
		convID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TurnTrace
	for rows.Next() {
		var t TurnTrace
		var created string
		if err := rows.Scan(&t.ID, &t.ConvID, &t.Phase, &t.LatencyMS, &t.Error, &created); err != nil {
			return nil, err
		}
		if ts, err := parseSQLiteTime(created); err == nil {
			t.CreatedAt = ts
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

type TurnTrace struct {
	ID        int64
	ConvID    string
	Phase     string
	LatencyMS int64
	Error     string
	CreatedAt time.Time
}
