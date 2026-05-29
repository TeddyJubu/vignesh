package httpapi

import (
	"net/http"
	"os"
	"strings"

	"ai-receptionist/internal/settings"
)

type providerStatus struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	Configured bool   `json:"configured"`
}

func resolveProviderStatus(r *settings.Resolver, cfgProvider, cfgModel string) providerStatus {
	provider := strings.ToLower(strings.TrimSpace(cfgProvider))
	if r != nil {
		if p, err := r.ResolvedAIProvider(); err == nil && p != "" {
			provider = p
		}
	}
	if provider == "" {
		provider = "ollama"
	}

	model := strings.TrimSpace(cfgModel)
	configured := providerConfigured(r, provider)

	switch provider {
	case "openai":
		if r != nil {
			if m, _ := r.Get("openai.model"); strings.TrimSpace(m) != "" {
				model = strings.TrimSpace(m)
			}
		}
	case "anthropic":
		if r != nil {
			if m, _ := r.Get("anthropic.model"); strings.TrimSpace(m) != "" {
				model = strings.TrimSpace(m)
			} else {
				model = "claude-sonnet-4-6"
			}
		}
	case "openrouter":
		if r != nil {
			if m, _ := r.Get("openrouter.model"); strings.TrimSpace(m) != "" {
				model = strings.TrimSpace(m)
			}
		}
	case "custom":
		if r != nil {
			if m, _ := r.Get("custom.model"); strings.TrimSpace(m) != "" {
				model = strings.TrimSpace(m)
			}
		}
	default:
		if r != nil {
			if m, _ := r.Get("ollama.model"); strings.TrimSpace(m) != "" {
				model = strings.TrimSpace(m)
			}
		}
		if model == "" {
			model = strings.TrimSpace(os.Getenv("OLLAMA_MODEL"))
		}
	}

	return providerStatus{
		Provider:   provider,
		Model:      model,
		Configured: configured,
	}
}

func providerConfigured(r *settings.Resolver, provider string) bool {
	if r == nil {
		return false
	}
	switch provider {
	case "openai":
		key, _ := r.Resolved("openai.api_key", "OPENAI_API_KEY")
		return strings.TrimSpace(key) != ""
	case "anthropic":
		key, _ := r.Resolved("anthropic.api_key", "ANTHROPIC_API_KEY")
		return strings.TrimSpace(key) != ""
	case "openrouter":
		key, _ := r.Resolved("openrouter.api_key", "OPENROUTER_API_KEY")
		return strings.TrimSpace(key) != ""
	case "custom":
		key, _ := r.Resolved("custom.api_key", "CUSTOM_API_KEY")
		base, _ := r.Resolved("custom.base_url", "CUSTOM_BASE_URL")
		return strings.TrimSpace(key) != "" && strings.TrimSpace(base) != ""
	default:
		key, _ := r.Resolved("ollama.api_key", "OLLAMA_API_KEY")
		return strings.TrimSpace(key) != ""
	}
}

func (s *Server) handleProviderStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	st := resolveProviderStatus(s.settings, s.cfg.ResolvedAIProvider(), s.cfg.Model)
	writeJSON(w, 200, map[string]any{
		"provider":   st.Provider,
		"model":      st.Model,
		"configured": st.Configured,
	})
}
