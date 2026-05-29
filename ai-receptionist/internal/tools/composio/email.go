package composio

import (
	"context"
	"fmt"
	"strings"
)

// EmailService sends mail via Composio Gmail tools.
type EmailService struct {
	client    *Client
	accountID string
	userID    string
}

func NewEmailService(client *Client, cfg Config) (*EmailService, error) {
	if client == nil || !client.Configured() {
		return nil, fmt.Errorf("composio client not configured")
	}
	if strings.TrimSpace(cfg.GmailAccountID) == "" {
		return nil, fmt.Errorf("composio gmail connected account not set")
	}
	return &EmailService{
		client:    client,
		accountID: cfg.GmailAccountID,
		userID:    cfg.UserID,
	}, nil
}

func (s *EmailService) SendEmail(ctx context.Context, to, subject, body string) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("recipient email required")
	}
	text := fmt.Sprintf("Send an email to %s with subject %q and body:\n%s", to, subject, body)
	res, err := s.client.Execute(ctx, "GMAIL_SEND_EMAIL", ExecuteRequest{
		ConnectedAccountID: s.accountID,
		UserID:             s.userID,
		Text:               text,
	})
	if err != nil {
		return err
	}
	if !res.Successful {
		if res.Error != "" {
			return fmt.Errorf("gmail send failed: %s", res.Error)
		}
		return fmt.Errorf("gmail send failed")
	}
	return nil
}

// Mailer is the interface injected into agent tools.
type Mailer interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}
