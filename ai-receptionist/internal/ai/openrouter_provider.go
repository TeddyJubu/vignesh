package ai

const openRouterBaseURL = "https://openrouter.ai/api/v1"

type OpenRouterProvider struct{ *OpenAICompatProvider }

func NewOpenRouterProvider(model, apiKey string) (*OpenRouterProvider, error) {
	p, err := newOpenAICompatProvider(
		"openrouter",
		model,
		openRouterBaseURL,
		apiKey,
		map[string]string{
			"HTTP-Referer": "http://localhost",
			"X-Title":      "ai-receptionist",
		},
	)
	if err != nil {
		return nil, err
	}
	return &OpenRouterProvider{OpenAICompatProvider: p}, nil
}
