package receptionist

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode"

	"ai-receptionist/internal/agent"
)

const (
	maxWhatsAppReplyRunes = 420
	ownerOnlyCommandMsg   = "That command is only available to the bot owner."
)

var (
	modelDisclosure = regexp.MustCompile(`(?i)(large language model|language model|trained by google|trained by openai|gemini|gpt-|chatgpt|i am an ai|i'm an ai|your ai assistant|as an ai assistant|google's ai)`)
	internalLeak    = regexp.MustCompile(`(?i)(new qualified lead|recent tools:|recent messages:|julia escalation|conv:|conversation id|idempotency_key|tool outputs|notify_owner|paused_until)`)
	jsonBlob        = regexp.MustCompile(`(?s)\{[^{}]*"(?:booked|escalated|available|slots|email|timezone)"[^{}]*\}`)
	// Impossible 12-hour clock values only (e.g. Fri 37am), not valid two-digit hours like 10am/11am/12pm.
	invalidSlot = regexp.MustCompile(`(?i)\b(?:mon|tue|wed|thu|fri|sat|sun)\s+(?:1[3-9]|[2-9]\d|\d{3,})(?:am|pm)\b`)
	multiQuestion   = regexp.MustCompile(`\?`)
)

// FinalizeCustomerReply applies production safety and persona rules before sending to WhatsApp.
func FinalizeCustomerReply(reply, userText, businessName, businessDesc string, toolResults []agent.ToolResult) string {
	r := SanitizeReplyWithTools(reply, toolResults)
	r = stripInternalContent(r)
	r = enforcePersona(r, userText, businessName, businessDesc)
	r = fixServiceQuestionEcho(r, userText, businessName, businessDesc)
	r = enforceSingleQuestion(r)
	r = stripInvalidCalendarSlots(r)
	r = limitReplyLength(r)
	if strings.TrimSpace(r) == "" {
		return deferReply
	}
	return r
}

func stripInternalContent(reply string) string {
	r := strings.TrimSpace(reply)
	if internalLeak.MatchString(r) || strings.Contains(r, "🔥") {
		return deferReply
	}
	r = jsonBlob.ReplaceAllString(r, "")
	r = modelDisclosure.ReplaceAllString(r, "")
	// Drop lines that look like debug/tool dumps.
	lines := strings.Split(r, "\n")
	var kept []string
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}
		lower := strings.ToLower(l)
		if strings.HasPrefix(lower, "- [") ||
			strings.Contains(lower, "recent tools") ||
			strings.Contains(lower, "recent messages") ||
			strings.HasPrefix(lower, "conv:") ||
			strings.HasPrefix(lower, "conversation id") {
			continue
		}
		kept = append(kept, l)
	}
	r = strings.TrimSpace(strings.Join(kept, "\n"))
	return r
}

func enforcePersona(reply, userText, businessName, businessDesc string) string {
	r := strings.TrimSpace(reply)
	lowerUser := strings.ToLower(strings.TrimSpace(userText))
	lowerReply := strings.ToLower(r)

	if asksName(lowerUser) {
		name := strings.TrimSpace(businessName)
		if name == "" {
			name = "the business"
		}
		return "I'm Julia — " + name + "'s WhatsApp receptionist. I help with enquiries, services, and booking a call. What can I help you with?"
	}

	if asksModel(lowerUser) {
		return "I'm Julia, the receptionist here — Vignesh built and maintains me. I can't share technical details, but I'm happy to help with your enquiry."
	}

	if asksIdentity(lowerUser) && (strings.Contains(lowerReply, "ai assistant") ||
		strings.Contains(lowerReply, "don't have a personal name") ||
		strings.Contains(lowerReply, "no personal name") ||
		modelDisclosure.MatchString(lowerReply)) {
		return "I'm Julia, the WhatsApp receptionist for " + strings.TrimSpace(businessName) + ". I help with questions, lead details, and scheduling — what would you like to know?"
	}

	if modelDisclosure.MatchString(r) {
		return "I'm Julia, the receptionist here — Vignesh built and maintains me. How can I help you today?"
	}

	return r
}

func asksName(user string) bool {
	return strings.Contains(user, "your name") ||
		strings.Contains(user, "what's your name") ||
		strings.Contains(user, "whats your name") ||
		strings.Contains(user, "who are you")
}

func asksModel(user string) bool {
	return strings.Contains(user, "what model") ||
		strings.Contains(user, "which model") ||
		strings.Contains(user, "what llm") ||
		strings.Contains(user, "are you chatgpt") ||
		strings.Contains(user, "are you gemini") ||
		strings.Contains(user, "how were you built")
}

