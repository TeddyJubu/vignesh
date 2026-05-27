package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ai-receptionist/internal/ops"
)

const defaultAnthropicModel = "claude-3-5-sonnet-latest"

type AnthropicProvider struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewAnthropicProvider(model, apiKey string) (*AnthropicProvider, error) {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is not set")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultAnthropicModel
	}
	return &AnthropicProvider{
		apiKey: key,
		model:  model,
		http:   &http.Client{Timeout: 180 * time.Second},
	}, nil
}

func (p *AnthropicProvider) Name() string { return "anthropic" }

func (p *AnthropicProvider) Ping(ctx context.Context) error {
	_, err := p.Complete(ctx, []ChatMessage{{Role: "user", Content: "Reply with exactly: ok"}}, false)
	return err
}

type anthropicReq struct {
	Model     string         `json:"model"`
	MaxTokens int            `json:"max_tokens"`
	Messages  []anthropicMsg `json:"messages"`
	System    string         `json:"system,omitempty"`
}

type anthropicMsg struct {
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicResp struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *AnthropicProvider) Complete(ctx context.Context, messages []ChatMessage, jsonMode bool) (string, error) {
	// Keep the implementation minimal: we flatten into a single user message
	// while preserving role boundaries in text.
	text := flattenMessages(messages)
	if jsonMode {
		text = text + "\n\nReturn ONLY a single JSON object matching: " +
			`{"reply":"string","lead_updates":{},"qualified":false,"summary":"string"}` +
			"\nDo not wrap in markdown fences."
	}
	reqBody := anthropicReq{
		Model:     p.model,
		MaxTokens: 1024,
		Messages: []anthropicMsg{
			{Role: "user", Content: []anthropicContent{{Type: "text", Text: text}}},
		},
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.http.Do(req)
	if err != nil {
		ops.AppendErrorLog("anthropic", err)
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := fmt.Errorf("Anthropic HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		ops.AppendErrorLog("anthropic", apiErr)
		return "", apiErr
	}
	var out anthropicResp
	if err := json.Unmarshal(body, &out); err != nil {
		ops.AppendErrorLog("anthropic", err)
		return "", err
	}
	if out.Error != nil && strings.TrimSpace(out.Error.Message) != "" {
		apiErr := fmt.Errorf("Anthropic error: %s", out.Error.Message)
		ops.AppendErrorLog("anthropic", apiErr)
		return "", apiErr
	}
	var b strings.Builder
	for _, c := range out.Content {
		if c.Type == "text" && c.Text != "" {
			b.WriteString(c.Text)
		}
	}
	s := strings.TrimSpace(b.String())
	if s == "" {
		return "", fmt.Errorf("Anthropic returned empty message")
	}
	return s, nil
}

func flattenMessages(messages []ChatMessage) string {
	var b strings.Builder
	for _, m := range messages {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		if role == "" {
			role = "user"
		}
		b.WriteString("[" + role + "] ")
		b.WriteString(strings.TrimSpace(m.Content))
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}
