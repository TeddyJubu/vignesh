package memory

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

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	baseURL = strings.TrimSpace(baseURL)
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

type IngestRequest struct {
	ConvID    string         `json:"conv_id"`
	Timestamp string         `json:"timestamp"`
	Role      string         `json:"role"`
	Text      string         `json:"text"`
	Meta      map[string]any `json:"meta,omitempty"`
}

type RecallItem struct {
	Text      string  `json:"text"`
	Score     float64 `json:"score"`
	Source    string  `json:"source"`
	CreatedAt string  `json:"created_at,omitempty"`
}

type RecallResponse struct {
	Items   []RecallItem `json:"items"`
	Snippet string       `json:"snippet"`
}

func (c *Client) Enabled() bool {
	return c != nil && strings.TrimSpace(c.baseURL) != ""
}

func (c *Client) BaseURL() string {
	if c == nil {
		return ""
	}
	return c.baseURL
}

func (c *Client) HTTPClient() *http.Client {
	if c == nil {
		return http.DefaultClient
	}
	return c.http
}

func (c *Client) Ingest(ctx context.Context, req IngestRequest) error {
	if !c.Enabled() {
		return fmt.Errorf("graphiti base url not set")
	}
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/ingest", bytes.NewReader(b))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("graphiti ingest status=%d body=%q", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) Recall(ctx context.Context, convID, q string, limit int) (*RecallResponse, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("graphiti base url not set")
	}
	u, err := url.Parse(c.baseURL + "/recall")
	if err != nil {
		return nil, err
	}
	qs := u.Query()
	qs.Set("conv_id", convID)
	qs.Set("q", q)
	if limit > 0 {
		qs.Set("limit", fmt.Sprintf("%d", limit))
	}
	u.RawQuery = qs.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("graphiti recall status=%d body=%q", resp.StatusCode, string(body))
	}
	var out RecallResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

