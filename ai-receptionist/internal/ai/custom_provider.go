package ai

type CustomProvider struct{ *OpenAICompatProvider }

func NewCustomProvider(model, baseURL, apiKey string) (*CustomProvider, error) {
	p, err := newOpenAICompatProvider(
		"custom",
		model,
		baseURL,
		apiKey,
		nil,
	)
	if err != nil {
		return nil, err
	}
	return &CustomProvider{OpenAICompatProvider: p}, nil
}
