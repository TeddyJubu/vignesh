package composio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	apiKey string
	http   *http.Client
}

func New(apiKey string) *Client {
	return &Client{
		apiKey: strings.TrimSpace(apiKey),
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.apiKey != ""
}

// Status is intentionally minimal: it validates only presence of a key and can optionally
// try a lightweight authenticated request if verify=true.
func (c *Client) Status(ctx context.Context, verify bool) (map[string]any, error) {
	out := map[string]any{
		"configured": c.Configured(),
	}
	if !verify || !c.Configured() {
		return out, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://backend.composio.dev/api/v1/me", nil)
	if err != nil {
		return out, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		out["verify_error"] = err.Error()
		return out, nil
	}
	defer resp.Body.Close()
	out["verify_status"] = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var payload any
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		out["me"] = payload
		return out, nil
	}
	out["verify_error"] = fmt.Sprintf("HTTP %d", resp.StatusCode)
	return out, nil
}
