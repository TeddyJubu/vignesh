package config

import "testing"

func TestResolvedAIProvider(t *testing.T) {
	c := &Config{AIProvider: "openai"}
	if c.ResolvedAIProvider() != "openai" {
		t.Fatalf("got %q", c.ResolvedAIProvider())
	}
	c.AIProvider = "invalid"
	if c.ResolvedAIProvider() != "ollama" {
		t.Fatalf("got %q", c.ResolvedAIProvider())
	}
}

func TestResolvedOpenAIBaseURL(t *testing.T) {
	c := &Config{}
	if c.ResolvedOpenAIBaseURL() != "https://sg.api.openai.com" {
		t.Fatalf("base=%q", c.ResolvedOpenAIBaseURL())
	}
}
