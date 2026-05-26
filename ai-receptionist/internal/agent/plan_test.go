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

