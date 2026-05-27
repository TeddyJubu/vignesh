package settings

import (
	"os"
	"strings"

	"ai-receptionist/internal/store"
)

// Resolver loads settings from SQLite (app_settings) with env overrides.
//
// Precedence: env > DB.
type Resolver struct {
	db *store.DB
}

func New(db *store.DB) *Resolver {
	return &Resolver{db: db}
}

func (r *Resolver) Get(key string) (string, error) {
	return r.db.GetAppSetting(key)
}

func (r *Resolver) Resolved(key, envKey string) (string, error) {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v, nil
	}
	return r.Get(key)
}

func (r *Resolver) ResolvedAIProvider() (string, error) {
	v, err := r.Get("ai.provider")
	if err != nil {
		return "", err
	}
	p := strings.ToLower(strings.TrimSpace(v))
	switch p {
	case "openai", "anthropic", "openrouter", "custom", "ollama":
		return p, nil
	default:
		return "", nil
	}
}
