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

const openAIURL = "https://api.openai.com/v1/chat/completions"

type Client struct {
	model  string
	apiKey string
	http   *http.Client
}

func NewClient(model string) (*Client, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}
	if strings.TrimSpace(model) == "" {
		model = "gpt-4o-mini"
	}
	return &Client{
		model:  model,
		apiKey: key,
		http:   &http.Client{Timeout: 90 * time.Second},
	}, nil
}

type chatRequest struct {
	Model          string        `json:"model"`
	Messages       []ChatMessage `json:"messages"`
	Temperature    float64       `json:"temperature,omitempty"`
	ResponseFormat *struct {
		Type string `json:"type"`
	} `json:"response_format,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) Complete(ctx context.Context, messages []ChatMessage, jsonMode bool) (string, error) {
	reqBody := chatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: 0.4,
	}
	if jsonMode {
		reqBody.ResponseFormat = &struct {
			Type string `json:"type"`
		}{Type: "json_object"}
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
		time.Sleep(2 * time.Second)
		raw, status, err = c.postOnce(ctx, body)
		if err != nil {
			ops.AppendErrorLog("openai", err)
			return "", err
		}
	}
	if status < 200 || status >= 300 {
		apiErr := fmt.Errorf("OpenAI HTTP %d: %s", status, string(raw))
		ops.AppendErrorLog("openai", apiErr)
		return "", apiErr
	}

	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		ops.AppendErrorLog("openai", err)
		return "", err
	}
	if out.Error != nil && out.Error.Message != "" {
		apiErr := fmt.Errorf("OpenAI error: %s", out.Error.Message)
		ops.AppendErrorLog("openai", apiErr)
		return "", apiErr
	}
	if len(out.Choices) == 0 {
		apiErr := fmt.Errorf("OpenAI returned no choices")
		ops.AppendErrorLog("openai", apiErr)
		return "", apiErr
	}
	return out.Choices[0].Message.Content, nil
}

func shouldRetry(status int) bool {
	return status == 429 || status >= 500
}

func (c *Client) postOnce(ctx context.Context, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIURL, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		ops.AppendErrorLog("openai", err)
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
		{Role: "user", Content: "ping"},
	}, false)
	return err
}
