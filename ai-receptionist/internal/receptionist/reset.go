package receptionist

import "strings"

// IsResetKeyword detects owner-only reset commands (case-insensitive).
// This clears per-conversation state (planner session, facts, recent messages) so the bot starts fresh.
func IsResetKeyword(text string) bool {
	t := strings.TrimSpace(strings.ToLower(text))
	switch t {
	case "reset", "reset session", "refresh", "refresh session", "restart", "restart session":
		return true
	}
	return false
}

