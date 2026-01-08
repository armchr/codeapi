package llm

import (
	"context"
)

// LLMService defines the interface for LLM providers
type LLMService interface {
	// Generate generates a response from the LLM given a prompt
	Generate(ctx context.Context, prompt string, opts GenerateOptions) (*GenerateResponse, error)

	// GenerateWithSystem generates a response with a system prompt
	GenerateWithSystem(ctx context.Context, systemPrompt, userPrompt string, opts GenerateOptions) (*GenerateResponse, error)

	// Name returns the provider name
	Name() string

	// ModelName returns the model being used
	ModelName() string
}

// GenerateOptions contains options for LLM generation
type GenerateOptions struct {
	MaxTokens   int     // Maximum tokens to generate
	Temperature float64 // Temperature for sampling (0.0-1.0)
	Model       string  // Optional model override
	TopP        float64 // Top-p sampling
	TopK        int     // Top-k sampling
}

// DefaultGenerateOptions returns sensible defaults for code summarization
func DefaultGenerateOptions() GenerateOptions {
	return GenerateOptions{
		MaxTokens:   500,
		Temperature: 0.3,
		TopP:        0.9,
	}
}

// GenerateResponse contains the LLM response
type GenerateResponse struct {
	Content      string // Generated text
	Model        string // Model used
	PromptTokens int    // Tokens in prompt
	OutputTokens int    // Tokens in response
	TotalTokens  int    // Total tokens used
}

// Provider represents the LLM provider type
type Provider string

const (
	ProviderOllama  Provider = "ollama"
	ProviderClaude  Provider = "claude"
	ProviderOpenAI  Provider = "openai"
)

// Config holds configuration for LLM providers
type Config struct {
	Provider    Provider `yaml:"provider"`
	Model       string   `yaml:"model"`
	MaxTokens   int      `yaml:"max_tokens"`
	Temperature float64  `yaml:"temperature"`

	// Ollama-specific
	OllamaURL string `yaml:"ollama_url"`

	// Claude-specific
	ClaudeAPIKey string `yaml:"claude_api_key"`

	// OpenAI-specific
	OpenAIAPIKey  string `yaml:"openai_api_key"`
	OpenAIBaseURL string `yaml:"openai_base_url"` // For API-compatible services
}

// DefaultConfig returns a default configuration using Ollama
func DefaultConfig() Config {
	return Config{
		Provider:    ProviderOllama,
		Model:       "llama3.2",
		MaxTokens:   500,
		Temperature: 0.3,
		OllamaURL:   "http://localhost:11434",
	}
}
