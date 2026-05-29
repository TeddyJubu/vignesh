package ai

import (
	"testing"
)

func TestNewOpenAIProviderWithKey_DefaultsSingaporeBaseURL(t *testing.T) {
	p, err := NewOpenAIProviderWithKey("", "", "sk-test")
	if err != nil {
		t.Fatal(err)
	}
	if p.model != "gpt-4.1-mini" {
		t.Fatalf("model=%q", p.model)
	}
	if p.baseURL != defaultOpenAIBaseURL {
		t.Fatalf("baseURL=%q want %q", p.baseURL, defaultOpenAIBaseURL)
	}
}

func TestNewOpenAIProviderWithKey_RequiresKey(t *testing.T) {
	_, err := NewOpenAIProviderWithKey("gpt-4.1-mini", defaultOpenAIBaseURL, "")
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}
