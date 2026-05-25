package receptionist

import "strings"

// IsPauseKeyword detects owner/human takeover commands (case-insensitive).
func IsPauseKeyword(text string) bool {
	t := strings.TrimSpace(strings.ToLower(text))
	switch t {
	case "pause", "human", "stop bot":
		return true
	}
	return false
}
