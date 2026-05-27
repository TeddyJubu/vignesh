package store

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

type Contact struct {
	ID                int64
	Phone             string
	Name              string
	LeadData          map[string]string
	LeadDataRaw       string
	Status            string
	Mode              string
	StatusBeforePause string
	PausedUntil       *time.Time
	Language          string
	LeadScore         string
	LastBotReplyAt    *time.Time
	WebhookSentAt     *time.Time
	CreatedAt         time.Time
	LastMessageAt     time.Time
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
	dsn := path
	if path == ":memory:" {
		dsn = "file:ai_receptionist_test?mode=memory&cache=shared"
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
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

const contactSelect = `SELECT id, phone, name, lead_data, status,
	COALESCE(mode, ''), COALESCE(status_before_pause, ''), COALESCE(paused_until, ''), COALESCE(language, ''), COALESCE(lead_score, ''),
	COALESCE(last_bot_reply_at, ''), COALESCE(webhook_sent_at, ''), created_at, last_message_at FROM contacts WHERE phone = ?`

func (d *DB) GetContact(phone string) (*Contact, error) {
	row := d.db.QueryRow(contactSelect, phone)
	return scanContact(row)
}

func scanContact(row *sql.Row) (*Contact, error) {
	var c Contact
	var leadRaw, pausedRaw, lastBotRaw, webhookRaw, created, last string
	if err := row.Scan(&c.ID, &c.Phone, &c.Name, &leadRaw, &c.Status, &c.Mode,
		&c.StatusBeforePause, &pausedRaw, &c.Language, &c.LeadScore, &lastBotRaw, &webhookRaw, &created, &last); err != nil {
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
	if pausedRaw != "" {
		if t, err := parseSQLiteTime(pausedRaw); err == nil {
			c.PausedUntil = &t
		}
	}
	if lastBotRaw != "" {
		if t, err := parseSQLiteTime(lastBotRaw); err == nil {
			c.LastBotReplyAt = &t
		}
	}
	if webhookRaw != "" {
		if t, err := parseSQLiteTime(webhookRaw); err == nil {
			c.WebhookSentAt = &t
		}
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

// IsPaused returns true when human takeover is active for this contact.
func (c *Contact) IsPaused(now time.Time) bool {
	if c.Status != "paused" {
		return false
	}
	if c.PausedUntil == nil {
		return true
	}
	return now.Before(*c.PausedUntil)
}

func (d *DB) PauseContact(phone string, until time.Time) error {
	c, err := d.GetContact(phone)
	if err != nil {
		return err
	}
	before := c.Status
	if before == "paused" {
		before = c.StatusBeforePause
	}
	if before == "" {
		before = "collecting"
	}
	_, err = d.db.Exec(
		`UPDATE contacts SET status = 'paused', status_before_pause = ?, paused_until = ?, last_message_at = datetime('now') WHERE phone = ?`,
		before, until.UTC().Format(time.RFC3339), phone,
	)
	return err
}

func (d *DB) ClearPauseIfExpired(phone string, now time.Time) error {
	c, err := d.GetContact(phone)
	if err != nil {
		return err
	}
	if c.Status != "paused" {
		return nil
	}
	if c.PausedUntil != nil && now.After(*c.PausedUntil) {
		restore := strings.TrimSpace(c.StatusBeforePause)
		if restore == "" || restore == "paused" {
			restore = "collecting"
		}
		_, err = d.db.Exec(
			`UPDATE contacts SET status = ?, paused_until = NULL, status_before_pause = NULL WHERE phone = ? AND status = 'paused'`,
			restore, phone,
		)
		return err
	}
	return nil
}

func (d *DB) UpdateContact(phone, name, leadJSON, status string) error {
	return d.UpdateContactWithScore(phone, name, leadJSON, status, "")
}

func (d *DB) UpdateContactWithScore(phone, name, leadJSON, status, leadScore string) error {
	if leadScore != "" {
		_, err := d.db.Exec(
			`UPDATE contacts SET name = ?, lead_data = ?, status = ?, lead_score = ?, last_message_at = datetime('now') WHERE phone = ?`,
			name, leadJSON, status, leadScore, phone,
		)
		return err
	}
	_, err := d.db.Exec(
		`UPDATE contacts SET name = ?, lead_data = ?, status = ?, last_message_at = datetime('now') WHERE phone = ?`,
		name, leadJSON, status, phone,
	)
	return err
}

func (d *DB) TouchLastBotReply(phone string) error {
	_, err := d.db.Exec(
		`UPDATE contacts SET last_bot_reply_at = datetime('now') WHERE phone = ?`,
		phone,
	)
	return err
}

func (d *DB) MarkWebhookSent(phone string) error {
	_, err := d.db.Exec(
		`UPDATE contacts SET webhook_sent_at = datetime('now') WHERE phone = ?`,
		phone,
	)
	return err
}

func (d *DB) SetContactLanguage(phone, lang string) error {
	_, err := d.db.Exec(
		`UPDATE contacts SET language = ? WHERE phone = ?`,
		lang, phone,
	)
	return err
}

func (d *DB) SetContactMode(phone, mode string) error {
	_, err := d.db.Exec(
		`UPDATE contacts SET mode = ? WHERE phone = ?`,
		mode, phone,
	)
	return err
}

func (d *DB) ListStaleCollecting(cutoff time.Time) ([]string, error) {
	cutoffStr := cutoff.UTC().Format("2006-01-02 15:04:05")
	rows, err := d.db.Query(
		`SELECT c.phone FROM contacts c
		 WHERE c.status = 'collecting'
		   AND (c.nudge_sent_at IS NULL OR c.nudge_sent_at = '')
		   AND (c.paused_until IS NULL OR datetime(c.paused_until) <= datetime('now'))
		   AND EXISTS (
		     SELECT 1 FROM messages m WHERE m.phone = c.phone AND m.role = 'user'
		   )
		   AND (
		     SELECT MAX(datetime(m.created_at)) FROM messages m
		     WHERE m.phone = c.phone AND m.role = 'user'
		   ) <= datetime(?)`,
		cutoffStr,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var phone string
		if err := rows.Scan(&phone); err != nil {
			return nil, err
		}
		out = append(out, phone)
	}
	return out, rows.Err()
}

func (d *DB) MarkNudgeSent(phone string) error {
	_, err := d.db.Exec(
		`UPDATE contacts SET nudge_sent_at = datetime('now') WHERE phone = ?`,
		phone,
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
	if role == "user" {
		_, _ = d.db.Exec(
			`UPDATE contacts SET nudge_sent_at = NULL, last_message_at = datetime('now') WHERE phone = ?`,
			phone,
		)
		return nil
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
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, rows.Err()
}
