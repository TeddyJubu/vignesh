package models

import (
	"testing"
)

func TestGetModel_intentClassify_envOverride(t *testing.T) {
	t.Setenv("INTENT_CLASSIFY_MODEL", "custom-intent-model")
	SetActiveProvider("anthropic")
	SetConfigModel("cfg-model")
	if got := GetModel("intent_classify"); got != "custom-intent-model" {
		t.Fatalf("got %q want custom-intent-model", got)
	}
}

func TestGetModel_intentClassify_anthropicUsesHaiku(t *testing.T) {
	t.Setenv("INTENT_CLASSIFY_MODEL", "")
	SetActiveProvider("anthropic")
	SetSettingsModelResolver(func() string { return "claude-sonnet-4-6" })
	if got := GetModel("intent_classify"); got != AnthropicModelHaiku {
		t.Fatalf("got %q want %q", got, AnthropicModelHaiku)
	}
	SetSettingsModelResolver(nil)
}

func TestGetModel_intentClassify_ollamaUsesConfig(t *testing.T) {
	t.Setenv("INTENT_CLASSIFY_MODEL", "")
	SetActiveProvider("ollama")
	SetSettingsModelResolver(nil)
	SetConfigModel("nemotron-3-super:cloud")
	if got := GetModel("intent_classify"); got != "nemotron-3-super:cloud" {
		t.Fatalf("got %q", got)
	}
}

func TestGetModel_intentClassify_ollamaFallback(t *testing.T) {
	t.Setenv("INTENT_CLASSIFY_MODEL", "")
	SetActiveProvider("ollama")
	SetConfigModel("")
	SetSettingsModelResolver(nil)
	if got := GetModel("intent_classify"); got != OllamaModelDefault {
		t.Fatalf("got %q", got)
	}
}

func TestGetModel_planner_anthropicUsesDashboardSonnet(t *testing.T) {
	SetActiveProvider("anthropic")
	SetConfigModel("ignored")
	SetSettingsModelResolver(func() string { return "claude-sonnet-4-6" })
	if got := GetModel("planner"); got != "claude-sonnet-4-6" {
		t.Fatalf("got %q", got)
	}
	SetSettingsModelResolver(nil)
}

func TestGetModel_planner_anthropicDefaultWhenUnset(t *testing.T) {
	SetActiveProvider("anthropic")
	SetConfigModel("")
	SetSettingsModelResolver(nil)
	if got := GetModel("planner"); got != AnthropicModelSonnet {
		t.Fatalf("got %q", got)
	}
}
