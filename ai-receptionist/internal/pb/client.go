package pb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const defaultTimeout = 15 * time.Second

// Client talks to a PocketBase REST API. Empty base URL means disabled (all repo ops no-op).
type Client struct {
	baseURL string
	token   string
	http    *http.Client

	mu          sync.Mutex
	cachedToken string
}

func NewClient(baseURL, token string) *Client {
	baseURL = strings.TrimSpace(baseURL)
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL: baseURL,
		token:   strings.TrimSpace(token),
		http: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// NewFromEnv builds a client from POCKETBASE_URL and auth env vars.
func NewFromEnv() *Client {
	url := strings.TrimSpace(os.Getenv("POCKETBASE_URL"))
	token := strings.TrimSpace(os.Getenv("POCKETBASE_TOKEN"))
	return NewClient(url, token)
}

func (c *Client) Enabled() bool {
	return c != nil && strings.TrimSpace(c.baseURL) != "" && c.hasAuth()
}

func (c *Client) hasAuth() bool {
	if c == nil {
		return false
	}
	if strings.TrimSpace(c.token) != "" {
		return true
	}
	email := strings.TrimSpace(os.Getenv("POCKETBASE_ADMIN_EMAIL"))
	pass := os.Getenv("POCKETBASE_ADMIN_PASSWORD")
	return email != "" && pass != ""
}

func (c *Client) BaseURL() string {
	if c == nil {
		return ""
	}
	return c.baseURL
}

func (c *Client) Ping(ctx context.Context) error {
	if !c.Enabled() {
		return fmt.Errorf("pocketbase url not set")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/health", nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("pocketbase health status=%d body=%q", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) authToken(ctx context.Context) (string, error) {
	if c == nil {
		return "", fmt.Errorf("pocketbase client nil")
	}
	if t := strings.TrimSpace(c.token); t != "" {
		return t, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if strings.TrimSpace(c.cachedToken) != "" {
		return c.cachedToken, nil
	}
	email := strings.TrimSpace(os.Getenv("POCKETBASE_ADMIN_EMAIL"))
	pass := os.Getenv("POCKETBASE_ADMIN_PASSWORD")
	if email == "" || pass == "" {
		return "", fmt.Errorf("pocketbase: set POCKETBASE_TOKEN or admin email/password")
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := c.request(ctx, http.MethodPost, "/api/collections/_superusers/auth-with-password", map[string]string{
		"identity": email,
		"password": pass,
	}, &out, ""); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.Token) == "" {
		return "", fmt.Errorf("pocketbase auth: empty token")
	}
	c.cachedToken = out.Token
	return c.cachedToken, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	if !c.Enabled() {
		return fmt.Errorf("pocketbase url not set")
	}
	token, err := c.authToken(ctx)
	if err != nil {
		return err
	}
	return c.request(ctx, method, path, body, out, token)
}

func (c *Client) request(ctx context.Context, method, path string, body any, out any, bearer string) error {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("pocketbase %s %s status=%d body=%q", method, path, resp.StatusCode, string(raw))
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}

type listRecordsResponse struct {
	Items []recordItem `json:"items"`
}

type recordItem struct {
	ID      string `json:"id"`
	Created string `json:"created"`
	Updated string `json:"updated"`
}

func (c *Client) listRecords(ctx context.Context, collection, filter string) ([]recordItem, error) {
	path := fmt.Sprintf("/api/collections/%s/records?perPage=1", collection)
	if filter != "" {
		q := url.Values{}
		q.Set("filter", filter)
		path += "&" + q.Encode()
	}
	var out listRecordsResponse
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) createRecord(ctx context.Context, collection string, fields map[string]any) (string, error) {
	var out recordItem
	if err := c.doJSON(ctx, http.MethodPost, "/api/collections/"+collection+"/records", fields, &out); err != nil {
		return "", err
	}
	return out.ID, nil
}

func (c *Client) patchRecord(ctx context.Context, collection, id string, fields map[string]any) error {
	return c.doJSON(ctx, http.MethodPatch, "/api/collections/"+collection+"/records/"+id, fields, nil)
}

func escapeFilterString(s string) string {
	return strings.ReplaceAll(s, "'", "\\'")
}
