package store

import (
	"database/sql"
	"fmt"
	"strings"
)

const schemaVersionCurrent = 11

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
	`CREATE TABLE IF NOT EXISTS app_settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL DEFAULT '',
		updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
	)`,
	`CREATE TABLE IF NOT EXISTS dream_proposals (
		id TEXT PRIMARY KEY,
		created_at DATETIME NOT NULL DEFAULT (datetime('now')),
		status TEXT NOT NULL DEFAULT 'pending',
		title TEXT NOT NULL DEFAULT '',
		patch TEXT NOT NULL DEFAULT '',
		rationale TEXT NOT NULL DEFAULT ''
	)`,
	`CREATE INDEX IF NOT EXISTS idx_dream_proposals_created ON dream_proposals(created_at)`,
	`CREATE TABLE IF NOT EXISTS async_jobs (
		id TEXT PRIMARY KEY,
		conv_id TEXT NOT NULL DEFAULT '',
		job_type TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		payload TEXT NOT NULL DEFAULT '{}',
		result TEXT NOT NULL DEFAULT '',
		error TEXT,
		notify_owner INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT (datetime('now')),
		updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_async_jobs_status ON async_jobs(status, created_at)`,
	`CREATE TABLE IF NOT EXISTS booking_requests (
		id TEXT PRIMARY KEY,
		owner_conv TEXT NOT NULL DEFAULT '',
		guest_phone TEXT NOT NULL DEFAULT '',
		guest_name TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'pending',
		guest_slots_json TEXT NOT NULL DEFAULT '[]',
		proposed_slot TEXT,
		event_id TEXT,
		created_at DATETIME NOT NULL DEFAULT (datetime('now')),
		updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_booking_requests_guest ON booking_requests(guest_phone, status)`,
	`CREATE INDEX IF NOT EXISTS idx_messages_phone_role_created ON messages(phone, role, created_at)`,
	`CREATE INDEX IF NOT EXISTS idx_contacts_collecting_nudge ON contacts(status, nudge_sent_at, paused_until)`,
	`CREATE TABLE IF NOT EXISTS access_roles (
		phone TEXT PRIMARY KEY,
		role TEXT NOT NULL,
		permissions_json TEXT NOT NULL DEFAULT '{}',
		created_at DATETIME NOT NULL DEFAULT (datetime('now')),
		updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_access_roles_role ON access_roles(role, phone)`,
	`CREATE TABLE IF NOT EXISTS dashboard_otp_codes (
		phone TEXT NOT NULL,
		code_hash TEXT NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL DEFAULT (datetime('now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_dashboard_otp_phone ON dashboard_otp_codes(phone)`,
	`CREATE INDEX IF NOT EXISTS idx_dashboard_otp_expires ON dashboard_otp_codes(expires_at)`,
	`CREATE TABLE IF NOT EXISTS dashboard_sessions (
		token_hash TEXT PRIMARY KEY,
		phone TEXT NOT NULL,
		role TEXT NOT NULL,
		permissions_json TEXT NOT NULL DEFAULT '{}',
		expires_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL DEFAULT (datetime('now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_dashboard_sessions_phone ON dashboard_sessions(phone)`,
	`CREATE INDEX IF NOT EXISTS idx_dashboard_sessions_expires ON dashboard_sessions(expires_at)`,
}


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
	schemaBefore, err := schemaVersion(db)
	if err != nil {
		return err
	}
	if err := seedAgentNotes(db); err != nil {
		return err
	}
	for _, stmt := range infraMigrations {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate infra: %w", err)
		}
	}
	if err := seedAccessDefaults(db); err != nil {
		return err
	}
	for _, stmt := range contactMigrations {
		if _, err := db.Exec(stmt); err != nil {
			if !isDuplicateColumn(err) {
				return fmt.Errorf("migrate: %w", err)
			}
		}
	}
	if schemaBefore < 6 {
		if err := refreshIdentityAgentNotes(db); err != nil {
			return err
		}
	}
	if schemaBefore < 7 {
		if err := refreshIdentitySoul(db); err != nil {
			return err
		}
	}
	if schemaBefore < 8 {
		if err := refreshClientInstructions(db); err != nil {
			return err
		}
	}
	if schemaBefore < 9 {
		// Ensure the initial allowlist settings exist (safe no-op on conflict).
		if err := seedAccessDefaults(db); err != nil {
			return err
		}
	}
	if schemaBefore < 10 {
		if err := refreshClientInstructions(db); err != nil {
			return err
		}
	}
	if schemaBefore < 11 {
		if err := refreshAgentRunbooks(db); err != nil {
			return err
		}
		if err := refreshClientInstructions(db); err != nil {
			return err
		}
	}
	if schemaBefore < schemaVersionCurrent {
		if _, err := db.Exec(
			`INSERT INTO schema_version (version, applied_at) VALUES (?, datetime('now'))
			 ON CONFLICT(version) DO NOTHING`,
			schemaVersionCurrent,
		); err != nil {
			return fmt.Errorf("schema_version: %w", err)
		}
	}
	return nil
}

