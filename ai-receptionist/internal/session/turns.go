package session

import (
	"context"
	"fmt"
	"strings"

	"ai-receptionist/internal/store"
)

// GetLastTurns returns recent messages for a conversation in chronological order.
func GetLastTurns(ctx context.Context, db *store.DB, convID string, limit int) ([]store.Message, error) {
	if db == nil {
		return nil, fmt.Errorf("store db is nil")
	}
	_ = ctx
	return db.RecentMessages(convID, limit)
}

// FormatLastTurnsForPrompt formats the last maxTurns user/assistant messages for classifier prompts.
func FormatLastTurnsForPrompt(msgs []store.Message, maxTurns int) string {
	if maxTurns <= 0 || len(msgs) == 0 {
		return ""
	}
	var filtered []store.Message
	for _, m := range msgs {
		role := strings.TrimSpace(m.Role)
		if role != "user" && role != "assistant" {
			continue
		}
		if strings.TrimSpace(m.Message) == "" {
			continue
		}
		filtered = append(filtered, m)
	}
	if len(filtered) == 0 {
		return ""
	}
	start := 0
	if len(filtered) > maxTurns {
		start = len(filtered) - maxTurns
	}
	var b strings.Builder
	for _, m := range filtered[start:] {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%s: %s", strings.TrimSpace(m.Role), strings.TrimSpace(m.Message))
	}
	return b.String()
}
