package ai

import (
	"context"

	"ai-receptionist/internal/aiface"
)

// AsAIFace wraps a Provider for packages that must not import internal/ai.
func AsAIFace(p Provider) aiface.Provider {
	if p == nil {
		return nil
	}
	return providerAdapter{p}
}

type providerAdapter struct{ Provider }

func (a providerAdapter) Complete(ctx context.Context, messages []aiface.Message, jsonMode bool) (string, error) {
	msgs := make([]ChatMessage, len(messages))
	for i, m := range messages {
		msgs[i] = ChatMessage{Role: m.Role, Content: m.Content}
	}
	return a.Provider.Complete(ctx, msgs, jsonMode)
}
