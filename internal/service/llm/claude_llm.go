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

// ClaudeLLM implements LLMService using Anthropic's Claude API
type ClaudeLLM struct {
	apiKey  string
	model   string
	baseURL string
	logger  *zap.Logger
	client  *http.Client
}

// ClaudeConfig holds configuration for Claude LLM
type ClaudeConfig struct {
	APIKey  string // Anthropic API key
	Model   string // e.g., "claude-sonnet-4-20250514", "claude-3-5-haiku-20241022"
	BaseURL string // Optional custom base URL
}

// Claude model constants
const (
	ClaudeSonnet4     = "claude-sonnet-4-20250514"
	Claude35Haiku     = "claude-3-5-haiku-20241022"
	Claude35Sonnet    = "claude-3-5-sonnet-20241022"
	ClaudeDefaultURL  = "https://api.anthropic.com"
	ClaudeAPIVersion  = "2023-06-01"
)

// NewClaudeLLM creates a new Claude LLM client
func NewClaudeLLM(config ClaudeConfig, logger *zap.Logger) (*ClaudeLLM, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Claude API key is required")
	}

	if config.Model == "" {
		config.Model = Claude35Haiku // Default to cost-effective model
	}

	if config.BaseURL == "" {
		config.BaseURL = ClaudeDefaultURL
	}

	return &ClaudeLLM{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: config.BaseURL,
		logger:  logger,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

// claudeMessage represents a message in the Claude API
type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// claudeRequest represents the request body for Claude API
type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system,omitempty"`
	Messages  []claudeMessage `json:"messages"`
}

// claudeResponse represents the response from Claude API
type claudeResponse struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Role         string `json:"role"`
	Content      []claudeContentBlock `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence,omitempty"`
	Usage        claudeUsage `json:"usage"`
}

type claudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type claudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type claudeErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Generate generates a response from Claude
func (c *ClaudeLLM) Generate(ctx context.Context, prompt string, opts GenerateOptions) (*GenerateResponse, error) {
	return c.GenerateWithSystem(ctx, "", prompt, opts)
}

// GenerateWithSystem generates a response with a system prompt
func (c *ClaudeLLM) GenerateWithSystem(ctx context.Context, systemPrompt, userPrompt string, opts GenerateOptions) (*GenerateResponse, error) {
	if userPrompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	model := c.model
	if opts.Model != "" {
		model = opts.Model
	}

	maxTokens := opts.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 500
	}

	reqBody := claudeRequest{
		Model:     model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages: []claudeMessage{
			{Role: "user", Content: userPrompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", ClaudeAPIVersion)

	c.logger.Debug("Sending request to Claude",
		zap.String("model", model),
		zap.Int("prompt_length", len(userPrompt)),
		zap.Int("max_tokens", maxTokens))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp claudeErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("Claude API error (%s): %s", errResp.Error.Type, errResp.Error.Message)
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var genResp claudeResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract text from content blocks
	var content string
	for _, block := range genResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &GenerateResponse{
		Content:      content,
		Model:        genResp.Model,
		PromptTokens: genResp.Usage.InputTokens,
		OutputTokens: genResp.Usage.OutputTokens,
		TotalTokens:  genResp.Usage.InputTokens + genResp.Usage.OutputTokens,
	}, nil
}

// Name returns the provider name
func (c *ClaudeLLM) Name() string {
	return string(ProviderClaude)
}

// ModelName returns the model being used
func (c *ClaudeLLM) ModelName() string {
	return c.model
}
