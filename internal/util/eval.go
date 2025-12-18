package util

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// EvalMetadata represents the structure of metadata.json in test cases
type EvalMetadata struct {
	TestCaseID      string        `json:"test_case_id"`
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	Type            string        `json:"type"` // "similar" or "different"
	SimilarityLevel string        `json:"similarity_level"`
	Snippets        []SnippetInfo `json:"snippets"`
	SimilarPairs    [][]string    `json:"similar_pairs"`
	DifferentPairs  [][]string    `json:"different_pairs"`
	Notes           string        `json:"notes"`
}

// SnippetInfo contains metadata about a code snippet
type SnippetInfo struct {
	File             string                 `json:"file"`
	OriginalLocation map[string]interface{} `json:"original_location"`
	Language         string                 `json:"language"`
	Description      string                 `json:"description"`
}

// SnippetSimilarity represents the similarity score between snippets
type SnippetSimilarity struct {
	SnippetName string
	Similarity  float64
}

// EmbeddingGenerator is an interface for generating embeddings from text
type EmbeddingGenerator interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// EvaluateTestCase reads a test case folder, processes snippets, and calculates
// cosine similarity of all snippets compared to the first snippet
func EvaluateTestCase(
	ctx context.Context,
	testCaseDir string,
	embeddingGen EmbeddingGenerator,
	logger *zap.Logger,
) ([]SnippetSimilarity, error) {
	// Read metadata.json
	metadataPath := filepath.Join(testCaseDir, "metadata.json")
	metadata, err := readMetadata(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	if len(metadata.Snippets) == 0 {
		return nil, fmt.Errorf("no snippets found in metadata")
	}

	logger.Info("Evaluating test case",
		zap.String("test_case_id", metadata.TestCaseID),
		zap.String("name", metadata.Name),
		zap.Int("num_snippets", len(metadata.Snippets)))

	// Read and process all snippets
	snippetEmbeddings := make(map[string][]float32)

	for _, snippet := range metadata.Snippets {
		snippetPath := filepath.Join(testCaseDir, snippet.File)

		// Read snippet content
		content, err := os.ReadFile(snippetPath)
		if err != nil {
			logger.Error("Failed to read snippet",
				zap.String("file", snippet.File),
				zap.Error(err))
			continue
		}

		// Generate embedding
		embedding, err := embeddingGen.GenerateEmbedding(ctx, string(content))
		if err != nil {
			logger.Error("Failed to generate embedding",
				zap.String("file", snippet.File),
				zap.Error(err))
			continue
		}

		snippetEmbeddings[snippet.File] = embedding
		logger.Debug("Generated embedding for snippet",
			zap.String("file", snippet.File),
			zap.Int("embedding_dim", len(embedding)))
	}

	// Get the first snippet as the query
	querySnippet := metadata.Snippets[0].File
	queryEmbedding, exists := snippetEmbeddings[querySnippet]
	if !exists {
		return nil, fmt.Errorf("failed to generate embedding for query snippet: %s", querySnippet)
	}

	// Calculate cosine similarity for all snippets compared to the first
	results := make([]SnippetSimilarity, 0, len(metadata.Snippets))

	for _, snippet := range metadata.Snippets {
		embedding, exists := snippetEmbeddings[snippet.File]
		if !exists {
			logger.Warn("Skipping snippet without embedding", zap.String("file", snippet.File))
			continue
		}

		similarity := cosineSimilarity(queryEmbedding, embedding)
		results = append(results, SnippetSimilarity{
			SnippetName: snippet.File,
			Similarity:  similarity,
		})

		logger.Info("Calculated similarity",
			zap.String("query", querySnippet),
			zap.String("target", snippet.File),
			zap.Float64("similarity", similarity))
	}

	return results, nil
}

// readMetadata reads and parses the metadata.json file
func readMetadata(path string) (*EvalMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata EvalMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata JSON: %w", err)
	}

	return &metadata, nil
}

// cosineSimilarity calculates the cosine similarity between two vectors
// Returns a value between -1 and 1, where 1 means identical, 0 means orthogonal, -1 means opposite
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// EvaluateAllTestCases evaluates all test cases in a directory and returns aggregated results
func EvaluateAllTestCases(
	ctx context.Context,
	evalDir string,
	embeddingGen EmbeddingGenerator,
	logger *zap.Logger,
) (map[string][]SnippetSimilarity, error) {
	entries, err := os.ReadDir(evalDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read eval directory: %w", err)
	}

	results := make(map[string][]SnippetSimilarity)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip non-test-case directories
		if entry.Name() == "." || entry.Name() == ".." || entry.Name()[0] == '.' {
			continue
		}

		testCaseDir := filepath.Join(evalDir, entry.Name())

		// Check if metadata.json exists
		metadataPath := filepath.Join(testCaseDir, "metadata.json")
		if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
			logger.Warn("Skipping directory without metadata.json", zap.String("dir", entry.Name()))
			continue
		}

		logger.Info("Evaluating test case directory", zap.String("dir", entry.Name()))

		similarities, err := EvaluateTestCase(ctx, testCaseDir, embeddingGen, logger)
		if err != nil {
			logger.Error("Failed to evaluate test case",
				zap.String("dir", entry.Name()),
				zap.Error(err))
			continue
		}

		results[entry.Name()] = similarities
	}

	return results, nil
}
