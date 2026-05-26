package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Plan struct {
	Goal              string      `json:"goal"`
	Agents            []AgentTask `json:"agents"`
	Questions         []string    `json:"questions"`
	FinalResponseMode string      `json:"final_response_mode"` // "structured" or "text"
}

type AgentTask struct {
	Name           string `json:"name"`
	Tool           string `json:"tool"`
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
}

type State struct {
	Plan        Plan              `json:"plan"`
	NextQIndex  int               `json:"next_q_index"`
	Answers     map[string]string `json:"answers"` // question -> answer
	StartedAtUNIX int64           `json:"started_at_unix"`
}

func ParsePlan(raw string) (*Plan, error) {
	raw = strings.TrimSpace(raw)
	raw = stripCodeFences(raw)
	var p Plan
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return nil, fmt.Errorf("parse planner JSON: %w", err)
	}
	if strings.TrimSpace(p.Goal) == "" && len(p.Agents) == 0 && len(p.Questions) == 0 {
		return nil, fmt.Errorf("planner returned empty plan")
	}
	// normalize
	if p.Agents == nil {
		p.Agents = nil
	}
	if p.Questions == nil {
		p.Questions = nil
	}
	mode := strings.ToLower(strings.TrimSpace(p.FinalResponseMode))
	if mode != "structured" {
		mode = "text"
	}
	p.FinalResponseMode = mode
	return &p, nil
}

func stripCodeFences(raw string) string {
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

