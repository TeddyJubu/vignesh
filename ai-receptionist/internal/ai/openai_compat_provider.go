package ai

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

type OpenAICompatProvider struct {
	name    string
	client  openai.Client
	model   string
	baseURL string
}

func newOpenAICompatProvider(name, model, baseURL, apiKey string, headers map[string]string) (*OpenAICompatProvider, error) {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return nil, fmt.Errorf("%s api key is not set", strings.ToUpper(name))
	}
	if strings.TrimSpace(model) == "" {
		model = "gpt-4.1-mini"
	}
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("%s base_url is not set", strings.ToUpper(name))
	}

	opts := []option.RequestOption{
		option.WithAPIKey(key),
		option.WithBaseURL(baseURL),
		option.WithMaxRetries(2),
	}
	for k, v := range headers {
		if strings.TrimSpace(k) != "" && strings.TrimSpace(v) != "" {
			opts = append(opts, option.WithHeader(k, v))
		}
	}

	c := openai.NewClient(opts...)
	return &OpenAICompatProvider{name: name, client: c, model: model, baseURL: baseURL}, nil
}

func (p *OpenAICompatProvider) Name() string { return p.name }

func (p *OpenAICompatProvider) Ping(ctx context.Context) error {
	_, err := p.Complete(ctx, []ChatMessage{{Role: "user", Content: "Reply with exactly: ok"}}, false)
	return err
}

func (p *OpenAICompatProvider) Complete(ctx context.Context, messages []ChatMessage, jsonMode bool) (string, error) {
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
		return "", fmt.Errorf("%s returned empty message", strings.ToUpper(p.name))
	}
	return out, nil
}
