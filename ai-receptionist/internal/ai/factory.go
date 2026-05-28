package ai

import (
	"fmt"
	"os"
	"strings"

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

// NewProviderForModel returns the same provider/credentials as settings resolution, with an overridden model.
func NewProviderForModel(cfg *config.Config, r *settings.Resolver, model string) (Provider, error) {
	model = strings.TrimSpace(model)
	if r == nil {
		return newProviderWithModel(cfg, cfg.ResolvedAIProvider(), model)
	}
	p, err := r.ResolvedAIProvider()
	if err != nil {
		return nil, err
	}
	if p == "" {
		return newProviderWithModel(cfg, cfg.ResolvedAIProvider(), model)
	}
	if model == "" {
		return NewProviderFromSettings(cfg, r)
	}
	switch p {
	case "openai":
		key, err := r.Resolved("openai.api_key", "OPENAI_API_KEY")
		if err != nil {
			return nil, err
		}
		baseURL, _ := r.Resolved("openai.base_url", "OPENAI_BASE_URL")
		return NewOpenAIProviderWithKey(model, baseURL, key)
	case "anthropic":
		key, err := r.Resolved("anthropic.api_key", "ANTHROPIC_API_KEY")
		if err != nil {
			return nil, err
		}
		return NewAnthropicProvider(model, key)
	case "openrouter":
		key, err := r.Resolved("openrouter.api_key", "OPENROUTER_API_KEY")
		if err != nil {
			return nil, err
		}
		return NewOpenRouterProvider(model, key)
	case "custom":
		key, err := r.Resolved("custom.api_key", "CUSTOM_API_KEY")
		if err != nil {
			return nil, err
		}
		baseURL, _ := r.Resolved("custom.base_url", "CUSTOM_BASE_URL")
		return NewCustomProvider(model, baseURL, key)
	default:
		key, err := r.Resolved("ollama.api_key", "OLLAMA_API_KEY")
		if err != nil {
			return nil, err
		}
		url, _ := r.Resolved("ollama.api_url", "OLLAMA_API_URL")
		return NewClientWithConfig(model, key, url)
	}
}

func newProviderWithModel(cfg *config.Config, providerName, model string) (Provider, error) {
	if strings.TrimSpace(model) == "" {
		return NewProvider(cfg)
	}
	switch providerName {
	case "openai":
		return NewOpenAIProvider(model, cfg.ResolvedOpenAIBaseURL())
	case "anthropic":
		key := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
		if key == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY is not set")
		}
		return NewAnthropicProvider(model, key)
	case "openrouter":
		key := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
		if key == "" {
			return nil, fmt.Errorf("OPENROUTER_API_KEY is not set")
		}
		return NewOpenRouterProvider(model, key)
	case "custom":
		key := strings.TrimSpace(os.Getenv("CUSTOM_API_KEY"))
		baseURL := strings.TrimSpace(os.Getenv("CUSTOM_BASE_URL"))
		return NewCustomProvider(model, baseURL, key)
	default:
		return NewClientWithConfig(model, strings.TrimSpace(os.Getenv("OLLAMA_API_KEY")), strings.TrimSpace(os.Getenv("OLLAMA_API_URL")))
	}
}

func ProviderFailureHint(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}
