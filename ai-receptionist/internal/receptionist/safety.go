package receptionist

import (
	"encoding/json"
	"regexp"
	"strings"

	"ai-receptionist/internal/agent"
)

const deferReply = "Let me pass this to the team so they can confirm properly."

var (
	// Block firm sales promises, not factual pricing or refund policy from the knowledge base.
	priceGuarantee = regexp.MustCompile(`(?i)(?:\bfixed price\b|\bwill cost exactly\b|\bi guarantee you(?:'ll| will)\b|\bwe guarantee you(?:'ll| will)\b|\bguaranteed to rank\b|\bguaranteed (?:#1|number one)\b|\bprice (?:is )?guaranteed\b)`)
	bookingConfirm = regexp.MustCompile(`(?i)(booked|booking(?:\s+is)?\s+confirmed|appointment (?:is )?confirmed|scheduled for (?:monday|tuesday|wednesday|thursday|friday|saturday|sunday|\d)|see you (?:on|at) \d)`)
)

func SanitizeReply(reply string) string {
	r := strings.TrimSpace(reply)
	if r == "" {
		return deferReply
	}
	if priceGuarantee.MatchString(r) || bookingConfirm.MatchString(r) {
		return deferReply
	}
	return r
}

// SanitizeReplyWithTools enforces the same safety rules as SanitizeReply, but allows
// booking confirmations only when the booking tool returned booked=true.
func SanitizeReplyWithTools(reply string, toolResults []agent.ToolResult) string {
	r := strings.TrimSpace(reply)
	if r == "" {
		return deferReply
	}
	// Pricing guarantees are always disallowed.
	if priceGuarantee.MatchString(r) {
		return deferReply
	}
	// Booking confirmations are allowed only if book_appointment succeeded.
	if bookingConfirm.MatchString(r) {
		for _, tr := range toolResults {
			if strings.ToLower(strings.TrimSpace(tr.Tool)) != "book_appointment" || strings.TrimSpace(tr.Error) != "" {
				continue
			}
			var m map[string]any
			if json.Unmarshal([]byte(tr.Output), &m) == nil {
				if booked, _ := m["booked"].(bool); booked {
					return r
				}
			}
		}
		return deferReply
	}
	return r
}
