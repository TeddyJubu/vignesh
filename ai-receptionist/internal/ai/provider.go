package ai

import "context"

// Provider is an abstract AI backend (Ollama, OpenAI, etc.).
//
// jsonMode indicates the caller expects a raw JSON object compatible with
// ParseStructuredResponse (may be wrapped in code fences, which is handled).
type Provider interface {
	Name() string
	Complete(ctx context.Context, messages []ChatMessage, jsonMode bool) (string, error)
	Ping(ctx context.Context) error
}

