package store

import (
	"database/sql"
	"strings"
	"time"
)

type AppSetting struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}

func (d *DB) ListAppSettings() ([]AppSetting, error) {
	rows, err := d.db.Query(`SELECT key, value, updated_at FROM app_settings ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AppSetting
	for rows.Next() {
		var s AppSetting
		var updated string
		if err := rows.Scan(&s.Key, &s.Value, &updated); err != nil {
			return nil, err
		}
		if t, err := parseSQLiteTime(updated); err == nil {
			s.UpdatedAt = t
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (d *DB) GetAppSetting(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", nil
	}
	row := d.db.QueryRow(`SELECT value FROM app_settings WHERE key = ?`, key)
	var v string
	if err := row.Scan(&v); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return v, nil
}

func (d *DB) UpsertAppSetting(key, value string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	_, err := d.db.Exec(
		`INSERT INTO app_settings (key, value, updated_at) VALUES (?, ?, datetime('now'))
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = datetime('now')`,
		key, value,
	)
	return err
}
