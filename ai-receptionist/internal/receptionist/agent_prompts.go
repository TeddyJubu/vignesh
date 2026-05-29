package receptionist

import (
	"encoding/json"
	"fmt"
	"strings"

	"ai-receptionist/internal/agent"
	"ai-receptionist/internal/agent/tools"
	"ai-receptionist/internal/ai"
)

func buildPlannerMessages(msgs []ai.ChatMessage, structured bool, reg *tools.Registry) []ai.ChatMessage {
	toolList := reg.PlannerToolList()
	var b strings.Builder
	b.WriteString("You are a planner for a WhatsApp business receptionist named Julia.\n")
	b.WriteString("Output ONLY valid JSON (no markdown). Schema:\n")
	b.WriteString(`{"goal":"string","agents":[{"name":"string","tool":"` + toolList + `","input":"string","expected_output":"string"}],"questions":["string"],"final_response_mode":"structured|text"}` + "\n")
	b.WriteString("Rules:\n")
	b.WriteString("- If you need missing info from the user, put it in questions[] and keep agents[] empty.\n")
	b.WriteString("- questions[] MUST be written in Julia's voice (warm, concise WhatsApp tone), not as a meta/planner.\n")
	b.WriteString("- If the user asks who you are / your name, answer plainly: \"I'm Julia\" + one short line about purpose.\n")
	b.WriteString("- Never ask for email before the user has given a preferred day, time, and timezone for a call.\n")
	b.WriteString("- For \"what services\" questions, answer from the business description — never repeat the user's question.\n")
	b.WriteString("- questions[] must contain exactly ONE question mark total.\n")
	b.WriteString("- Max ")
	b.WriteString(fmt.Sprintf("%d", agent.MaxPlannerAgents))
	b.WriteString(" agents.\n")
	b.WriteString("- Use registered tools only.\n")
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

func buildPlannerRepairMessages(invalid string, structured bool, reg *tools.Registry) []ai.ChatMessage {
	toolList := reg.PlannerToolList()
	schema := `{"goal":"string","agents":[{"name":"string","tool":"` + toolList + `","input":"string","expected_output":"string"}],"questions":["string"],"final_response_mode":"structured|text"}`
	mode := "text"
	if structured {
		mode = "structured"
	}
	return []ai.ChatMessage{
		{
			Role: "system",
			Content: "Fix the following into ONE valid JSON planner object. Schema: " + schema +
				"\nMax 4 agents. final_response_mode must be \"" + mode + "\". No markdown." +
				"\nIf questions[] is non-empty, write questions in Julia's WhatsApp voice (and say \"I'm Julia\" if asked name).",
		},
		{Role: "user", Content: invalid},
	}
}

func buildCollationMessages(plan *agent.Plan, answers map[string]string, results []agent.ToolResult, structured bool) []ai.ChatMessage {
	var b strings.Builder
	b.WriteString("You are a collation agent for Julia (WhatsApp receptionist). Use the plan and user answers to produce the final customer-facing reply.\n")
	b.WriteString("CRITICAL: Never paste tool JSON, conversation IDs, internal alerts, escalation dumps, or owner-only summaries into the reply.\n")
	b.WriteString("Never say you are a generic AI assistant or disclose model/provider. You are Julia.\n")
	b.WriteString("Booking: ask day + time + timezone before email; only confirm booking if book_appointment returned booked:true.\n")
	b.WriteString("Keep reply under 3 short lines and at most ONE question.\n")
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
		for q, a := range answers {
			b.WriteString("- ")
			b.WriteString(q)
			b.WriteString(": ")
			b.WriteString(a)
			b.WriteString("\n")
		}
	}
	b.WriteString("\nTool outputs (internal — summarize for customer, do not copy verbatim):\n")
	safe := make([]agent.ToolResult, len(results))
	for i, r := range results {
		safe[i] = r
		safe[i].Output = CustomerSafeToolOutput(r.Tool, r.Output)
	}
	j, _ := json.Marshal(safe)
	b.Write(j)
	b.WriteString("\n")

	return []ai.ChatMessage{
		{Role: "system", Content: b.String()},
		{Role: "user", Content: "Produce the final response now."},
	}
}
