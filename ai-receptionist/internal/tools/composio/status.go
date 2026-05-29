package composio

import (
	"context"
	"strings"
)

// ConnectedAccountSummary is a safe API view of a Composio connected account.
type ConnectedAccountSummary struct {
	ID          string `json:"id"`
	ToolkitSlug string `json:"toolkit_slug"`
	Status      string `json:"status"`
	UserID      string `json:"user_id,omitempty"`
}

// IntegrationStatus describes Composio readiness for calendar and Gmail.
type IntegrationStatus struct {
	Configured        bool                      `json:"configured"`
	CalendarReady     bool                      `json:"calendar_ready"`
	GmailReady        bool                      `json:"gmail_ready"`
	UserID            string                    `json:"user_id,omitempty"`
	Timezone          string                    `json:"timezone,omitempty"`
	CalendarAccount   string                    `json:"calendar_account_id,omitempty"`
	GmailAccount      string                    `json:"gmail_account_id,omitempty"`
	ConnectedAccounts []ConnectedAccountSummary `json:"connected_accounts,omitempty"`
	ExpiredAccounts   int                       `json:"expired_accounts,omitempty"`
	NeedsReauth       bool                      `json:"needs_reauth,omitempty"`
	VerifyError       string                    `json:"verify_error,omitempty"`
}

// BuildIntegrationStatus loads config, optionally verifies the API key, and resolves connected accounts.
func BuildIntegrationStatus(ctx context.Context, r settingsReader, verify bool) (IntegrationStatus, error) {
	cfg, err := LoadConfig(r)
	if err != nil {
		return IntegrationStatus{}, err
	}
	out := IntegrationStatus{
		Configured: cfg.Configured(),
		UserID:     cfg.UserID,
		Timezone:   cfg.Timezone,
	}
	if !cfg.Configured() {
		return out, nil
	}
	client := New(cfg.APIKey)
	if verify {
		raw, _ := client.Status(ctx, true)
		if errMsg, ok := raw["verify_error"].(string); ok && errMsg != "" {
			out.VerifyError = errMsg
			return out, nil
		}
		verified := false
		if n, ok := raw["verify_status"].(int); ok {
			verified = n >= 200 && n < 300
		}
		if !verified {
			out.VerifyError = "Composio verification failed"
			return out, nil
		}
	}
	cfg = client.ResolveConfig(ctx, cfg)
	out.UserID = cfg.UserID
	out.CalendarAccount = cfg.CalendarAccountID
	out.GmailAccount = cfg.GmailAccountID
	out.CalendarReady = cfg.CalendarReady()
	out.GmailReady = cfg.GmailReady()

	accounts, err := client.ListConnectedAccounts(ctx, "", false)
	if err == nil {
		out.ConnectedAccounts = make([]ConnectedAccountSummary, 0, len(accounts))
		activeCalendar, activeGmail := 0, 0
		for _, acct := range accounts {
			out.ConnectedAccounts = append(out.ConnectedAccounts, ConnectedAccountSummary{
				ID:          acct.ID,
				ToolkitSlug: acct.ToolkitSlug,
				Status:      acct.Status,
				UserID:      acct.UserID,
			})
			st := strings.ToUpper(strings.TrimSpace(acct.Status))
			if st == "EXPIRED" || st == "REVOKED" {
				out.ExpiredAccounts++
			}
			if st != "ACTIVE" {
				continue
			}
			switch normalizeToolkitSlug(acct.ToolkitSlug) {
			case "googlecalendar":
				activeCalendar++
			case "gmail":
				activeGmail++
			}
		}
		if !out.CalendarReady && activeCalendar == 0 && out.ExpiredAccounts > 0 {
			out.NeedsReauth = true
		}
		if out.Configured && !out.CalendarReady && !out.GmailReady && out.ExpiredAccounts > 0 {
			out.NeedsReauth = true
		}
	}
	return out, nil
}
