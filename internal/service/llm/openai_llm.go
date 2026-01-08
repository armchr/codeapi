package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// OpenAILLM implements LLMService using OpenAI's API
type OpenAILLM struct {
	apiKey  string
	model   string
	baseURL string
	logger  *zap.Logger
	client  *http.Client
}

// OpenAIConfig holds configuration for OpenAI LLM
type OpenAIConfig struct {
	APIKey  string // OpenAI API key
	Model   string // e.g., "gpt-4o", "gpt-4o-mini"
	BaseURL string // Optional custom base URL (for compatible APIs)
}

// OpenAI model constants
const (
	GPT4o           = "gpt-4o"
	GPT4oMini       = "gpt-4o-mini"
	GPT4Turbo       = "gpt-4-turbo"
	OpenAIDefaultURL = "https://api.openai.com"
)

// NewOpenAILLM creates a new OpenAI LLM client
func NewOpenAILLM(config OpenAIConfig, logger *zap.Logger) (*OpenAILLM, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	if config.Model == "" {
		config.Model = GPT4oMini // Default to cost-effective model
	}

	if config.BaseURL == "" {
		config.BaseURL = OpenAIDefaultURL
	}

	return &OpenAILLM{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: config.BaseURL,
		logger:  logger,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

// openaiMessage represents a message in the OpenAI API
type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openaiRequest represents the request body for OpenAI API
type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
}

// openaiResponse represents the response from OpenAI API
type openaiResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []openaiChoice `json:"choices"`
	Usage   openaiUsage `json:"usage"`
}

type openaiChoice struct {
	Index        int           `json:"index"`
	Message      openaiMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openaiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Generate generates a response from OpenAI
func (o *OpenAILLM) Generate(ctx context.Context, prompt string, opts GenerateOptions) (*GenerateResponse, error) {
	return o.GenerateWithSystem(ctx, "", prompt, opts)
}

// GenerateWithSystem generates a response with a system prompt
func (o *OpenAILLM) GenerateWithSystem(ctx context.Context, systemPrompt, userPrompt string, opts GenerateOptions) (*GenerateResponse, error) {
	if userPrompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	model := o.model
	if opts.Model != "" {
		model = opts.Model
	}

	maxTokens := opts.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 500
	}

	messages := make([]openaiMessage, 0, 2)
	if systemPrompt != "" {
		messages = append(messages, openaiMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}
	messages = append(messages, openaiMessage{
		Role:    "user",
		Content: userPrompt,
	})

	reqBody := openaiRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: opts.Temperature,
		TopP:        opts.TopP,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	o.logger.Debug("Sending request to OpenAI",
		zap.String("model", model),
		zap.Int("prompt_length", len(userPrompt)),
		zap.Int("max_tokens", maxTokens))

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp openaiErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("OpenAI API error (%s): %s", errResp.Error.Type, errResp.Error.Message)
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var genResp openaiResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(genResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &GenerateResponse{
		Content:      genResp.Choices[0].Message.Content,
		Model:        genResp.Model,
		PromptTokens: genResp.Usage.PromptTokens,
		OutputTokens: genResp.Usage.CompletionTokens,
		TotalTokens:  genResp.Usage.TotalTokens,
	}, nil
}

// Name returns the provider name
func (o *OpenAILLM) Name() string {
	return string(ProviderOpenAI)
}

// ModelName returns the model being used
func (o *OpenAILLM) ModelName() string {
	return o.model
}
