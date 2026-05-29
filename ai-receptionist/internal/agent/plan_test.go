package agent

import "testing"

func TestParsePlan_StripsCodeFences(t *testing.T) {
	raw := "```json\n" +
		`{"goal":"book","agents":[],"questions":["What day?"],"final_response_mode":"structured"}` +
		"\n```"
	p, err := ParsePlan(raw)
	if err != nil {
		t.Fatalf("ParsePlan error: %v", err)
	}
	if p.Goal != "book" {
		t.Fatalf("goal=%q", p.Goal)
	}
	if len(p.Questions) != 1 || p.Questions[0] != "What day?" {
		t.Fatalf("questions=%v", p.Questions)
	}
	if p.FinalResponseMode != "structured" {
		t.Fatalf("mode=%q", p.FinalResponseMode)
	}
}

func TestNormalizePlan_TruncatesAgentsAndClearsWhenQuestions(t *testing.T) {
	p := &Plan{
		Goal: "book",
		Agents: []AgentTask{
			{Name: "a1", Tool: "align_time", Input: "x"},
			{Name: "a2", Tool: "align_time", Input: "x"},
			{Name: "a3", Tool: "align_time", Input: "x"},
			{Name: "a4", Tool: "align_time", Input: "x"},
			{Name: "a5", Tool: "align_time", Input: "x"},
		},
		Questions: []string{"What day?"},
	}
	NormalizePlan(p, true)
	if len(p.Agents) != MaxPlannerAgents {
		t.Fatalf("agents should truncate when questions present, got %d", len(p.Agents))
	}
	if p.FinalResponseMode != "structured" {
		t.Fatalf("mode=%q", p.FinalResponseMode)
	}

	p2 := &Plan{
		Goal: "slots",
		Agents: []AgentTask{
			{Name: "a1", Tool: "align_time"},
			{Name: "a2", Tool: "align_time"},
			{Name: "a3", Tool: "align_time"},
			{Name: "a4", Tool: "align_time"},
			{Name: "a5", Tool: "align_time"},
		},
	}
	NormalizePlan(p2, false)
	if len(p2.Agents) != MaxPlannerAgents {
		t.Fatalf("agents=%d want %d", len(p2.Agents), MaxPlannerAgents)
	}
}

func TestParsePlan_NormalizesMode(t *testing.T) {
	raw := `{"goal":"x","agents":[],"questions":[],"final_response_mode":"INVALID"}`
	p, err := ParsePlan(raw)
	if err != nil {
		t.Fatalf("ParsePlan error: %v", err)
	}
	if p.FinalResponseMode != "text" {
		t.Fatalf("mode=%q", p.FinalResponseMode)
	}
}

