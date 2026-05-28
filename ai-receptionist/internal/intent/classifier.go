package intent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ai-receptionist/internal/ai"
)

const classifyTimeout = 12 * time.Second

var allowedIntents = map[string]struct{}{
	"support":          {},
	"sales_qualify":    {},
	"calendar_check":   {},
	"group_manage":     {},
	"research_request": {},
	"lead_scrape":      {},
	"outbound_book":    {},
	"image_generate":   {},
	"general":          {},
}

// Result is the structured output of intent classification.
type Result struct {
	Intent     string  `json:"intent"`
	Confidence float64 `json:"confidence"`
	Summary    string  `json:"summary"`
}

const classifySystemPrompt = `You classify the user's latest WhatsApp message into exactly ONE intent.
Respond with ONE JSON object only (no markdown): {"intent":"...","confidence":0.0,"summary":"..."}

Allowed intent values (use exactly one):
support | sales_qualify | calendar_check | group_manage | research_request | lead_scrape | outbound_book | image_generate | general

Intent definitions (pick the best match):
- support: pricing, plans, how it works, refunds, complaints, general product/service questions
- sales_qualify: wants to buy, book a sales/discovery call, become a lead, "interested in your services"
- calendar_check: user's own schedule/agenda ("what am I doing tomorrow", availability check for them)
- group_manage: WhatsApp group admin, members, group settings
- research_request: market/competitor/ad/industry research (not scraping contact lists)
- lead_scrape: scrape/export lists of businesses or contacts (e.g. "20 dental clinics")
- outbound_book: bot should coordinate booking on behalf of owner with a third party (outbound scheduling)
- image_generate: create/edit an image or creative asset
- general: small talk or unclear

Disambiguation rules:
- "pricing", "plans", "how much" → support (NOT sales_qualify)
- "book a meeting/call with you" or "I want to book" → sales_qualify (NOT outbound_book unless coordinating with someone else)
- "what am I doing tomorrow" / "my calendar" → calendar_check

confidence is 0.0–1.0. summary is at most ten words describing what the user wants.`

// Classify runs intent classification using the given provider and conversation context.
func Classify(ctx context.Context, p ai.Provider, message, lastTurnsText string) (Result, error) {
	if p == nil {
		return Result{}, fmt.Errorf("intent classifier: nil provider")
	}
	cctx, cancel := context.WithTimeout(ctx, classifyTimeout)
	defer cancel()

	user := "MESSAGE:\n" + strings.TrimSpace(message)
	if t := strings.TrimSpace(lastTurnsText); t != "" {
		user += "\n\nLAST_5_TURNS:\n" + t
	}
	msgs := []ai.ChatMessage{
		{Role: "system", Content: classifySystemPrompt},
		{Role: "user", Content: user},
	}
	// jsonMode must be false: providers map jsonMode to receptionist reply/lead_updates schema.
	raw, err := p.Complete(cctx, msgs, false)
	if err != nil {
		return Result{}, err
	}
	result, err := decodeResult(raw)
	if err == nil {
		return applyIntentHints(message, normalizeResult(result)), nil
	}
	repaired, err2 := p.Complete(cctx, ai.RepairIntentPrompt(raw), false)
	if err2 != nil {
		return Result{}, err2
	}
	result, err = decodeResult(repaired)
	if err != nil {
		return Result{}, err
	}
	return applyIntentHints(message, normalizeResult(result)), nil
}

func decodeResult(raw string) (Result, error) {
	raw = ai.StripCodeFences(raw)
	var r Result
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return Result{}, fmt.Errorf("parse intent JSON: %w", err)
	}
	if strings.TrimSpace(r.Intent) == "" {
		return Result{}, fmt.Errorf("intent JSON missing intent")
	}
	return r, nil
}

// applyIntentHints corrects common model confusions using cheap phrase rules (after normalizeResult).
func applyIntentHints(message string, r Result) Result {
	lower := strings.ToLower(strings.TrimSpace(message))
	switch {
	case containsAny(lower, "pricing", "price", "plans", "how much", "cost", "package"):
		if r.Intent == "sales_qualify" || r.Intent == "outbound_book" {
			r.Intent = "support"
		}
	case containsAny(lower, "book a meeting", "book a call", "schedule a call", "want to book", "book meeting"):
		if r.Intent == "outbound_book" || r.Intent == "calendar_check" {
			r.Intent = "sales_qualify"
		}
	case containsAny(lower, "what am i doing tomorrow", "my calendar", "my schedule", "tomorrow"):
		if r.Intent != "lead_scrape" && r.Intent != "research_request" {
			if containsAny(lower, "tomorrow", "calendar", "schedule") && !containsAny(lower, "book a", "book meeting", "scrape", "research") {
				r.Intent = "calendar_check"
			}
		}
	case containsAny(lower, "research", "meta ad", "trends"):
		if r.Intent == "general" || r.Intent == "support" {
			r.Intent = "research_request"
		}
	case containsAny(lower, "scrape", "clinics", "dental"):
		if containsAny(lower, "scrape") {
			r.Intent = "lead_scrape"
		}
	}
	return r
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func normalizeResult(r Result) Result {
	intent := strings.TrimSpace(strings.ToLower(r.Intent))
	if _, ok := allowedIntents[intent]; !ok {
		intent = "general"
	}
	conf := r.Confidence
	if conf < 0 {
		conf = 0
	}
	if conf > 1 {
		conf = 1
	}
	return Result{
		Intent:     intent,
		Confidence: conf,
		Summary:    strings.TrimSpace(r.Summary),
	}
}

// EchoLine formats a Day-1 debug reply.
func EchoLine(r Result) string {
	return fmt.Sprintf("intent=%s conf=%.2f summary=%s", r.Intent, r.Confidence, r.Summary)
}

// FallbackResult is used when classification fails in echo mode.
func FallbackResult() Result {
	return Result{Intent: "general", Confidence: 0, Summary: "classify_failed"}
}
