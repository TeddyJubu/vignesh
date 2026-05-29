package ai

import (
	"errors"
	"strings"
	"testing"
)

func TestProviderFailureHint_OpenAIAuth(t *testing.T) {
	hint := ProviderFailureHint("openai", errors.New("OpenAI stream error: 401 unauthorized"))
	if !strings.Contains(hint, "OPENAI_API_KEY") {
		t.Fatalf("hint=%q", hint)
	}
}

func TestProviderFailureHint_OllamaAuth(t *testing.T) {
	hint := ProviderFailureHint("ollama", errors.New("HTTP 403"))
	if !strings.Contains(hint, "OLLAMA_API_KEY") {
		t.Fatalf("hint=%q", hint)
	}
}
