package ai

import (
	"context"
	"fmt"
	"os"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

const defaultOpenAIBaseURL = "https://sg.api.openai.com"

type OpenAIProvider struct {
	client  openai.Client
	model   string
	baseURL string
}

func NewOpenAIProvider(model, baseURL string) (*OpenAIProvider, error) {
	key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if key == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}
	return NewOpenAIProviderWithKey(model, baseURL, key)
}

func NewOpenAIProviderWithKey(model, baseURL, apiKey string) (*OpenAIProvider, error) {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}
	if strings.TrimSpace(model) == "" {
		model = "gpt-4.1-mini"
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultOpenAIBaseURL
	}
	c := openai.NewClient(
		option.WithAPIKey(key),
		option.WithBaseURL(baseURL),
		option.WithMaxRetries(2),
	)
	return &OpenAIProvider{client: c, model: model, baseURL: baseURL}, nil
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) Ping(ctx context.Context) error {
	_, err := p.Complete(ctx, []ChatMessage{{Role: "user", Content: "Reply with exactly: ok"}}, false)
	return err
}

func (p *OpenAIProvider) Complete(ctx context.Context, messages []ChatMessage, jsonMode bool) (string, error) {
	items := openAIInputItems(messages)
	params := responses.ResponseNewParams{
		Model: openai.ChatModel(p.model),
		Input: responses.ResponseNewParamsInputUnion{OfInputItemList: items},
	}
	if jsonMode {
		params.Instructions = openai.String(
			"Return ONLY a single JSON object matching: " +
				`{"reply":"string","lead_updates":{},"qualified":false,"summary":"string"}` +
				"\nDo not wrap in markdown fences.",
		)
	}
	stream := p.client.Responses.NewStreaming(ctx, params)
	var b strings.Builder
	for stream.Next() {
		ev := stream.Current()
		if ev.Delta != "" {
			b.WriteString(ev.Delta)
		}
	}
	if err := stream.Err(); err != nil {
		return "", err
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "", fmt.Errorf("OpenAI returned empty message")
	}
	return out, nil
}

func openAIInputItems(messages []ChatMessage) responses.ResponseInputParam {
	items := make(responses.ResponseInputParam, 0, len(messages))
	for _, m := range messages {
		role := mapOpenAIRole(m.Role)
		items = append(items, responses.ResponseInputItemParamOfMessage(m.Content, role))
	}
	return items
}

func mapOpenAIRole(role string) responses.EasyInputMessageRole {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "system":
		return responses.EasyInputMessageRoleSystem
	case "assistant":
		return responses.EasyInputMessageRoleAssistant
	default:
		return responses.EasyInputMessageRoleUser
	}
}
