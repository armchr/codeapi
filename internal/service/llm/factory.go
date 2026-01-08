package llm

import (
	"fmt"
	"os"

	"go.uber.org/zap"
)

// NewLLMService creates an LLM service based on the provided configuration
func NewLLMService(config Config, logger *zap.Logger) (LLMService, error) {
	switch config.Provider {
	case ProviderOllama:
		return NewOllamaLLM(OllamaConfig{
			APIURL: config.OllamaURL,
			Model:  config.Model,
		}, logger)

	case ProviderClaude:
		apiKey := config.ClaudeAPIKey
		if apiKey == "" {
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("Claude API key not provided (set claude_api_key in config or ANTHROPIC_API_KEY env var)")
		}
		return NewClaudeLLM(ClaudeConfig{
			APIKey: apiKey,
			Model:  config.Model,
		}, logger)

	case ProviderOpenAI:
		apiKey := config.OpenAIAPIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("OpenAI API key not provided (set openai_api_key in config or OPENAI_API_KEY env var)")
		}
		return NewOpenAILLM(OpenAIConfig{
			APIKey:  apiKey,
			Model:   config.Model,
			BaseURL: config.OpenAIBaseURL,
		}, logger)

	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", config.Provider)
	}
}

// NewLLMServiceFromProvider creates an LLM service from just a provider string
// This is a convenience function for simpler configuration
func NewLLMServiceFromProvider(provider string, logger *zap.Logger) (LLMService, error) {
	config := DefaultConfig()
	config.Provider = Provider(provider)

	// Set default models based on provider
	switch config.Provider {
	case ProviderOllama:
		config.Model = Llama32
	case ProviderClaude:
		config.Model = Claude35Haiku
	case ProviderOpenAI:
		config.Model = GPT4oMini
	}

	return NewLLMService(config, logger)
}
