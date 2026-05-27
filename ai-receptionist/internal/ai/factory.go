package ai

import (
	"fmt"

	"ai-receptionist/internal/config"
	"ai-receptionist/internal/settings"
)

func NewProvider(cfg *config.Config) (Provider, error) {
	switch cfg.ResolvedAIProvider() {
	case "openai":
		return NewOpenAIProvider(cfg.ResolvedOpenAIModel(), cfg.ResolvedOpenAIBaseURL())
	default:
		return NewClient(cfg.Model)
	}
}

func NewProviderFromSettings(cfg *config.Config, r *settings.Resolver) (Provider, error) {
	if r == nil {
		return NewProvider(cfg)
	}
	p, err := r.ResolvedAIProvider()
	if err != nil {
		return nil, err
	}
	if p == "" {
		return NewProvider(cfg)
	}
	switch p {
	case "openai":
		key, err := r.Resolved("openai.api_key", "OPENAI_API_KEY")
		if err != nil {
			return nil, err
		}
		model, _ := r.Get("openai.model")
		baseURL, _ := r.Resolved("openai.base_url", "OPENAI_BASE_URL")
		return NewOpenAIProviderWithKey(model, baseURL, key)
	case "anthropic":
		key, err := r.Resolved("anthropic.api_key", "ANTHROPIC_API_KEY")
		if err != nil {
			return nil, err
		}
		model, _ := r.Get("anthropic.model")
		return NewAnthropicProvider(model, key)
	case "openrouter":
		key, err := r.Resolved("openrouter.api_key", "OPENROUTER_API_KEY")
		if err != nil {
			return nil, err
		}
		model, _ := r.Get("openrouter.model")
		return NewOpenRouterProvider(model, key)
	case "custom":
		key, err := r.Resolved("custom.api_key", "CUSTOM_API_KEY")
		if err != nil {
			return nil, err
		}
		model, _ := r.Get("custom.model")
		baseURL, _ := r.Resolved("custom.base_url", "CUSTOM_BASE_URL")
		return NewCustomProvider(model, baseURL, key)
	default:
		key, err := r.Resolved("ollama.api_key", "OLLAMA_API_KEY")
		if err != nil {
			return nil, err
		}
		model, _ := r.Get("ollama.model")
		url, _ := r.Resolved("ollama.api_url", "OLLAMA_API_URL")
		return NewClientWithConfig(model, key, url)
	}
}

func ProviderFailureHint(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}
