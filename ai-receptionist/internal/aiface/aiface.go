package aiface

import "context"

// Message is one chat turn for completion APIs.
type Message struct {
	Role    string
	Content string
}

// Provider completes chat prompts (implemented by internal/ai providers).
type Provider interface {
	Complete(ctx context.Context, messages []Message, jsonMode bool) (string, error)
}
