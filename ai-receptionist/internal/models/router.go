package models

import (
	"os"
	"strings"
)

// Anthropic API model IDs (pinned; see https://docs.anthropic.com/en/docs/about-claude/models/overview).
const (
	AnthropicModelSonnet = "claude-sonnet-4-6"
	AnthropicModelOpus   = "claude-opus-4-8"
	AnthropicModelHaiku  = "claude-haiku-4-5-20251001"
)

// OpenAI defaults (Singapore gateway compatible).
const (
	OpenAIModelFast  = "gpt-4.1-mini"
	OpenAIModelMain  = "gpt-4.1-mini"
)

// Ollama Cloud default when config/dashboard model unset.
const OllamaModelDefault = "gemma4:31b-cloud"

var (
	defaultCfgModel     string
	settingsModelFunc   func() string
	activeProvider      = "ollama"
)

// SetConfigModel sets the model from config.json (call once at startup from main).
func SetConfigModel(model string) {
	defaultCfgModel = strings.TrimSpace(model)
}

// SetSettingsModelResolver supplies the active dashboard provider model (call once at startup).
func SetSettingsModelResolver(fn func() string) {
	settingsModelFunc = fn
}

// SetActiveProvider sets the resolved AI provider for task routing (ollama|openai|anthropic|...).
func SetActiveProvider(provider string) {
	p := strings.ToLower(strings.TrimSpace(provider))
	if p == "" {
		p = "ollama"
	}
	activeProvider = p
}

// ActiveProvider returns the provider set via SetActiveProvider.
func ActiveProvider() string {
	if activeProvider == "" {
		return "ollama"
	}
	return activeProvider
}

// GetModel resolves the model name for a task type (plan: single router, no scattered strings).
//
// Precedence: task-specific env override > provider/task defaults > dashboard model > config.json model.
func GetModel(taskType string) string {
	if m := envModelOverride(taskType); m != "" {
		return m
	}
	switch strings.TrimSpace(taskType) {
	case "intent_classify":
		return modelForIntentClassify()
	case "planner", "planner_repair":
		return modelForMainWork()
	case "collate":
		return modelForMainWork()
	case "fast":
		return modelForMainWork()
	default:
		return modelForMainWork()
	}
}

func envModelOverride(taskType string) string {
	switch strings.TrimSpace(taskType) {
	case "intent_classify":
		return strings.TrimSpace(os.Getenv("INTENT_CLASSIFY_MODEL"))
	case "planner", "planner_repair":
		return strings.TrimSpace(os.Getenv("PLANNER_MODEL"))
	case "collate":
		return strings.TrimSpace(os.Getenv("COLLATE_MODEL"))
	default:
		return ""
	}
}

func modelForIntentClassify() string {
	switch ActiveProvider() {
	case "anthropic":
		return AnthropicModelHaiku
	case "openai":
		return OpenAIModelFast
	default:
		if m := dashboardOrConfigModel(); m != "" {
			return m
		}
		return OllamaModelDefault
	}
}

func modelForMainWork() string {
	if m := dashboardOrConfigModel(); m != "" && modelMatchesProvider(m, ActiveProvider()) {
		return m
	}
	switch ActiveProvider() {
	case "anthropic":
		return AnthropicModelSonnet
	case "openai":
		return OpenAIModelMain
	default:
		if m := dashboardOrConfigModel(); m != "" {
			return m
		}
		return OllamaModelDefault
	}
}

func modelMatchesProvider(model, provider string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "anthropic":
		return strings.HasPrefix(m, "claude")
	case "openai":
		return strings.HasPrefix(m, "gpt-") || strings.HasPrefix(m, "o1") || strings.HasPrefix(m, "o3")
	default:
		return true
	}
}

func dashboardOrConfigModel() string {
	if settingsModelFunc != nil {
		if m := strings.TrimSpace(settingsModelFunc()); m != "" {
			return m
		}
	}
	return strings.TrimSpace(defaultCfgModel)
}
