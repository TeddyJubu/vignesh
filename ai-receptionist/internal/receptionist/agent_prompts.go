package receptionist

import (
	"encoding/json"
	"strings"

	"ai-receptionist/internal/agent"
	"ai-receptionist/internal/ai"
)

func buildPlannerMessages(msgs []ai.ChatMessage, structured bool) []ai.ChatMessage {
	var b strings.Builder
	b.WriteString("You are a planner. Output ONLY valid JSON (no markdown). Schema:\n")
	b.WriteString(`{"goal":"string","agents":[{"name":"string","tool":"check_calendar_availability|collect_email|align_time|book_appointment","input":"string","expected_output":"string"}],"questions":["string"],"final_response_mode":"structured|text"}` + "\n")
	b.WriteString("Rules:\n")
	b.WriteString("- If you need missing info from the user, put it in questions[] and keep agents[] empty.\n")
	b.WriteString("- Max 4 agents.\n")
	b.WriteString("- Use stub tools only; no side effects.\n")
	if structured {
		b.WriteString("- final_response_mode must be \"structured\".\n")
	} else {
		b.WriteString("- final_response_mode must be \"text\".\n")
	}
	b.WriteString("\nConversation:\n")
	for _, m := range msgs {
		b.WriteString(m.Role)
		b.WriteString(": ")
		b.WriteString(m.Content)
		b.WriteString("\n")
	}
	return []ai.ChatMessage{
		{Role: "system", Content: b.String()},
		{Role: "user", Content: "Plan now."},
	}
}

func buildCollationMessages(plan *agent.Plan, answers map[string]string, results []agent.ToolResult, structured bool) []ai.ChatMessage {
	var b strings.Builder
	b.WriteString("You are a collation agent. Use the plan, user answers, and tool outputs to produce the final reply.\n")
	if structured {
		b.WriteString("Return ONLY a single JSON object compatible with this schema:\n")
		b.WriteString(`{"reply":"string","lead_updates":{"key":"value"},"qualified":false,"summary":"string"}` + "\n")
		b.WriteString("Do not wrap in markdown fences.\n")
	} else {
		b.WriteString("Return a short natural-language reply.\n")
	}
	b.WriteString("\nPlan goal:\n")
	b.WriteString(strings.TrimSpace(plan.Goal))
	b.WriteString("\n\nUser answers:\n")
	if len(answers) == 0 {
		b.WriteString("(none)\n")
	} else {
		keys := make([]string, 0, len(answers))
		for k := range answers {
			keys = append(keys, k)
		}
		// stable enough without sorting; not user-visible.
		for _, q := range keys {
			b.WriteString("- ")
			b.WriteString(q)
			b.WriteString(": ")
			b.WriteString(answers[q])
			b.WriteString("\n")
		}
	}
	b.WriteString("\nTool outputs:\n")
	j, _ := json.Marshal(results)
	b.Write(j)
	b.WriteString("\n")

	return []ai.ChatMessage{
		{Role: "system", Content: b.String()},
		{Role: "user", Content: "Produce the final response now."},
	}
}

