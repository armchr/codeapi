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

// OllamaLLM implements LLMService using Ollama
type OllamaLLM struct {
	apiURL string
	model  string
	logger *zap.Logger
	client *http.Client
}

// OllamaConfig holds configuration for Ollama LLM
type OllamaConfig struct {
	APIURL string // e.g., "http://localhost:11434"
	Model  string // e.g., "llama3.2", "codellama", "mistral"
}

// Common Ollama LLM models
const (
	Llama32     = "llama3.2"
	Llama31     = "llama3.1"
	CodeLlama   = "codellama"
	Mistral     = "mistral"
	DeepSeekR1  = "deepseek-r1"
	Qwen25Coder = "qwen2.5-coder"
)

// NewOllamaLLM creates a new Ollama LLM client
func NewOllamaLLM(config OllamaConfig, logger *zap.Logger) (*OllamaLLM, error) {
	if config.APIURL == "" {
		config.APIURL = "http://localhost:11434"
	}

	if config.Model == "" {
		config.Model = Llama32
	}

	return &OllamaLLM{
		apiURL: config.APIURL,
		model:  config.Model,
		logger: logger,
		client: &http.Client{
			Timeout: 120 * time.Second, // LLM generation can be slow
		},
	}, nil
}

// ollamaGenerateRequest represents the request body for Ollama generate API
type ollamaGenerateRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	System  string         `json:"system,omitempty"`
	Stream  bool           `json:"stream"`
	Options *ollamaOptions `json:"options,omitempty"`
}

type ollamaOptions struct {
	NumPredict  int     `json:"num_predict,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	TopK        int     `json:"top_k,omitempty"`
}

// ollamaGenerateResponse represents the response from Ollama generate API
type ollamaGenerateResponse struct {
	Model              string `json:"model"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context,omitempty"`
	TotalDuration      int64  `json:"total_duration,omitempty"`
	LoadDuration       int64  `json:"load_duration,omitempty"`
	PromptEvalCount    int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64  `json:"prompt_eval_duration,omitempty"`
	EvalCount          int    `json:"eval_count,omitempty"`
	EvalDuration       int64  `json:"eval_duration,omitempty"`
}

// Generate generates a response from the LLM
func (o *OllamaLLM) Generate(ctx context.Context, prompt string, opts GenerateOptions) (*GenerateResponse, error) {
	return o.GenerateWithSystem(ctx, "", prompt, opts)
}

// GenerateWithSystem generates a response with a system prompt
func (o *OllamaLLM) GenerateWithSystem(ctx context.Context, systemPrompt, userPrompt string, opts GenerateOptions) (*GenerateResponse, error) {
	if userPrompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	model := o.model
	if opts.Model != "" {
		model = opts.Model
	}

	reqBody := ollamaGenerateRequest{
		Model:  model,
		Prompt: userPrompt,
		System: systemPrompt,
		Stream: false,
		Options: &ollamaOptions{
			NumPredict:  opts.MaxTokens,
			Temperature: opts.Temperature,
			TopP:        opts.TopP,
			TopK:        opts.TopK,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.apiURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	o.logger.Debug("Sending request to Ollama",
		zap.String("model", model),
		zap.Int("prompt_length", len(userPrompt)),
		zap.Int("max_tokens", opts.MaxTokens))

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var genResp ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &GenerateResponse{
		Content:      genResp.Response,
		Model:        genResp.Model,
		PromptTokens: genResp.PromptEvalCount,
		OutputTokens: genResp.EvalCount,
		TotalTokens:  genResp.PromptEvalCount + genResp.EvalCount,
	}, nil
}

// Name returns the provider name
func (o *OllamaLLM) Name() string {
	return string(ProviderOllama)
}

// ModelName returns the model being used
func (o *OllamaLLM) ModelName() string {
	return o.model
}
