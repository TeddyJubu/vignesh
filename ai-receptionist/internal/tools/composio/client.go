package composio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultBaseURL = "https://backend.composio.dev/api/v3.1"

// Client calls the Composio REST API.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

func New(apiKey string) *Client {
	return &Client{
		apiKey:  strings.TrimSpace(apiKey),
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.apiKey != ""
}

type ExecuteRequest struct {
	ConnectedAccountID string
	UserID             string
	Arguments          map[string]any
	Text               string
}

type ExecuteResult struct {
	Successful bool
	Data       map[string]any
	Error      string
	Raw        json.RawMessage
}

type ConnectedAccount struct {
	ID           string
	ToolkitSlug  string
	Status       string
	UserID       string
}

type accountsListResponse struct {
	Items []struct {
		ID      string `json:"id"`
		UserID  string `json:"user_id"`
		Status  string `json:"status"`
		Toolkit struct {
			Slug string `json:"slug"`
		} `json:"toolkit"`
	} `json:"items"`
}

// Status validates API key presence and optionally pings Composio.
func (c *Client) Status(ctx context.Context, verify bool) (map[string]any, error) {
	out := map[string]any{
		"configured": c.Configured(),
	}
	if !verify || !c.Configured() {
		return out, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/auth/session/info", nil)
	if err != nil {
		return out, err
	}
	req.Header.Set("x-api-key", c.apiKey)
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

// ListConnectedAccounts returns connected accounts. When activeOnly is true, only ACTIVE accounts are returned.
func (c *Client) ListConnectedAccounts(ctx context.Context, toolkitSlug string, activeOnly bool) ([]ConnectedAccount, error) {
	if !c.Configured() {
		return nil, fmt.Errorf("composio not configured")
	}
	q := url.Values{}
	q.Set("limit", "50")
	if activeOnly {
		q.Set("statuses", "ACTIVE")
	}
	if s := strings.TrimSpace(toolkitSlug); s != "" {
		q.Set("toolkit_slugs", normalizeToolkitSlug(s))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/connected_accounts?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("composio list accounts HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed accountsListResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	out := make([]ConnectedAccount, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		out = append(out, ConnectedAccount{
			ID:          item.ID,
			ToolkitSlug: item.Toolkit.Slug,
			Status:      item.Status,
			UserID:      item.UserID,
		})
	}
	return out, nil
}

// ResolveConnectedAccount returns explicit ID or first ACTIVE account for toolkit.
func (c *Client) ResolveConnectedAccount(ctx context.Context, toolkitSlug, explicitID string) (string, error) {
	id, _, err := c.ResolveConnectedAccountWithUser(ctx, toolkitSlug, explicitID)
	return id, err
}

// ResolveConnectedAccountWithUser returns account ID and Composio user ID for tool execution.
func (c *Client) ResolveConnectedAccountWithUser(ctx context.Context, toolkitSlug, explicitID string) (accountID, userID string, err error) {
	if id := strings.TrimSpace(explicitID); id != "" {
		return id, "", nil
	}
	slug := normalizeToolkitSlug(toolkitSlug)
	accounts, err := c.ListConnectedAccounts(ctx, slug, true)
	if err != nil {
		return "", "", err
	}
	for _, acct := range accounts {
		if strings.EqualFold(acct.Status, "ACTIVE") {
			return acct.ID, acct.UserID, nil
		}
	}
	return "", "", fmt.Errorf("no active Composio connected account for toolkit %q", slug)
}

func normalizeToolkitSlug(slug string) string {
	s := strings.ToLower(strings.TrimSpace(slug))
	switch s {
	case "calendar", "googlecalendar", "google_calendar":
		return "googlecalendar"
	case "gmail", "googlemail":
		return "gmail"
	default:
		return s
	}
}

// Execute runs a Composio tool by slug.
func (c *Client) Execute(ctx context.Context, toolSlug string, req ExecuteRequest) (*ExecuteResult, error) {
	if !c.Configured() {
		return nil, fmt.Errorf("composio not configured")
	}
	toolSlug = strings.TrimSpace(toolSlug)
	if toolSlug == "" {
		return nil, fmt.Errorf("tool slug required")
	}
	payload := map[string]any{}
	if id := strings.TrimSpace(req.ConnectedAccountID); id != "" {
		payload["connected_account_id"] = id
	}
	if uid := strings.TrimSpace(req.UserID); uid != "" {
		payload["user_id"] = uid
	}
	if text := strings.TrimSpace(req.Text); text != "" {
		payload["text"] = text
	} else if len(req.Arguments) > 0 {
		payload["arguments"] = req.Arguments
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	url := c.baseURL + "/tools/execute/" + url.PathEscape(toolSlug)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("composio execute %s HTTP %d: %s", toolSlug, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var parsed struct {
		Successful bool           `json:"successful"`
		Data       map[string]any `json:"data"`
		Error      *string        `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode composio response: %w", err)
	}
	res := &ExecuteResult{
		Successful: parsed.Successful,
		Data:       parsed.Data,
		Raw:        json.RawMessage(raw),
	}
	if parsed.Error != nil {
		res.Error = strings.TrimSpace(*parsed.Error)
	}
	return res, nil
}

// ResolveConfig fills missing connected account IDs by querying Composio.
func (c *Client) ResolveConfig(ctx context.Context, cfg Config) Config {
	out := cfg
	if !out.Configured() {
		return out
	}
	if out.CalendarAccountID == "" {
		if id, uid, err := c.ResolveConnectedAccountWithUser(ctx, "googlecalendar", ""); err == nil {
			out.CalendarAccountID = id
			out.UserID = preferComposioUserID(out.UserID, uid)
		}
	}
	if out.GmailAccountID == "" {
		if id, uid, err := c.ResolveConnectedAccountWithUser(ctx, "gmail", ""); err == nil {
			out.GmailAccountID = id
			out.UserID = preferComposioUserID(out.UserID, uid)
		}
	}
	if uid := strings.TrimSpace(out.UserID); uid == "" || uid == "default" {
		if found := c.userIDForAccount(ctx, out.CalendarAccountID); found != "" {
			out.UserID = found
		} else if found := c.userIDForAccount(ctx, out.GmailAccountID); found != "" {
			out.UserID = found
		}
	}
	return out
}

func (c *Client) userIDForAccount(ctx context.Context, accountID string) string {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return ""
	}
	accounts, err := c.ListConnectedAccounts(ctx, "", false)
	if err != nil {
		return ""
	}
	for _, acct := range accounts {
		if acct.ID == accountID && strings.TrimSpace(acct.UserID) != "" {
			return acct.UserID
		}
	}
	return ""
}

func preferComposioUserID(configured, fromAccount string) string {
	configured = strings.TrimSpace(configured)
	fromAccount = strings.TrimSpace(fromAccount)
	if configured == "" || configured == "default" {
		if fromAccount != "" {
			return fromAccount
		}
		return "default"
	}
	return configured
}
