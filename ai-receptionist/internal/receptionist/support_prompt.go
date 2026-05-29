package receptionist

import (
	"os"
	"strings"

	"ai-receptionist/internal/ai"
	"ai-receptionist/internal/session"
	"ai-receptionist/internal/store"
)

const supportTaskInstruction = "TASK: Answer the above message as Julia. If you cannot answer confidently from the knowledge base, say so and offer to flag it for Vignesh. Do not guess. Do not fabricate pricing or features."

// UseStackedPromptLayout returns true when PROMPT_LAYOUT=stacked (legacy all-in-system).
func UseStackedPromptLayout() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("PROMPT_LAYOUT")), "stacked")
}

// BuildSupportUserTurn is the user message for the support agent (KNOWLEDGE + history + current + TASK).
func BuildSupportUserTurn(knowledgeBase, historyFormatted, currentMessage string) string {
	kb := strings.TrimSpace(knowledgeBase)
	if kb == "" {
		kb = "(empty)"
	}
	hist := strings.TrimSpace(historyFormatted)
	if hist == "" {
		hist = "(none)"
	}
	var b strings.Builder
	b.WriteString("EPICWARE KNOWLEDGE BASE:\n")
	b.WriteString(kb)
	b.WriteString("\n\nCONVERSATION HISTORY:\n")
	b.WriteString(hist)
	b.WriteString("\n\nCURRENT MESSAGE:\n")
	b.WriteString(strings.TrimSpace(currentMessage))
	b.WriteString("\n\n")
	b.WriteString(supportTaskInstruction)
	return b.String()
}

// BuildBundledSupportMessages implements system=[SOUL+runtime] and a single user turn per the support-agent spec.
func historyForBundle(history []store.Message, currentMessage string) []store.Message {
	if len(history) == 0 {
		return history
	}
	last := history[len(history)-1]
	if last.Role == "user" && strings.TrimSpace(last.Message) == strings.TrimSpace(currentMessage) {
		return history[:len(history)-1]
	}
	return history
}

func BuildBundledSupportMessages(system, knowledgeBase string, history []store.Message, currentMessage string) []ai.ChatMessage {
	histText := session.FormatLastTurnsForPrompt(historyForBundle(history, currentMessage), 5)
	userTurn := BuildSupportUserTurn(knowledgeBase, histText, currentMessage)
	return []ai.ChatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: userTurn},
	}
}