func asksIdentity(user string) bool {
	return asksName(user) || strings.Contains(user, "who are you") || strings.Contains(user, "what are you")
}

func fixServiceQuestionEcho(reply, userText, businessName, businessDesc string) string {
	u := normalizeForCompare(userText)
	r := normalizeForCompare(reply)
	if u == "" {
		return reply
	}
	if r != u && !isEchoQuestion(reply, userText) {
		return reply
	}
	if !looksLikeServicesQuestion(userText) {
		return reply
	}
	return defaultServicesReply(businessName, businessDesc)
}

func looksLikeServicesQuestion(text string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	return strings.Contains(t, "what services") ||
		strings.Contains(t, "services do you offer") ||
		strings.Contains(t, "what do you offer") ||
		strings.Contains(t, "what can you do")
}

func isEchoQuestion(reply, userText string) bool {
	r := strings.TrimSpace(reply)
	u := strings.TrimSpace(userText)
	if r == "" || u == "" {
		return false
	}
	if strings.EqualFold(r, u) {
		return true
	}
	// Planner echoed the user's question back.
	if strings.HasSuffix(strings.TrimSpace(r), "?") && strings.Contains(strings.ToLower(r), strings.ToLower(strings.Trim(u, "? "))) {
		return true
	}
	return false
}

func defaultServicesReply(businessName, businessDesc string) string {
	name := strings.TrimSpace(businessName)
	if name == "" {
		name = "us"
	}
	desc := strings.TrimSpace(businessDesc)
	if desc == "" {
		desc = "websites, landing pages, and lead qualification before a strategy call."
	}
	// Keep first sentence of business description if it's short enough.
	first := desc
	if idx := strings.IndexAny(desc, ".\n"); idx > 0 && idx < 160 {
		first = strings.TrimSpace(desc[:idx+1])
	}
	if len(first) > 200 {
		first = first[:200] + "…"
	}
	return "I'm Julia for " + name + ". We help with " + first + " What are you looking to build?"
}

func enforceSingleQuestion(reply string) string {
	r := strings.TrimSpace(reply)
	if r == "" {
		return r
	}
	idxs := questionIndices(r)
	if len(idxs) <= 1 {
		return r
	}
	// Keep everything up through the first question mark (inclusive).
	cut := idxs[0] + 1
	out := strings.TrimSpace(r[:cut])
	return limitReplyLength(out)
}

func questionIndices(s string) []int {
	var idxs []int
	for i, r := range s {
		if r == '?' {
			idxs = append(idxs, i)
		}
	}
	return idxs
}

func stripInvalidCalendarSlots(reply string) string {
	if !invalidSlot.MatchString(reply) {
		return reply
	}
	// If we detect impossible times, avoid quoting specific slots.
	return "Let me check availability with the team and come back with proper times. What day and timezone works best for you?"
}

func limitReplyLength(reply string) string {
	r := strings.TrimSpace(reply)
	if r == "" {
		return r
	}
	runes := []rune(r)
	if len(runes) <= maxWhatsAppReplyRunes {
		return r
	}
	// Trim at sentence boundary when possible.
	truncated := string(runes[:maxWhatsAppReplyRunes])
	if idx := strings.LastIndexAny(truncated, ".!?"); idx > 80 {
		return strings.TrimSpace(truncated[:idx+1])
	}
	return strings.TrimSpace(truncated) + "…"
}

func normalizeForCompare(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// CustomerSafeToolOutput returns a redacted summary for collation when tools return internal payloads.
func CustomerSafeToolOutput(tool string, output string) string {
	tool = strings.ToLower(strings.TrimSpace(tool))
	var m map[string]any
	if json.Unmarshal([]byte(output), &m) != nil {
		return output
	}
	switch tool {
	case "escalate_to_vignesh":
		return `{"escalated":true,"customer_note":"Owner notified; human will follow up."}`
	case "check_calendar_availability":
		slots, _ := m["slots"].([]any)
		var clean []string
		for _, s := range slots {
			if str, ok := s.(string); ok && !invalidSlot.MatchString(str) {
				clean = append(clean, str)
			}
		}
		m["slots"] = clean
		delete(m, "query")
		b, _ := json.Marshal(m)
		return string(b)
	case "collect_email":
		if email, _ := m["email"].(string); email == "missing" {
			return `{"email":"missing","note":"Ask for preferred day/time/timezone before email."}`
		}
		b, _ := json.Marshal(m)
		return string(b)
	default:
		delete(m, "idempotency_key")
		delete(m, "conv_id")
		b, _ := json.Marshal(m)
		return string(b)
	}
}
