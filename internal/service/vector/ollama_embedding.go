package vector

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

// OllamaEmbedding implements EmbeddingModel interface using Ollama
type OllamaEmbedding struct {
	apiURL    string
	model     string
	dimension int
	logger    *zap.Logger
	client    *http.Client
}

// OllamaEmbeddingConfig holds configuration for Ollama embedding model
type OllamaEmbeddingConfig struct {
	APIURL    string // e.g., "http://localhost:11434"
	APIKey    string // Optional API key for authentication
	Model     string // e.g., "nomic-embed-text", "all-minilm"
	Dimension int    // Dimension of the embedding vector
}

// Common Ollama embedding models
const (
	// NomicEmbedText is a high-quality 768-dimensional embedding model
	NomicEmbedText = "nomic-embed-text"

	// AllMiniLM is a lightweight 384-dimensional embedding model
	AllMiniLM = "all-minilm"

	// MxbaiEmbedLarge is a large 1024-dimensional embedding model
	MxbaiEmbedLarge = "mxbai-embed-large"
)

// Model dimensions mapping
var modelDimensions = map[string]int{
	NomicEmbedText:  768,
	AllMiniLM:       384,
	MxbaiEmbedLarge: 1024,
}

// NewOllamaEmbedding creates a new Ollama embedding model client
func NewOllamaEmbedding(config OllamaEmbeddingConfig, logger *zap.Logger) (*OllamaEmbedding, error) {
	if config.APIURL == "" {
		config.APIURL = "http://localhost:11434"
	}

	if config.Model == "" {
		config.Model = NomicEmbedText
	}

	// Set dimension from known models or use provided dimension
	dimension := config.Dimension
	if dimension == 0 {
		if knownDim, ok := modelDimensions[config.Model]; ok {
			dimension = knownDim
		} else {
			dimension = 768 // Default dimension
		}
	}

	return &OllamaEmbedding{
		apiURL:    config.APIURL,
		model:     config.Model,
		dimension: dimension,
		logger:    logger,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// ollamaEmbeddingRequest represents the request body for Ollama embedding API
type ollamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// ollamaEmbeddingResponse represents the response from Ollama embedding API
type ollamaEmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// GenerateEmbedding generates a vector embedding for the given text
func (o *OllamaEmbedding) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	reqBody := ollamaEmbeddingRequest{
		Model:  o.model,
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.apiURL+"/api/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var embeddingResp ollamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert float64 to float32
	embedding := make([]float32, len(embeddingResp.Embedding))
	for i, v := range embeddingResp.Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// GenerateEmbeddings generates vector embeddings for multiple texts (batch operation)
func (o *OllamaEmbedding) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("texts cannot be empty")
	}

	embeddings := make([][]float32, 0, len(texts))

	// Ollama doesn't support batch embedding natively, so we process sequentially
	for i, text := range texts {
		embedding, err := o.GenerateEmbedding(ctx, text)
		if err != nil {
			o.logger.Error("Failed to generate embedding", zap.Int("index", i), zap.Error(err))
			return nil, fmt.Errorf("failed to generate embedding for text %d: %w", i, err)
		}
		embeddings = append(embeddings, embedding)
	}

	return embeddings, nil
}

// GetDimension returns the dimension of the embedding vectors
func (o *OllamaEmbedding) GetDimension() int {
	return o.dimension
}

// GetModelName returns the name of the embedding model being used
func (o *OllamaEmbedding) GetModelName() string {
	return o.model
}
