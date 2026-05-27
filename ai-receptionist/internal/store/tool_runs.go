package store

import "time"

// InsertToolRun records a tool invocation audit row.
func (d *DB) InsertToolRun(convID, tool, input, output, errMsg string, latencyMS int64) error {
	_, err := d.db.Exec(
		`INSERT INTO tool_runs (conv_id, tool, input, output, error, latency_ms, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, datetime('now'))`,
		convID, tool, input, output, errMsg, latencyMS,
	)
	return err
}

type ToolRun struct {
	ID        int64
	ConvID    string
	Tool      string
	Input     string
	Output    string
	Error     string
	LatencyMS int64
	CreatedAt time.Time
}

// RecentToolRuns returns latest tool runs for a conversation.
func (d *DB) RecentToolRuns(convID string, limit int) ([]ToolRun, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := d.db.Query(
		`SELECT id, conv_id, tool, input, COALESCE(output,''), COALESCE(error,''), latency_ms, created_at
		 FROM tool_runs WHERE conv_id = ? ORDER BY id DESC LIMIT ?`,
		convID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ToolRun
	for rows.Next() {
		var r ToolRun
		var created string
		if err := rows.Scan(&r.ID, &r.ConvID, &r.Tool, &r.Input, &r.Output, &r.Error, &r.LatencyMS, &created); err != nil {
			return nil, err
		}
		if t, err := parseSQLiteTime(created); err == nil {
			r.CreatedAt = t
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
