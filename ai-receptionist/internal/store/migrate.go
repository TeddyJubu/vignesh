package store

import (
	"database/sql"
	"fmt"
	"strings"
)

var contactMigrations = []string{
	`ALTER TABLE contacts ADD COLUMN paused_until TEXT`,
	`ALTER TABLE contacts ADD COLUMN language TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE contacts ADD COLUMN lead_score TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE contacts ADD COLUMN last_bot_reply_at TEXT`,
	`ALTER TABLE contacts ADD COLUMN status_before_pause TEXT`,
	`ALTER TABLE contacts ADD COLUMN webhook_sent_at TEXT`,
}

func migrate(db *sql.DB) error {
	for _, stmt := range contactMigrations {
		if _, err := db.Exec(stmt); err != nil {
			if !isDuplicateColumn(err) {
				return fmt.Errorf("migrate: %w", err)
			}
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
