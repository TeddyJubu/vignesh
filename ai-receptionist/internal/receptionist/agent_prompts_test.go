package receptionist

import (
	"strings"
	"testing"

	"ai-receptionist/internal/agent"
	"ai-receptionist/internal/agent/tools"
	"ai-receptionist/internal/ai"
)

var testToolReg = tools.DefaultRegistry()

func TestBuildPlannerMessages_IncludesSchema(t *testing.T) {
	msgs := []ai.ChatMessage{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "hi"},
	}
	out := buildPlannerMessages(msgs, true, testToolReg)
	if len(out) < 1 {
		t.Fatalf("len=%d", len(out))
	}
	if !strings.Contains(out[0].Content, "check_calendar_availability") {
		t.Fatalf("missing tool list: %q", out[0].Content)
	}
	if !strings.Contains(out[0].Content, `"final_response_mode"`) {
		t.Fatalf("missing schema: %q", out[0].Content)
	}
	if !strings.Contains(out[0].Content, `final_response_mode must be "structured"`) {
		t.Fatalf("missing structured rule: %q", out[0].Content)
	}
}

func TestBuildCollationMessages_EmbedsToolJSON(t *testing.T) {
	plan := &agent.Plan{Goal: "g", FinalResponseMode: "structured"}
	results := []agent.ToolResult{{TaskName: "t1", Tool: "align_time", Output: "ok"}}
	msgs := buildCollationMessages(plan, map[string]string{"q": "a"}, results, true)
	if len(msgs) < 1 {
		t.Fatalf("len=%d", len(msgs))
	}
	if !strings.Contains(msgs[0].Content, `"tool":"align_time"`) {
		t.Fatalf("missing tool output: %q", msgs[0].Content)
	}
	if !strings.Contains(msgs[0].Content, `Return ONLY a single JSON object`) {
		t.Fatalf("missing structured instruction: %q", msgs[0].Content)
	}
}

