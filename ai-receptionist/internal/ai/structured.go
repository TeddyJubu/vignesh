package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DecodeStructured parses receptionist JSON with fence stripping and validation.
func DecodeStructured(raw string) (*StructuredResponse, error) {
	raw = StripCodeFences(raw)
	var r StructuredResponse
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return nil, fmt.Errorf("parse AI JSON: %w", err)
	}
	if strings.TrimSpace(r.Reply) == "" {
		return nil, fmt.Errorf("AI response missing reply")
	}
	if r.LeadUpdates == nil {
		r.LeadUpdates = map[string]string{}
	}
	return &r, nil
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
				"\nNo markdown fences.",
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
