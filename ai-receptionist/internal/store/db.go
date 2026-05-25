package store

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

type Contact struct {
	ID            int64
	Phone         string
	Name          string
	LeadData      map[string]string
	LeadDataRaw   string
	Status        string
	CreatedAt     time.Time
	LastMessageAt time.Time
}

type Message struct {
	ID        int64
	Phone     string
	Role      string
	Message   string
	CreatedAt time.Time
}

type DB struct {
	db *sql.DB
}

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) GetOrCreateContact(phone string) (*Contact, error) {
	c, err := d.GetContact(phone)
	if err == nil {
		return c, nil
	}
	_, err = d.db.Exec(
		`INSERT INTO contacts (phone, lead_data, status) VALUES (?, '{}', 'new')`,
		phone,
	)
	if err != nil {
		return nil, err
	}
	return d.GetContact(phone)
}

func (d *DB) GetContact(phone string) (*Contact, error) {
	row := d.db.QueryRow(
		`SELECT id, phone, name, lead_data, status, created_at, last_message_at FROM contacts WHERE phone = ?`,
		phone,
	)
	return scanContact(row)
}

func scanContact(row *sql.Row) (*Contact, error) {
	var c Contact
	var leadRaw string
	var created, last string
	if err := row.Scan(&c.ID, &c.Phone, &c.Name, &leadRaw, &c.Status, &created, &last); err != nil {
		return nil, err
	}
	c.LeadDataRaw = leadRaw
	c.LeadData = map[string]string{}
	if leadRaw != "" && leadRaw != "{}" {
		_ = json.Unmarshal([]byte(leadRaw), &c.LeadData)
	}
	if c.LeadData == nil {
		c.LeadData = map[string]string{}
	}
	var err error
	c.CreatedAt, err = parseSQLiteTime(created)
	if err != nil {
		return nil, err
	}
	c.LastMessageAt, err = parseSQLiteTime(last)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func parseSQLiteTime(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("parse time %q", s)
}

func (d *DB) UpdateContact(phone, name, leadJSON, status string) error {
	_, err := d.db.Exec(
		`UPDATE contacts SET name = ?, lead_data = ?, status = ?, last_message_at = datetime('now') WHERE phone = ?`,
		name, leadJSON, status, phone,
	)
	return err
}

func (d *DB) InsertMessage(phone, role, text string) error {
	_, err := d.db.Exec(
		`INSERT INTO messages (phone, role, message) VALUES (?, ?, ?)`,
		phone, role, text,
	)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`UPDATE contacts SET last_message_at = datetime('now') WHERE phone = ?`, phone)
	return err
}

func (d *DB) RecentMessages(phone string, limit int) ([]Message, error) {
	rows, err := d.db.Query(
		`SELECT id, phone, role, message, created_at FROM messages WHERE phone = ? ORDER BY created_at DESC LIMIT ?`,
		phone, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		var m Message
		var created string
		if err := rows.Scan(&m.ID, &m.Phone, &m.Role, &m.Message, &created); err != nil {
			return nil, err
		}
		m.CreatedAt, err = parseSQLiteTime(created)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	// reverse to chronological
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, rows.Err()
}
