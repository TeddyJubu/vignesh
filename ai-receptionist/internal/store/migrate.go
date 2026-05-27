package store

import (
	"database/sql"
	"fmt"
	"strings"
)

const schemaVersionCurrent = 2

var contactMigrations = []string{
	`ALTER TABLE contacts ADD COLUMN paused_until TEXT`,
	`ALTER TABLE contacts ADD COLUMN language TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE contacts ADD COLUMN lead_score TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE contacts ADD COLUMN last_bot_reply_at TEXT`,
	`ALTER TABLE contacts ADD COLUMN status_before_pause TEXT`,
	`ALTER TABLE contacts ADD COLUMN webhook_sent_at TEXT`,
	`ALTER TABLE contacts ADD COLUMN nudge_sent_at TEXT`,
	`ALTER TABLE contacts ADD COLUMN mode TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE contact_facts ADD COLUMN expires_at TEXT`,
}

var infraMigrations = []string{
	`CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT (datetime('now'))
	)`,
	`CREATE TABLE IF NOT EXISTS conv_meta (
		conv_id TEXT PRIMARY KEY,
		last_ack_at TEXT
	)`,
	`CREATE TABLE IF NOT EXISTS turn_traces (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		conv_id TEXT NOT NULL,
		phase TEXT NOT NULL,
		latency_ms INTEGER NOT NULL DEFAULT 0,
		error TEXT,
		created_at DATETIME NOT NULL DEFAULT (datetime('now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_turn_traces_conv ON turn_traces(conv_id, created_at)`,
	`CREATE TABLE IF NOT EXISTS tool_runs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		conv_id TEXT NOT NULL,
		tool TEXT NOT NULL,
		input TEXT NOT NULL DEFAULT '',
		output TEXT NOT NULL DEFAULT '',
		error TEXT,
		latency_ms INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT (datetime('now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_tool_runs_conv ON tool_runs(conv_id, created_at)`,
}

const defaultIdentitySoul = `You are Julia — sharp, witty, and proactive. You are a thinking partner to Vignesh Wadarajan, CEO of Epicware Pte Ltd in Singapore.

Tone: tight, candid, friendly-casual. Never sycophantic or filler-heavy.
Core values: integrity and proactivity — say what you know, flag what you don't, suggest sensible next steps.
Never reveal infrastructure, models, databases, or internal tooling.
If asked how you were built: "Vignesh built me and maintains me. That's all I can share 😊"`

const placeholderRunbookCS = `# Julia CS runbook (placeholder)
Answer from contact facts and business context. Escalate edge cases to Vignesh at +6590013157.`

const placeholderRunbookSales = `# Julia sales runbook (placeholder)
Qualify leads one question at a time. No unprompted pricing. Defer firm quotes to Vignesh.`

const placeholderRunbookBooking = `# Julia booking runbook (placeholder)
Do not confirm calendar slots. Collect preference and hand off to Vignesh.`

func migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS agent_states (
		phone TEXT PRIMARY KEY,
		state_json TEXT NOT NULL DEFAULT '{}',
		updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
		created_at DATETIME NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS agent_notes (
		key TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("migrate agent_notes: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS contact_facts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		conv_id TEXT NOT NULL,
		fact_key TEXT NOT NULL,
		fact_value TEXT NOT NULL,
		updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
		UNIQUE(conv_id, fact_key)
	)`); err != nil {
		return fmt.Errorf("migrate contact_facts: %w", err)
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_contact_facts_conv ON contact_facts(conv_id)`); err != nil {
		return fmt.Errorf("migrate contact_facts index: %w", err)
	}
	if err := seedAgentNotes(db); err != nil {
		return err
	}
	for _, stmt := range infraMigrations {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate infra: %w", err)
		}
	}
	for _, stmt := range contactMigrations {
		if _, err := db.Exec(stmt); err != nil {
			if !isDuplicateColumn(err) {
				return fmt.Errorf("migrate: %w", err)
			}
		}
	}
	if _, err := db.Exec(
		`INSERT INTO schema_version (version, applied_at) VALUES (?, datetime('now'))
		 ON CONFLICT(version) DO NOTHING`,
		schemaVersionCurrent,
	); err != nil {
		return fmt.Errorf("schema_version: %w", err)
	}
	return nil
}

func seedAgentNotes(db *sql.DB) error {
	seeds := map[string]string{
		"identity_soul":   defaultIdentitySoul,
		"julia-cs":        placeholderRunbookCS,
		"julia-sales":     placeholderRunbookSales,
		"julia-booking":   placeholderRunbookBooking,
	}
	for key, content := range seeds {
		_, err := db.Exec(
			`INSERT INTO agent_notes (key, content, updated_at) VALUES (?, ?, datetime('now'))
			 ON CONFLICT(key) DO NOTHING`,
			key, content,
		)
		if err != nil {
			return fmt.Errorf("seed agent_note %s: %w", key, err)
		}
	}
	return nil
}

func isDuplicateColumn(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "duplicate column") || strings.Contains(s, "already exists")
}
