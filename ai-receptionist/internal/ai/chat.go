package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

type StructuredResponse struct {
	Reply       string            `json:"reply"`
	LeadUpdates map[string]string `json:"lead_updates"`
	Qualified   bool              `json:"qualified"`
	Summary     string            `json:"summary"`
}

func ParseStructuredResponse(raw string) (*StructuredResponse, error) {
	raw = strings.TrimSpace(raw)
	// strip markdown code fences if model adds them
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) >= 2 {
			if lines[0] == "```" || strings.HasPrefix(lines[0], "```json") {
				lines = lines[1:]
			}
			if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
				lines = lines[:len(lines)-1]
			}
			raw = strings.Join(lines, "\n")
		}
	}
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

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func BuildMessages(systemPrompt string, history []ChatMessage, userText string) []ChatMessage {
	msgs := make([]ChatMessage, 0, len(history)+2)
	msgs = append(msgs, ChatMessage{Role: "system", Content: systemPrompt})
	msgs = append(msgs, history...)
	msgs = append(msgs, ChatMessage{Role: "user", Content: userText})
	return msgs
}
