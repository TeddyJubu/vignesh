package composio

import (
	"strings"
)

// Config holds Composio integration settings (DB with env override).
type Config struct {
	APIKey            string
	UserID            string
	CalendarAccountID string
	GmailAccountID    string
	Timezone          string
}

type settingsReader interface {
	Resolved(key, envKey string) (string, error)
}

// LoadConfig reads Composio settings from the settings resolver.
func LoadConfig(r settingsReader) (Config, error) {
	if r == nil {
		return Config{}, nil
	}
	var cfg Config
	var err error
	if cfg.APIKey, err = r.Resolved("composio.api_key", "COMPOSIO_API_KEY"); err != nil {
		return Config{}, err
	}
	if cfg.UserID, err = r.Resolved("composio.user_id", "COMPOSIO_USER_ID"); err != nil {
		return Config{}, err
	}
	if cfg.CalendarAccountID, err = r.Resolved("composio.calendar_connected_account_id", "COMPOSIO_CALENDAR_CONNECTED_ACCOUNT_ID"); err != nil {
		return Config{}, err
	}
	if cfg.GmailAccountID, err = r.Resolved("composio.gmail_connected_account_id", "COMPOSIO_GMAIL_CONNECTED_ACCOUNT_ID"); err != nil {
		return Config{}, err
	}
	if cfg.Timezone, err = r.Resolved("composio.timezone", "COMPOSIO_TIMEZONE"); err != nil {
		return Config{}, err
	}
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	cfg.UserID = strings.TrimSpace(cfg.UserID)
	cfg.CalendarAccountID = strings.TrimSpace(cfg.CalendarAccountID)
	cfg.GmailAccountID = strings.TrimSpace(cfg.GmailAccountID)
	cfg.Timezone = strings.TrimSpace(cfg.Timezone)
	if cfg.UserID == "" {
		cfg.UserID = "default"
	}
	if cfg.Timezone == "" {
		cfg.Timezone = "Asia/Singapore"
	}
	return cfg, nil
}

func (c Config) Configured() bool {
	return strings.TrimSpace(c.APIKey) != ""
}

func (c Config) CalendarReady() bool {
	return c.Configured() && c.CalendarAccountID != ""
}

func (c Config) GmailReady() bool {
	return c.Configured() && c.GmailAccountID != ""
}