func seedAccessDefaults(db *sql.DB) error {
	// Default WhatsApp inbound gating: allow-all on, empty allow-list.
	if _, err := db.Exec(
		`INSERT INTO app_settings (key, value, updated_at) VALUES (?, ?, datetime('now'))
		 ON CONFLICT(key) DO NOTHING`,
		"access.allow_all", "1",
	); err != nil {
		return fmt.Errorf("seed access.allow_all: %w", err)
	}
	if _, err := db.Exec(
		`INSERT INTO app_settings (key, value, updated_at) VALUES (?, ?, datetime('now'))
		 ON CONFLICT(key) DO NOTHING`,
		"access.allow_list", "[]",
	); err != nil {
		return fmt.Errorf("seed access.allow_list: %w", err)
	}

	// Seed default admin if no admins exist.
	var admins int
	if err := db.QueryRow(`SELECT COUNT(1) FROM access_roles WHERE role = 'admin'`).Scan(&admins); err != nil {
		// Table may not exist in legacy DBs; migrations should have created it already.
		return fmt.Errorf("seed access_roles admin count: %w", err)
	}
	if admins == 0 {
		if _, err := db.Exec(
			`INSERT INTO access_roles (phone, role, permissions_json, created_at, updated_at)
			 VALUES (?, 'admin', '{}', datetime('now'), datetime('now'))
			 ON CONFLICT(phone) DO UPDATE SET role='admin', updated_at=datetime('now')`,
			"6590013157",
		); err != nil {
			return fmt.Errorf("seed default admin: %w", err)
		}
	}
	return nil
}

func schemaVersion(db *sql.DB) (int, error) {
	var v int
	err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&v)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return 0, nil
		}
		return 0, fmt.Errorf("schema_version: %w", err)
	}
	return v, nil
}

func refreshClientInstructions(db *sql.DB) error {
	_, err := db.Exec(
		`INSERT INTO agent_notes (key, content, updated_at) VALUES (?, ?, datetime('now'))
		 ON CONFLICT(key) DO UPDATE SET content = excluded.content, updated_at = datetime('now')`,
		"client_instructions", defaultClientInstructions(),
	)
	if err != nil {
		return fmt.Errorf("refresh agent_note client_instructions: %w", err)
	}
	return nil
}

func refreshAgentRunbooks(db *sql.DB) error {
	runbooks := map[string]string{
		"julia-cs":      RunbookCS,
		"julia-sales":   RunbookSales,
		"julia-booking": RunbookBooking,
	}
	for key, content := range runbooks {
		if _, err := db.Exec(
			`INSERT INTO agent_notes (key, content, updated_at) VALUES (?, ?, datetime('now'))
			 ON CONFLICT(key) DO UPDATE SET content = excluded.content, updated_at = datetime('now')`,
			key, content,
		); err != nil {
			return fmt.Errorf("refresh agent_note %s: %w", key, err)
		}
	}
	return nil
}

func refreshIdentitySoul(db *sql.DB) error {
	_, err := db.Exec(
		`INSERT INTO agent_notes (key, content, updated_at) VALUES (?, ?, datetime('now'))
		 ON CONFLICT(key) DO UPDATE SET content = excluded.content, updated_at = datetime('now')`,
		"identity_soul", defaultIdentitySoul(),
	)
	if err != nil {
		return fmt.Errorf("refresh agent_note identity_soul: %w", err)
	}
	return nil
}

func refreshIdentityAgentNotes(db *sql.DB) error {
	notes := map[string]string{
		"identity_soul":        defaultIdentitySoul(),
		"client_instructions": defaultClientInstructions(),
	}
	for key, content := range notes {
		if _, err := db.Exec(
			`INSERT INTO agent_notes (key, content, updated_at) VALUES (?, ?, datetime('now'))
			 ON CONFLICT(key) DO UPDATE SET content = excluded.content, updated_at = datetime('now')`,
			key, content,
		); err != nil {
			return fmt.Errorf("refresh agent_note %s: %w", key, err)
		}
	}
	return nil
}

func seedAgentNotes(db *sql.DB) error {
	seeds := map[string]string{
		"identity_soul":        defaultIdentitySoul(),
		"client_instructions": defaultClientInstructions(),
		"julia-cs":             RunbookCS,
		"julia-sales":          RunbookSales,
		"julia-booking":        RunbookBooking,
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
