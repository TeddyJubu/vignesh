CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS contacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    phone TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL DEFAULT '',
    lead_data TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'new',
    mode TEXT NOT NULL DEFAULT '',
    paused_until TEXT,
    language TEXT NOT NULL DEFAULT '',
    lead_score TEXT NOT NULL DEFAULT '',
    last_bot_reply_at TEXT,
    status_before_pause TEXT,
    webhook_sent_at TEXT,
    nudge_sent_at TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    last_message_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    phone TEXT NOT NULL,
    role TEXT NOT NULL,
    message TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_messages_phone_created ON messages(phone, created_at);
CREATE INDEX IF NOT EXISTS idx_messages_phone_role_created ON messages(phone, role, created_at);
CREATE INDEX IF NOT EXISTS idx_contacts_collecting_nudge ON contacts(status, nudge_sent_at, paused_until);

CREATE TABLE IF NOT EXISTS agent_states (
    phone TEXT PRIMARY KEY,
    state_json TEXT NOT NULL DEFAULT '{}',
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS agent_notes (
    key TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS contact_facts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conv_id TEXT NOT NULL,
    fact_key TEXT NOT NULL,
    fact_value TEXT NOT NULL,
    expires_at TEXT,
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(conv_id, fact_key)
);

CREATE INDEX IF NOT EXISTS idx_contact_facts_conv ON contact_facts(conv_id);

CREATE TABLE IF NOT EXISTS conv_meta (
    conv_id TEXT PRIMARY KEY,
    last_ack_at TEXT
);

CREATE TABLE IF NOT EXISTS turn_traces (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conv_id TEXT NOT NULL,
    phase TEXT NOT NULL,
    latency_ms INTEGER NOT NULL DEFAULT 0,
    error TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_turn_traces_conv ON turn_traces(conv_id, created_at);

CREATE TABLE IF NOT EXISTS tool_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conv_id TEXT NOT NULL,
    tool TEXT NOT NULL,
    input TEXT NOT NULL DEFAULT '',
    output TEXT NOT NULL DEFAULT '',
    error TEXT,
    latency_ms INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_tool_runs_conv ON tool_runs(conv_id, created_at);

CREATE TABLE IF NOT EXISTS app_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS dream_proposals (
    id TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    status TEXT NOT NULL DEFAULT 'pending',
    title TEXT NOT NULL DEFAULT '',
    patch TEXT NOT NULL DEFAULT '',
    rationale TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_dream_proposals_created ON dream_proposals(created_at);

CREATE TABLE IF NOT EXISTS async_jobs (
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
);

CREATE INDEX IF NOT EXISTS idx_async_jobs_status ON async_jobs(status, created_at);

CREATE TABLE IF NOT EXISTS booking_requests (
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
);

CREATE INDEX IF NOT EXISTS idx_booking_requests_guest ON booking_requests(guest_phone, status);

-- RBAC + dashboard auth
CREATE TABLE IF NOT EXISTS access_roles (
    phone TEXT PRIMARY KEY,
    role TEXT NOT NULL,
    permissions_json TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_access_roles_role ON access_roles(role, phone);

CREATE TABLE IF NOT EXISTS dashboard_otp_codes (
    phone TEXT NOT NULL,
    code_hash TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_dashboard_otp_phone ON dashboard_otp_codes(phone);
CREATE INDEX IF NOT EXISTS idx_dashboard_otp_expires ON dashboard_otp_codes(expires_at);

CREATE TABLE IF NOT EXISTS dashboard_sessions (
    token_hash TEXT PRIMARY KEY,
    phone TEXT NOT NULL,
    role TEXT NOT NULL,
    permissions_json TEXT NOT NULL DEFAULT '{}',
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_dashboard_sessions_phone ON dashboard_sessions(phone);
CREATE INDEX IF NOT EXISTS idx_dashboard_sessions_expires ON dashboard_sessions(expires_at);
