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

confidence is 0.0–1.0. summary is one short sentence describing what the user wants.`

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
	raw, err := p.Complete(cctx, msgs, true)
	if err != nil {
		return Result{}, err
	}
	result, err := decodeResult(raw)
	if err == nil {
		return normalizeResult(result), nil
	}
	repaired, err2 := p.Complete(cctx, ai.RepairIntentPrompt(raw), true)
	if err2 != nil {
		return Result{}, err
	}
	result, err = decodeResult(repaired)
	if err != nil {
		return Result{}, err
	}
	return normalizeResult(result), nil
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
