package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ai-receptionist/internal/agent"
)

const (
	defaultAgentStateTTL   = 60 * time.Minute
	defaultMessagesPerConv = 200
	defaultCleanupInterval = 15 * time.Minute
)

// CleanupConfig tunes periodic retention.
type CleanupConfig struct {
	Interval         time.Duration
	AgentStateTTL    time.Duration
	MessagesPerConv  int
	PurgeExpiredFacts bool
}

func DefaultCleanupConfig() CleanupConfig {
	return CleanupConfig{
		Interval:          defaultCleanupInterval,
		AgentStateTTL:     defaultAgentStateTTL,
		MessagesPerConv:   defaultMessagesPerConv,
		PurgeExpiredFacts: true,
	}
}

// RunCleanupLoop periodically purges stale agent state, caps messages, and expired facts.
func (d *DB) RunCleanupLoop(ctx context.Context, cfg CleanupConfig) {
	if cfg.Interval <= 0 {
		cfg.Interval = defaultCleanupInterval
	}
	if cfg.AgentStateTTL <= 0 {
		cfg.AgentStateTTL = defaultAgentStateTTL
	}
	if cfg.MessagesPerConv <= 0 {
		cfg.MessagesPerConv = defaultMessagesPerConv
	}
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := d.RunCleanupOnce(cfg); err != nil {
				log.Printf("cleanup: %v", err)
			}
		}
	}
}

func (d *DB) RunCleanupOnce(cfg CleanupConfig) error {
	if n, err := d.PurgeStaleAgentStates(cfg.AgentStateTTL); err != nil {
		return err
	} else if n > 0 {
		log.Printf("cleanup: removed %d stale agent_states", n)
	}
	if n, err := d.CapMessagesPerContact(cfg.MessagesPerConv); err != nil {
		return err
	} else if n > 0 {
		log.Printf("cleanup: trimmed %d old messages", n)
	}
	if cfg.PurgeExpiredFacts {
		if n, err := d.PurgeExpiredContactFacts(time.Now()); err != nil {
			return err
		} else if n > 0 {
			log.Printf("cleanup: removed %d expired contact_facts", n)
		}
	}
	return nil
}

// PurgeStaleAgentStates deletes planner state older than ttl based on StartedAtUNIX in state JSON.
func (d *DB) PurgeStaleAgentStates(ttl time.Duration) (int64, error) {
	cutoff := time.Now().Add(-ttl).Unix()
	rows, err := d.db.Query(`SELECT phone, state_json FROM agent_states`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var stale []string
	for rows.Next() {
		var phone, raw string
		if err := rows.Scan(&phone, &raw); err != nil {
			return 0, err
		}
		var st agent.State
		if json.Unmarshal([]byte(raw), &st) != nil {
			continue
		}
		started := st.StartedAtUNIX
		if started == 0 {
			continue
		}
		if started < cutoff {
			stale = append(stale, phone)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	var removed int64
	for _, phone := range stale {
		if err := d.ClearAgentState(phone); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

// CapMessagesPerContact keeps only the newest limit rows per phone.
func (d *DB) CapMessagesPerContact(limit int) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM messages WHERE id IN (
			SELECT id FROM (
				SELECT id,
				       ROW_NUMBER() OVER (PARTITION BY phone ORDER BY datetime(created_at) DESC) AS rn
				FROM messages
			) ranked WHERE rn > ?
		)`,
		limit,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// PurgeExpiredContactFacts removes facts past expires_at.
func (d *DB) PurgeExpiredContactFacts(now time.Time) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM contact_facts WHERE expires_at IS NOT NULL AND expires_at != '' AND datetime(expires_at) <= datetime(?)`,
		now.UTC().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (d *DB) SetSchemaVersion(version int) error {
	_, err := d.db.Exec(
		`INSERT INTO schema_version (version, applied_at) VALUES (?, datetime('now'))
		 ON CONFLICT(version) DO UPDATE SET applied_at = datetime('now')`,
		version,
	)
	return err
}

func (d *DB) CurrentSchemaVersion() (int, error) {
	var v int
	err := d.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&v)
	if err != nil {
		return 0, fmt.Errorf("schema_version: %w", err)
	}
	return v, nil
}
