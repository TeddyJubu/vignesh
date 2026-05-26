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
	if strings.TrimSpace(model) == "" {
		model = "gpt-4.1-mini"
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultOpenAIBaseURL
	}
	c := openai.NewClient(
		option.WithAPIKey(key),
		option.WithBaseURL(baseURL),
		// Keep retries modest; handler enforces strict ctx timeouts.
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
	input := openAIInputFromChat(messages, jsonMode)
	stream := p.client.Responses.NewStreaming(ctx, responses.ResponseNewParams{
		Model: openai.ChatModel(p.model),
		Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(input)},
	})
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

func openAIInputFromChat(messages []ChatMessage, jsonMode bool) string {
	// Keep minimal: pack into a single prompt string. This preserves existing system/user/assistant roles
	// without mapping into the Responses structured "input_items" schema.
	var b strings.Builder
	if jsonMode {
		b.WriteString("Return ONLY a single JSON object matching this schema:\n")
		b.WriteString(`{"reply":"string","lead_updates":{"key":"value"},"qualified":true,"summary":"string"}` + "\n")
		b.WriteString("Do not wrap in markdown fences.\n\n")
	}
	for _, m := range messages {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		switch role {
		case "system":
			b.WriteString("[system]\n")
		case "assistant":
			b.WriteString("[assistant]\n")
		default:
			b.WriteString("[user]\n")
		}
		b.WriteString(m.Content)
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String())
}

