package ai

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// DecodeStructured parses receptionist JSON with fence stripping and validation.
func DecodeStructured(raw string) (*StructuredResponse, error) {
	raw = StripCodeFences(raw)
	var wire structuredWire
	if err := json.Unmarshal([]byte(raw), &wire); err != nil {
		return nil, fmt.Errorf("parse AI JSON: %w", err)
	}
	r := StructuredResponse{
		Reply:       strings.TrimSpace(wire.Reply),
		LeadUpdates: normalizeLeadUpdates(wire.LeadUpdates),
		Qualified:   wire.Qualified,
		Summary:     strings.TrimSpace(wire.Summary),
	}
	if r.Reply == "" {
		return nil, fmt.Errorf("AI response missing reply")
	}
	if r.LeadUpdates == nil {
		r.LeadUpdates = map[string]string{}
	}
	return &r, nil
}

type structuredWire struct {
	Reply       string         `json:"reply"`
	LeadUpdates map[string]any `json:"lead_updates"`
	Qualified   bool           `json:"qualified"`
	Summary     string         `json:"summary"`
}

func normalizeLeadUpdates(in map[string]any) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		if s := leadUpdateString(v); s != "" {
			out[key] = s
		}
	}
	return out
}

func leadUpdateString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return strings.TrimSpace(t.String())
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

// StripCodeFences removes markdown code fences from model output.
func StripCodeFences(raw string) string {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "```") {
		return raw
	}
	lines := strings.Split(raw, "\n")
	if len(lines) >= 2 {
		if lines[0] == "```" || strings.HasPrefix(lines[0], "```json") {
			lines = lines[1:]
		}
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
			lines = lines[:len(lines)-1]
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// RepairStructuredPrompt asks the model to fix invalid JSON (one retry).
func RepairStructuredPrompt(invalid string) []ChatMessage {
	return []ChatMessage{
		{
			Role: "system",
			Content: "Fix the following into ONE valid JSON object only. Schema: " +
				`{"reply":"string","lead_updates":{},"qualified":false,"summary":"string"}` +
				"\nNo markdown fences. lead_updates values must be strings.",
		},
		{Role: "user", Content: invalid},
	}
}

// RepairIntentPrompt asks the model to fix invalid intent-classifier JSON (one retry).
func RepairIntentPrompt(invalid string) []ChatMessage {
	return []ChatMessage{
		{
			Role: "system",
			Content: "Fix the following into ONE valid JSON object only. Schema: " +
				`{"intent":"general","confidence":0.0,"summary":"string"}` +
				"\nNo markdown fences.",
		},
		{Role: "user", Content: invalid},
	}
}
