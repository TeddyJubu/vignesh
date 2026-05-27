package ai

import (
	"testing"

	"ai-receptionist/internal/config"
	"ai-receptionist/internal/settings"
	"ai-receptionist/internal/store"
)

func TestNewProviderFromSettings_UsesDBProvider(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.UpsertAppSetting("ai.provider", "openrouter"); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertAppSetting("openrouter.api_key", "test-key"); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertAppSetting("openrouter.model", "gpt-4.1-mini"); err != nil {
		t.Fatal(err)
	}

	p, err := NewProviderFromSettings(&config.Config{BusinessName: "Acme", Model: "x"}, settings.New(db))
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "openrouter" {
		t.Fatalf("provider=%q", p.Name())
	}
}

func TestNewProviderFromSettings_EnvOverridesSecrets(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "env-key")

	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.UpsertAppSetting("ai.provider", "openai"); err != nil {
		t.Fatal(err)
	}
	// OPENAI_API_KEY should be taken from env even when DB has empty value.
	if err := db.UpsertAppSetting("openai.api_key", ""); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertAppSetting("openai.model", "gpt-4.1-mini"); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertAppSetting("openai.base_url", "https://example.com"); err != nil {
		t.Fatal(err)
	}

	p, err := NewProviderFromSettings(&config.Config{BusinessName: "Acme", Model: "x"}, settings.New(db))
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "openai" {
		t.Fatalf("provider=%q", p.Name())
	}
}
