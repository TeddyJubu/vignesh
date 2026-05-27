package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"ai-receptionist/internal/ops"
)

const (
	defaultOllamaChatURL = "https://ollama.com/api/chat"
	defaultModel         = "gemma4:31b-cloud"
)

type Client struct {
	model  string
	apiKey string
	apiURL string
	http   *http.Client
}

func NewClient(model string) (*Client, error) {
	key := strings.TrimSpace(os.Getenv("OLLAMA_API_KEY"))
	if key == "" {
		return nil, fmt.Errorf("OLLAMA_API_KEY is not set (create one at https://ollama.com/settings/keys)")
	}
	url := strings.TrimSpace(os.Getenv("OLLAMA_API_URL"))
	return NewClientWithConfig(model, key, url)
}

func NewClientWithConfig(model, apiKey, apiURL string) (*Client, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("OLLAMA_API_KEY is not set (create one at https://ollama.com/settings/keys)")
	}
	if strings.TrimSpace(model) == "" {
		model = defaultModel
	}
	apiURL = strings.TrimSpace(apiURL)
	if apiURL == "" {
		apiURL = defaultOllamaChatURL
	}
	return &Client{
		model:  model,
		apiKey: apiKey,
		apiURL: apiURL,
		http:   &http.Client{Timeout: 180 * time.Second},
	}, nil
}

func (c *Client) Name() string {
	return "ollama"
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Format   any           `json:"format,omitempty"`
	Options  *chatOptions  `json:"options,omitempty"`
}

type chatOptions struct {
	Temperature float64 `json:"temperature"`
}

type chatResponse struct {
	Message struct {
		Role     string `json:"role"`
		Content  string `json:"content"`
		Thinking string `json:"thinking"`
	} `json:"message"`
	Error string `json:"error,omitempty"`
}

func receptionistJSONFormat() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"reply":        map[string]any{"type": "string"},
			"lead_updates": map[string]any{"type": "object"},
			"qualified":    map[string]any{"type": "boolean"},
			"summary":      map[string]any{"type": "string"},
		},
		"required": []string{"reply"},
	}
}

func (c *Client) Complete(ctx context.Context, messages []ChatMessage, jsonMode bool) (string, error) {
	reqBody := chatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
		Options:  &chatOptions{Temperature: 0.4},
	}
	if jsonMode {
		reqBody.Format = receptionistJSONFormat()
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	raw, status, err := c.postOnce(ctx, body)
	if err != nil {
		return "", err
	}
	if shouldRetry(status) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(2 * time.Second):
		}
		raw, status, err = c.postOnce(ctx, body)
		if err != nil {
			ops.AppendErrorLog("ollama", err)
			return "", err
		}
	}
	if status < 200 || status >= 300 {
		apiErr := fmt.Errorf("Ollama HTTP %d: %s", status, string(raw))
		ops.AppendErrorLog("ollama", apiErr)
		return "", apiErr
	}

	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		// Some error bodies are {"error":"..."} without message
		var errWrap struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(raw, &errWrap) == nil && errWrap.Error != "" {
			apiErr := fmt.Errorf("Ollama error: %s", errWrap.Error)
			ops.AppendErrorLog("ollama", apiErr)
			return "", apiErr
		}
		ops.AppendErrorLog("ollama", err)
		return "", err
	}
	if out.Error != "" {
		apiErr := fmt.Errorf("Ollama error: %s", out.Error)
		ops.AppendErrorLog("ollama", apiErr)
		return "", apiErr
	}
	content := strings.TrimSpace(out.Message.Content)
	if content == "" {
		content = strings.TrimSpace(out.Message.Thinking)
	}
	if content == "" {
		apiErr := fmt.Errorf("Ollama returned empty message")
		ops.AppendErrorLog("ollama", apiErr)
		return "", apiErr
	}
	return content, nil
}

func shouldRetry(status int) bool {
	return status == 429 || status >= 500
}

func (c *Client) postOnce(ctx context.Context, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		ops.AppendErrorLog("ollama", err)
		return nil, 0, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return raw, resp.StatusCode, nil
}

// Ping verifies the API key with a minimal completion.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.Complete(ctx, []ChatMessage{
		{Role: "user", Content: "Reply with exactly: ok"},
	}, false)
	return err
}
