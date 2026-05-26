package store

import (
	"encoding/json"
	"time"
)

type AgentState struct {
	Phone     string
	StateJSON string
	UpdatedAt time.Time
	CreatedAt time.Time
}

func (d *DB) GetAgentState(phone string) (*AgentState, error) {
	row := d.db.QueryRow(`SELECT phone, state_json, updated_at, created_at FROM agent_states WHERE phone = ?`, phone)
	var s AgentState
	var updated, created string
	if err := row.Scan(&s.Phone, &s.StateJSON, &updated, &created); err != nil {
		return nil, err
	}
	if t, err := parseSQLiteTime(updated); err == nil {
		s.UpdatedAt = t
	}
	if t, err := parseSQLiteTime(created); err == nil {
		s.CreatedAt = t
	}
	return &s, nil
}

func (d *DB) UpsertAgentState(phone string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(
		`INSERT INTO agent_states (phone, state_json, updated_at) VALUES (?, ?, datetime('now'))
		 ON CONFLICT(phone) DO UPDATE SET state_json = excluded.state_json, updated_at = datetime('now')`,
		phone, string(b),
	)
	return err
}

func (d *DB) ClearAgentState(phone string) error {
	_, err := d.db.Exec(`DELETE FROM agent_states WHERE phone = ?`, phone)
	return err
}

