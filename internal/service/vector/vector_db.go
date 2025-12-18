package vector

import (
	"github.com/armchr/codeapi/internal/model"
	"context"
)

// VectorDatabase represents a generic vector database interface
// This abstraction allows swapping between Qdrant, Weaviate, Pinecone, etc.
type VectorDatabase interface {
	// CreateCollection creates a new collection with the specified dimension and distance metric
	CreateCollection(ctx context.Context, collectionName string, vectorDim int, distance DistanceMetric) error

	// DeleteCollection deletes a collection
	DeleteCollection(ctx context.Context, collectionName string) error

	// CollectionExists checks if a collection exists
	CollectionExists(ctx context.Context, collectionName string) (bool, error)

	// UpsertChunks inserts or updates code chunks in the vector database
	UpsertChunks(ctx context.Context, collectionName string, chunks []*model.CodeChunk) error

	// SearchSimilar finds similar code chunks using vector similarity search
	SearchSimilar(ctx context.Context, collectionName string, queryVector []float32, limit int, filter map[string]interface{}) ([]*model.CodeChunk, []float32, error)

	// GetChunkByID retrieves a specific chunk by its ID
	GetChunkByID(ctx context.Context, collectionName string, chunkID string) (*model.CodeChunk, error)

	// DeleteChunk deletes a chunk by its ID
	DeleteChunk(ctx context.Context, collectionName string, chunkID string) error

	// GetChunksByFilePath retrieves all chunks for a specific file path
	GetChunksByFilePath(ctx context.Context, collectionName string, filePath string) ([]*model.CodeChunk, error)

	// Close closes the database connection
	Close() error

	// Health checks the health of the vector database
	Health(ctx context.Context) error
}

// DistanceMetric represents the distance metric used for vector similarity
type DistanceMetric string

const (
	// DistanceMetricCosine uses cosine similarity (best for normalized embeddings)
	DistanceMetricCosine DistanceMetric = "cosine"

	// DistanceMetricDot uses dot product similarity
	DistanceMetricDot DistanceMetric = "dot"

	// DistanceMetricEuclidean uses Euclidean distance
	DistanceMetricEuclidean DistanceMetric = "euclidean"
)
