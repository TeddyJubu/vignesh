package ai

type StructuredResponse struct {
	Reply       string            `json:"reply"`
	LeadUpdates map[string]string `json:"lead_updates"`
	Qualified   bool              `json:"qualified"`
	Summary     string            `json:"summary"`
}

func ParseStructuredResponse(raw string) (*StructuredResponse, error) {
	return DecodeStructured(raw)
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
