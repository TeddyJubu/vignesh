package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const openRouterURL = "https://openrouter.ai/api/v1/chat/completions"

type Client struct {
	provider string
	model    string
	apiKey   string
	http     *http.Client
}

func NewClient(provider, model string) (*Client, error) {
	key := ""
	switch provider {
	case "openrouter", "":
		key = os.Getenv("OPENROUTER_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("OPENROUTER_API_KEY is not set")
		}
		provider = "openrouter"
	case "openai":
		key = os.Getenv("OPENAI_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY is not set")
		}
	default:
		return nil, fmt.Errorf("unsupported ai_provider %q", provider)
	}
	return &Client{
		provider: provider,
		model:    model,
		apiKey:   key,
		http:     &http.Client{Timeout: 90 * time.Second},
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

	url := openRouterURL
	if c.provider == "openai" {
		url = "https://api.openai.com/v1/chat/completions"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if c.provider == "openrouter" {
		req.Header.Set("HTTP-Referer", "https://github.com/ai-receptionist")
		req.Header.Set("X-Title", "AI Receptionist")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("AI API HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.Error != nil && out.Error.Message != "" {
		return "", fmt.Errorf("AI API error: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("AI API returned no choices")
	}
	return out.Choices[0].Message.Content, nil
}
