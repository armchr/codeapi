package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
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
	Thinking           string `json:"thinking,omitempty"` // qwen3 and other thinking models put reasoning here
	Done               bool   `json:"done"`
	DoneReason         string `json:"done_reason,omitempty"`
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

	// For qwen3 models, prepend /no_think to disable thinking mode and get direct answers
	finalPrompt := userPrompt
	if isThinkingModel(model) {
		finalPrompt = "/no_think " + userPrompt
	}

	reqBody := ollamaGenerateRequest{
		Model:  model,
		Prompt: finalPrompt,
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
	o.logger.Debug("Ollama request payload", zap.String("payload", string(jsonData)))
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

	// Read full response body for debugging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Log raw response for debugging qwen3 thinking mode
	o.logger.Debug("Raw Ollama response body",
		zap.String("model", model),
		zap.Int("body_length", len(bodyBytes)),
		zap.String("body_preview", truncateString(string(bodyBytes), 500)))

	var genResp ollamaGenerateResponse
	if err := json.Unmarshal(bodyBytes, &genResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// For thinking models like qwen3, the actual response may be empty
	// and the content is in the "thinking" field instead
	var content string
	if genResp.Response != "" {
		// Normal response - process for any embedded think tags
		content = extractThinkingContent(genResp.Response)
		content = cleanThinkingTags(content)
	} else if genResp.Thinking != "" {
		// Thinking model with empty response - extract useful content from thinking
		content = extractUsefulContent(genResp.Thinking)
		o.logger.Debug("Extracted content from thinking field",
			zap.String("model", model),
			zap.Int("thinking_length", len(genResp.Thinking)),
			zap.Int("extracted_length", len(content)))
	}

	o.logger.Debug("Processed Ollama response",
		zap.String("model", model),
		zap.Int("response_length", len(genResp.Response)),
		zap.Int("thinking_length", len(genResp.Thinking)),
		zap.Int("content_length", len(content)),
		zap.Int("output_tokens", genResp.EvalCount))

	return &GenerateResponse{
		Content:      content,
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

// extractThinkingContent handles models like qwen3 that use <think>...</think> tags.
// It extracts the response content, handling cases where:
// 1. Response is after </think> tag (normal case)
// 2. Response is empty but content exists within <think> tags (edge case)
func extractThinkingContent(response string) string {
	response = strings.TrimSpace(response)
	if response == "" {
		return ""
	}

	// Check if response contains thinking tags
	thinkEndTag := "</think>"
	if idx := strings.Index(response, thinkEndTag); idx != -1 {
		// Extract content after </think> tag
		afterThink := strings.TrimSpace(response[idx+len(thinkEndTag):])
		if afterThink != "" {
			return afterThink
		}

		// If nothing after </think>, extract from within <think> tags as fallback
		thinkStartTag := "<think>"
		if startIdx := strings.Index(response, thinkStartTag); startIdx != -1 {
			thinkContent := response[startIdx+len(thinkStartTag) : idx]
			return strings.TrimSpace(thinkContent)
		}
	}

	// Check for unclosed <think> tag (model still thinking)
	thinkStartTag := "<think>"
	if idx := strings.Index(response, thinkStartTag); idx != -1 {
		// Return content after <think> tag
		return strings.TrimSpace(response[idx+len(thinkStartTag):])
	}

	// No thinking tags, return as-is
	return response
}

// cleanThinkingTags removes any remaining <think> tags from the response
// while preserving the actual content
var thinkingTagRegex = regexp.MustCompile(`</?think>`)

func cleanThinkingTags(content string) string {
	return strings.TrimSpace(thinkingTagRegex.ReplaceAllString(content, ""))
}

// truncateString truncates a string to maxLen characters for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// isThinkingModel returns true if the model uses thinking mode by default
// These models need /no_think prefix to get direct answers
func isThinkingModel(model string) bool {
	model = strings.ToLower(model)
	// qwen3 models use thinking mode by default
	if strings.HasPrefix(model, "qwen3") {
		return true
	}
	// deepseek-r1 also uses reasoning mode
	if strings.Contains(model, "deepseek-r1") {
		return true
	}
	return false
}

// extractUsefulContent extracts useful summary content from thinking/reasoning text
// It looks for structured patterns and extracts the most useful information
func extractUsefulContent(thinking string) string {
	lines := strings.Split(thinking, "\n")
	var result []string

	// Patterns that indicate useful summary content
	summaryPrefixes := []string{
		"ONE_LINE:", "DESCRIPTION:", "SIDE_EFFECTS:",
		"Summary:", "Purpose:", "What it does:",
		"This function", "This class", "This file", "This method",
		"The function", "The class", "The file", "The method",
		"It ", "Returns ", "Handles ", "Processes ", "Retrieves ",
	}

	// Skip patterns - lines that are just reasoning/planning
	skipPatterns := []string{
		"We are given", "We need to", "Let me", "I'll", "First,", "Next,",
		"Steps:", "Step ", "- Step", "Now ", "Okay,", "Hmm,",
		"The user", "Let's", "I need to", "I should",
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip reasoning lines
		shouldSkip := false
		for _, skip := range skipPatterns {
			if strings.HasPrefix(line, skip) {
				shouldSkip = true
				break
			}
		}
		if shouldSkip {
			continue
		}

		// Include lines with summary prefixes
		for _, prefix := range summaryPrefixes {
			if strings.Contains(line, prefix) || strings.HasPrefix(line, prefix) {
				result = append(result, line)
				break
			}
		}
	}

	// If we found structured content, use it
	if len(result) > 0 {
		return strings.Join(result, "\n")
	}

	// Fallback: extract first few meaningful sentences
	var sentences []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip very short lines and bullet points
		if len(line) < 20 || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
			continue
		}
		// Skip reasoning lines
		shouldSkip := false
		for _, skip := range skipPatterns {
			if strings.HasPrefix(line, skip) {
				shouldSkip = true
				break
			}
		}
		if !shouldSkip {
			sentences = append(sentences, line)
			if len(sentences) >= 3 {
				break
			}
		}
	}

	if len(sentences) > 0 {
		return strings.Join(sentences, " ")
	}

	// Last resort: return trimmed thinking content up to reasonable length
	if len(thinking) > 500 {
		return strings.TrimSpace(thinking[:500]) + "..."
	}
	return strings.TrimSpace(thinking)
}
