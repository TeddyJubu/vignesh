package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ai-receptionist/internal/lead"
)

type QualifyPayload struct {
	Event         string            `json:"event"`
	BusinessName  string            `json:"business_name"`
	Phone         string            `json:"phone"`
	Sender        string            `json:"sender"`
	LeadData      map[string]string `json:"lead_data"`
	LeadScore     string            `json:"lead_score"`
	Summary       string            `json:"summary"`
	QualifiedAt   string            `json:"qualified_at"`
}

func NotifyQualify(ctx context.Context, url, secret, businessName, convID, sender string, leadData map[string]string, summary string) error {
	if url == "" {
		return nil
	}
	payload := QualifyPayload{
		Event:        "lead.qualified",
		BusinessName: businessName,
		Phone:        convID,
		Sender:       sender,
		LeadData:     leadData,
		LeadScore:    lead.Score(leadData),
		Summary:      summary,
		QualifiedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		req.Header.Set("X-Webhook-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook HTTP %d", resp.StatusCode)
	}
	return nil
}
