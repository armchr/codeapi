package vector

import (
	"context"
)

// EmbeddingModel represents a generic embedding model interface
// This abstraction allows swapping between Ollama, OpenAI, Cohere, etc.
type EmbeddingModel interface {
	// GenerateEmbedding generates a vector embedding for the given text
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)

	// GenerateEmbeddings generates vector embeddings for multiple texts (batch operation)
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)

	// GetDimension returns the dimension of the embedding vectors
	GetDimension() int

	// GetModelName returns the name of the embedding model being used
	GetModelName() string
}
