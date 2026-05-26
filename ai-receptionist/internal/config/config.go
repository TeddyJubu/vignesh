package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type QuietHours struct {
	Enabled bool   `json:"enabled"`
	TZ      string `json:"tz"`
	Start   string `json:"start"` // HH:MM
	End     string `json:"end"`   // HH:MM
	Message string `json:"message,omitempty"`
}

type FollowUpNudge struct {
	Enabled   bool   `json:"enabled"`
	IdleHours int    `json:"idle_hours,omitempty"` // default 24
	Message   string `json:"message,omitempty"`
}

type Config struct {
	BusinessName        string `json:"business_name"`
	OwnerNumber         string `json:"owner_number"`
	Model               string `json:"model"`
	BusinessDescription string `json:"business_description"`

	// Mode: "receptionist" (default) or "personal" (reply in your voice, no lead funnel).
	Mode string `json:"mode"`
	// ReplyToGroups allows auto-reply in WhatsApp groups (default false).
	ReplyToGroups bool `json:"reply_to_groups"`
	// ReplyToSelfChat replies in WhatsApp "Message yourself" (notes-to-self). Default true.
	ReplyToSelfChat *bool `json:"reply_to_self_chat,omitempty"`
	// EnableLeadTracking runs qualification + lead_data (receptionist default: true).
	EnableLeadTracking *bool `json:"enable_lead_tracking,omitempty"`
	// EnableOwnerAlerts sends qualified-lead summary to owner_number (receptionist default: true).
	EnableOwnerAlerts *bool `json:"enable_owner_alerts,omitempty"`

	AllowedNumbers  []string   `json:"allowed_numbers,omitempty"`
	BlockedNumbers  []string   `json:"blocked_numbers,omitempty"`
	QuietHours      QuietHours `json:"quiet_hours"`
	DebounceSeconds int        `json:"debounce_seconds"`
	WebhookURL      string     `json:"webhook_url"`
	WebhookSecret   string     `json:"webhook_secret"`
	PauseHours      int           `json:"pause_hours,omitempty"` // human takeover TTL (default 24)
	FollowUpNudge   FollowUpNudge `json:"follow_up_nudge"`
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
		c.Model = "gemma4:31b-cloud"
	}
	c.OwnerNumber = NormalizePhone(c.OwnerNumber)
	c.AllowedNumbers = normalizePhoneList(c.AllowedNumbers)
	c.BlockedNumbers = normalizePhoneList(c.BlockedNumbers)
	if c.DebounceSeconds <= 0 {
		c.DebounceSeconds = 3
	}
	if c.PauseHours <= 0 {
		c.PauseHours = 24
	}
	if strings.TrimSpace(c.QuietHours.TZ) == "" {
		c.QuietHours.TZ = "Asia/Dhaka"
	}
	c.applyModeDefaults()
	if c.ReplyToSelfChat == nil {
		t := true
		c.ReplyToSelfChat = &t
	}
	return &c, nil
}

func (c *Config) SelfChatEnabled() bool {
	return c.ReplyToSelfChat != nil && *c.ReplyToSelfChat
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

func (c *Config) NudgeEnabled() bool {
	return c.FollowUpNudge.Enabled && c.LeadTrackingEnabled() && !c.IsPersonal()
}

func (c *Config) NudgeIdleHours() int {
	if c.FollowUpNudge.IdleHours <= 0 {
		return 24
	}
	return c.FollowUpNudge.IdleHours
}

func (c *Config) NudgeMessage() string {
	if m := strings.TrimSpace(c.FollowUpNudge.Message); m != "" {
		return m
	}
	return "Hi — just checking in. Still interested? Reply anytime and we can pick up where we left off."
}

func (c *Config) IsAllowed(sender string) bool {
	if len(c.AllowedNumbers) == 0 {
		return true
	}
	sender = NormalizePhone(sender)
	for _, n := range c.AllowedNumbers {
		if n == sender {
			return true
		}
	}
	return false
}

func (c *Config) IsBlocked(sender string) bool {
	sender = NormalizePhone(sender)
	for _, n := range c.BlockedNumbers {
		if n == sender {
			return true
		}
	}
	return false
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

func normalizePhoneList(list []string) []string {
	if len(list) == 0 {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, s := range list {
		if n := NormalizePhone(s); n != "" {
			out = append(out, n)
		}
	}
	return out
}
