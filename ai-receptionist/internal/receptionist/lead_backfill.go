package receptionist

import (
	"regexp"
	"strings"

	"ai-receptionist/internal/store"
)

var emailRegex = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
var nameRegex1 = regexp.MustCompile(`(?i)(?:my name is|i am|i'm)\s+([A-Za-z]+)`)

// backfillLeadFromHistory fills empty name/email in lead data from recent user messages.
func backfillLeadFromHistory(history []store.Message, data map[string]string) map[string]string {
	out := make(map[string]string, len(data)+2)
	for k, v := range data {
		if strings.TrimSpace(v) != "" {
			out[k] = v
		}
	}

	nameSet := strings.TrimSpace(out["name"]) != ""
	emailSet := strings.TrimSpace(out["email"]) != ""
	if nameSet && emailSet {
		return out
	}

	for _, msg := range history {
		if msg.Role != "user" {
			continue
		}
		text := msg.Message

		if !emailSet {
			if match := emailRegex.FindString(text); match != "" {
				out["email"] = match
				emailSet = true
			}
		}

		if !nameSet {
			if m := nameRegex1.FindStringSubmatch(text); len(m) > 1 {
				out["name"] = m[1]
				nameSet = true
			} else if idx := strings.Index(text, ","); idx > 0 {
				first := strings.TrimSpace(text[:idx])
				if first != "" && !strings.Contains(first, " ") && len(first) < 20 {
					out["name"] = first
					nameSet = true
				}
			}
		}

		if nameSet && emailSet {
			break
		}
	}

	return out
}
