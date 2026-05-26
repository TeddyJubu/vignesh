package ai

import (
	"fmt"

	"ai-receptionist/internal/config"
)

func NewProvider(cfg *config.Config) (Provider, error) {
	switch cfg.ResolvedAIProvider() {
	case "openai":
		return NewOpenAIProvider(cfg.ResolvedOpenAIModel(), cfg.ResolvedOpenAIBaseURL())
	default:
		return NewClient(cfg.Model)
	}
}

func ProviderFailureHint(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}

