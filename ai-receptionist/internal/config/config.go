package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	BusinessName        string `json:"business_name"`
	OwnerNumber         string `json:"owner_number"`
	AIProvider          string `json:"ai_provider"`
	Model               string `json:"model"`
	BusinessDescription string `json:"business_description"`

	// Mode: "receptionist" (default) or "personal" (reply in your voice, no lead funnel).
	Mode string `json:"mode"`
	// ReplyToGroups allows auto-reply in WhatsApp groups (default false).
	ReplyToGroups bool `json:"reply_to_groups"`
	// EnableLeadTracking runs qualification + lead_data (receptionist default: true).
	EnableLeadTracking *bool `json:"enable_lead_tracking,omitempty"`
	// EnableOwnerAlerts sends qualified-lead summary to owner_number (receptionist default: true).
	EnableOwnerAlerts *bool `json:"enable_owner_alerts,omitempty"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if strings.TrimSpace(c.BusinessName) == "" {
		return nil, fmt.Errorf("business_name is required")
	}
	if strings.TrimSpace(c.OwnerNumber) == "" {
		return nil, fmt.Errorf("owner_number is required")
	}
	if strings.TrimSpace(c.Model) == "" {
		c.Model = "openai/gpt-4.1-mini"
	}
	if strings.TrimSpace(c.AIProvider) == "" {
		c.AIProvider = "openrouter"
	}
	c.OwnerNumber = NormalizePhone(c.OwnerNumber)
	c.applyModeDefaults()
	return &c, nil
}

func (c *Config) applyModeDefaults() {
	mode := strings.ToLower(strings.TrimSpace(c.Mode))
	if mode == "" {
		mode = "receptionist"
	}
	if mode != "personal" {
		mode = "receptionist"
	}
	c.Mode = mode

	if mode == "personal" {
		f := false
		c.EnableLeadTracking = &f
		c.EnableOwnerAlerts = &f
		return
	}
	if c.EnableLeadTracking == nil {
		t := true
		c.EnableLeadTracking = &t
	}
	if c.EnableOwnerAlerts == nil {
		t := true
		c.EnableOwnerAlerts = &t
	}
}

func (c *Config) IsPersonal() bool {
	return c.Mode == "personal"
}

func (c *Config) LeadTrackingEnabled() bool {
	return c.EnableLeadTracking != nil && *c.EnableLeadTracking
}

func (c *Config) OwnerAlertsEnabled() bool {
	return c.EnableOwnerAlerts != nil && *c.EnableOwnerAlerts
}

// NormalizePhone keeps digits only (country code, no +).
func NormalizePhone(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
