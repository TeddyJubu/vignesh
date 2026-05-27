package store

import (
	"database/sql"
	"fmt"
	"strings"
)

const schemaVersionCurrent = 6

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
}

const placeholderRunbookCS = `# Julia CS runbook
- Answer from contact facts, business description, and memory only — never invent policies.
- In groups: keep replies short; address the sender; stay on support topics (GBP, local SEO, websites).
- Escalate billing disputes, refunds, or angry threads to the owner.
- Use escalate_to_vignesh when unsure or when the user asks for a human.`

const placeholderRunbookSales = `# Julia sales runbook
- Qualify one missing field per message: name, business_type, service_needed, budget, timeline, current_website.
- No unprompted pricing; defer firm quotes to Vignesh.
- When qualified, offer to book a short call and use check_calendar_availability before suggesting times.`

const placeholderRunbookBooking = `# Julia booking runbook
- Use check_calendar_availability for real slots before proposing times.
- After the user picks a slot, use book_appointment; only confirm booking when the tool returns booked:true.
- Collect email with collect_email when needed for calendar invite.
- If calendar is unavailable, collect best_time and hand off to Vignesh — do not invent slots.`

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

func refreshIdentityAgentNotes(db *sql.DB) error {
	notes := map[string]string{
		"identity_soul":        defaultIdentitySoul,
		"client_instructions": defaultClientInstructions,
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
		"identity_soul":        defaultIdentitySoul,
		"client_instructions": defaultClientInstructions,
		"julia-cs":             placeholderRunbookCS,
		"julia-sales":          placeholderRunbookSales,
		"julia-booking":        placeholderRunbookBooking,
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
