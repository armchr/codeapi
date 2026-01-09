package vector

import (
	"github.com/armchr/codeapi/internal/chunk"
	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/internal/model"
	"github.com/armchr/codeapi/internal/util"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	golang "github.com/tree-sitter/tree-sitter-go/bindings/go"
	java "github.com/tree-sitter/tree-sitter-java/bindings/go"
	javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
	"go.uber.org/zap"
)

// CodeChunkService orchestrates code chunking, embedding, and vector storage
type CodeChunkService struct {
	vectorDB            VectorDatabase
	embedding           EmbeddingModel
	logger              *zap.Logger
	parser              *tree_sitter.Parser
	parserMutex         sync.Mutex // Protects parser access (tree-sitter is not thread-safe)
	minConditionalLines int
	minLoopLines        int
	gcThreshold         int64
	numFileThreads      int
}

// NewCodeChunkService creates a new code chunk service
func NewCodeChunkService(vectorDB VectorDatabase, embedding EmbeddingModel, minConditionalLines, minLoopLines int, gcThreshold int64, numFileThreads int, logger *zap.Logger) *CodeChunkService {
	return &CodeChunkService{
		vectorDB:            vectorDB,
		embedding:           embedding,
		logger:              logger,
		parser:              tree_sitter.NewParser(),
		minConditionalLines: minConditionalLines,
		minLoopLines:        minLoopLines,
		gcThreshold:         gcThreshold,
		numFileThreads:      numFileThreads,
	}
}

// ProcessFile processes a single source file and stores chunks in vector DB
// Returns (chunks, error) - if error is non-nil, processing failed but can be retried
func (ccs *CodeChunkService) ProcessFile(ctx context.Context, filePath, language, collectionName string) ([]*model.CodeChunk, error) {
	// Read file content
	sourceCode, err := ccs.readFile(filePath)
	if err != nil {
		// File read errors are common (permissions, symlinks, etc.) - log and skip
		ccs.logger.Warn("Failed to read file, skipping",
			zap.String("file", filePath),
			zap.Error(err))
		return nil, nil // Return nil error to continue processing other files
	}

	return ccs.ProcessFileWithContent(ctx, filePath, language, collectionName, sourceCode)
}

// ProcessFileWithContent processes a single source file with provided content and stores chunks in vector DB
// Returns (chunks, error) - if error is non-nil, processing failed but can be retried
func (ccs *CodeChunkService) ProcessFileWithContent(ctx context.Context, filePath, language, collectionName string, sourceCode []byte) ([]*model.CodeChunk, error) {
	// Check for existing chunks in the database
	existingChunks, err := ccs.vectorDB.GetChunksByFilePath(ctx, collectionName, filePath)
	if err != nil {
		ccs.logger.Warn("Failed to fetch existing chunks, will process file anyway",
			zap.String("file", filePath),
			zap.Error(err))
		existingChunks = nil
	}

	// Parse file and generate chunks
	chunks, err := ccs.parseAndChunk(ctx, filePath, language, sourceCode)
	if err != nil {
		// Parse errors might indicate corrupted files or unsupported syntax - log and skip
		ccs.logger.Warn("Failed to parse file, skipping",
			zap.String("file", filePath),
			zap.String("language", language),
			zap.Error(err))
		return nil, nil // Return nil error to continue processing other files
	}

	if len(chunks) == 0 {
		ccs.logger.Debug("No chunks generated for file", zap.String("file", filePath))
		return nil, nil
	}

	// Build a map of existing chunk IDs for quick lookup
	existingChunkMap := make(map[string]*model.CodeChunk)
	if existingChunks != nil {
		for _, chunk := range existingChunks {
			existingChunkMap[chunk.ID] = chunk
		}
	}

	// Separate new chunks from existing chunks
	var newChunks []*model.CodeChunk
	var existingMatchedChunks []*model.CodeChunk

	for _, chunk := range chunks {
		if existingChunk, exists := existingChunkMap[chunk.ID]; exists {
			// Chunk already exists, reuse its embedding
			chunk.Embedding = existingChunk.Embedding
			existingMatchedChunks = append(existingMatchedChunks, chunk)
		} else {
			// New chunk, needs embedding
			newChunks = append(newChunks, chunk)
		}
	}

	ccs.logger.Info("Chunk analysis for file",
		zap.String("file", filePath),
		zap.Int("total_chunks", len(chunks)),
		zap.Int("existing_chunks", len(existingMatchedChunks)),
		zap.Int("new_chunks", len(newChunks)))

	// Generate embeddings only for new chunks
	var chunksToStore []*model.CodeChunk
	if len(newChunks) > 0 {
		newChunksWithEmbeddings, err := ccs.generateAndPrepareEmbeddings(ctx, newChunks)
		if err != nil {
			// Embedding errors might be transient (API issues) - log and skip
			ccs.logger.Warn("Failed to generate embeddings, skipping file",
				zap.String("file", filePath),
				zap.Error(err))
			return nil, nil // Return nil error to continue processing other files
		}
		chunksToStore = append(chunksToStore, newChunksWithEmbeddings...)
	}

	// Add existing chunks with their embeddings
	chunksToStore = append(chunksToStore, existingMatchedChunks...)

	// Store all chunks in vector database (upsert will update existing ones)
	if len(chunksToStore) > 0 {
		if err := ccs.vectorDB.UpsertChunks(ctx, collectionName, chunksToStore); err != nil {
			// Vector DB errors might be transient - log and skip
			ccs.logger.Warn("Failed to store chunks, skipping file",
				zap.String("file", filePath),
				zap.Error(err))
			return nil, nil // Return nil error to continue processing other files
		}
	}

	ccs.logger.Info("Processed file successfully",
		zap.String("file", filePath),
		zap.Int("original_chunks", len(chunks)),
		zap.Int("new_embeddings_generated", len(newChunks)),
		zap.Int("stored_chunks", len(chunksToStore)))

	return chunks, nil
}

// ProcessFileWithContentAndFileID processes a single source file with provided content and FileID
// This version is used by the IndexBuilder which provides centralized FileID from MySQL
// Returns (chunks, error) - if error is non-nil, processing failed but can be retried
func (ccs *CodeChunkService) ProcessFileWithContentAndFileID(ctx context.Context, filePath, language, collectionName string, sourceCode []byte, fileID int32) ([]*model.CodeChunk, error) {
	// Check for existing chunks in the database
	existingChunks, err := ccs.vectorDB.GetChunksByFilePath(ctx, collectionName, filePath)
	if err != nil {
		ccs.logger.Warn("Failed to fetch existing chunks, will process file anyway",
			zap.String("file", filePath),
			zap.Int32("file_id", fileID),
			zap.Error(err))
		existingChunks = nil
	}

	// Parse file and generate chunks
	chunks, err := ccs.parseAndChunk(ctx, filePath, language, sourceCode)
	if err != nil {
		// Parse errors might indicate corrupted files or unsupported syntax - log and skip
		ccs.logger.Warn("Failed to parse file, skipping",
			zap.String("file", filePath),
			zap.String("language", language),
			zap.Int32("file_id", fileID),
			zap.Error(err))
		return nil, nil // Return nil error to continue processing other files
	}

	if len(chunks) == 0 {
		ccs.logger.Debug("No chunks generated for file",
			zap.String("file", filePath),
			zap.Int32("file_id", fileID))
		return nil, nil
	}

	// Set FileID on all chunks
	for _, chunk := range chunks {
		chunk.WithFileID(fileID)
	}

	// Build a map of existing chunk IDs for quick lookup
	existingChunkMap := make(map[string]*model.CodeChunk)
	if existingChunks != nil {
		for _, chunk := range existingChunks {
			existingChunkMap[chunk.ID] = chunk
		}
	}

	// Separate new chunks from existing chunks
	var newChunks []*model.CodeChunk
	var existingMatchedChunks []*model.CodeChunk

	for _, chunk := range chunks {
		if existingChunk, exists := existingChunkMap[chunk.ID]; exists {
			// Chunk already exists, reuse its embedding
			chunk.Embedding = existingChunk.Embedding
			existingMatchedChunks = append(existingMatchedChunks, chunk)
		} else {
			// New chunk, needs embedding
			newChunks = append(newChunks, chunk)
		}
	}

	ccs.logger.Info("Chunk analysis for file",
		zap.String("file", filePath),
		zap.Int32("file_id", fileID),
		zap.Int("total_chunks", len(chunks)),
		zap.Int("existing_chunks", len(existingMatchedChunks)),
		zap.Int("new_chunks", len(newChunks)))

	// Generate embeddings only for new chunks
	var chunksToStore []*model.CodeChunk
	if len(newChunks) > 0 {
		newChunksWithEmbeddings, err := ccs.generateAndPrepareEmbeddings(ctx, newChunks)
		if err != nil {
			// Embedding errors might be transient (API issues) - log and skip
			ccs.logger.Warn("Failed to generate embeddings, skipping file",
				zap.String("file", filePath),
				zap.Int32("file_id", fileID),
				zap.Error(err))
			return nil, nil // Return nil error to continue processing other files
		}
		chunksToStore = append(chunksToStore, newChunksWithEmbeddings...)
	}

	// Add existing chunks with their embeddings
	chunksToStore = append(chunksToStore, existingMatchedChunks...)

	// Store all chunks in vector database (upsert will update existing ones)
	if len(chunksToStore) > 0 {
		if err := ccs.vectorDB.UpsertChunks(ctx, collectionName, chunksToStore); err != nil {
			// Vector DB errors might be transient - log and skip
			ccs.logger.Warn("Failed to store chunks, skipping file",
				zap.String("file", filePath),
				zap.Int32("file_id", fileID),
				zap.Error(err))
			return nil, nil // Return nil error to continue processing other files
		}
	}

	ccs.logger.Info("Processed file successfully",
		zap.String("file", filePath),
		zap.Int32("file_id", fileID),
		zap.Int("original_chunks", len(chunks)),
		zap.Int("new_embeddings_generated", len(newChunks)),
		zap.Int("stored_chunks", len(chunksToStore)))

	return chunks, nil
}

// ProcessDirectory processes all supported files in a directory recursively
// Gracefully skips files that fail to read or process
func (ccs *CodeChunkService) ProcessDirectory(ctx context.Context, dirPath, collectionName string, repoConfig interface{}) (int, error) {
	totalChunks := 0
	filesFailed := 0

	// Extract repository configuration if provided
	var skipOtherLanguages bool
	var repoLanguage string
	if repo, ok := repoConfig.(*config.Repository); ok && repo != nil {
		skipOtherLanguages = repo.SkipOtherLanguages
		repoLanguage = repo.Language
		if skipOtherLanguages {
			ccs.logger.Info("Skip other languages enabled",
				zap.String("repo_language", repoLanguage),
				zap.String("dir", dirPath))
		}
	}

	err := util.WalkDirTree(dirPath, func(path string, err error) error {
		if err != nil {
			return err
		}

		language := ccs.detectLanguage(path)
		if language == "" {
			ccs.logger.Info("WalkDirTree - Skipping unsupported file", zap.String("path", path))
			return nil
		}
		// Process file
		chunks, err := ccs.ProcessFile(ctx, path, language, collectionName)
		if err != nil {
			// This shouldn't happen as ProcessFile now handles errors internally
			// But keep this as a safeguard
			ccs.logger.Error("WalkDirTree - Unexpected error processing file", zap.String("path", path), zap.Error(err))
			filesFailed++
			return nil // Continue processing other files
		}

		// Count successful processing (chunks might be nil if file was skipped internally)
		if chunks != nil {
			totalChunks += len(chunks)
		} else {
			// File was skipped due to error (logged by ProcessFile)
			filesFailed++
		}

		return nil
	},
		func(path string, isDir bool) bool {
			// Skip excluded directories
			if isDir {
				if ccs.shouldSkipDirectory(path, filepath.Base(path)) {
					ccs.logger.Info("WalkDirTree - Skipping directory", zap.String("path", path))
					return true
				}
				return false
			}

			language := ccs.detectLanguage(path)
			if language == "" {
				ccs.logger.Info("WalkDirTree - Skipping unsupported file", zap.String("path", path))
				return true
			}

			// Skip files of other languages if skip_other_languages is enabled
			if skipOtherLanguages && language != repoLanguage {
				ccs.logger.Error("WalkDirTree - Skipping file due to language mismatch",
					zap.String("path", path),
					zap.String("file_language", language),
					zap.String("repo_language", repoLanguage))
				return true
			}
			return false
		},
		ccs.logger,
		ccs.gcThreshold,
		ccs.numFileThreads)

	if err != nil {
		return totalChunks, fmt.Errorf("WalkDirTree - failed to process directory: %w", err)
	}

	// Final GC to clean up
	runtime.GC()

	ccs.logger.Info("WalkDirTree - Processed directory successfully",
		zap.String("dir", dirPath),
		//zap.Int("files_processed", filesProcessed),
		//zap.Int("files_skipped", filesSkipped),
		zap.Int("total_chunks", totalChunks))

	return totalChunks, nil
}

// SearchSimilarCode searches for code chunks similar to the given query text
func (ccs *CodeChunkService) SearchSimilarCode(ctx context.Context, collectionName, queryText string, limit int, filter map[string]interface{}) ([]*model.CodeChunk, []float32, error) {
	// Generate embedding for query text
	queryVector, err := ccs.embedding.GenerateEmbedding(ctx, queryText)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search in vector database
	chunks, scores, err := ccs.vectorDB.SearchSimilar(ctx, collectionName, queryVector, limit, filter)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to search: %w", err)
	}

	return chunks, scores, nil
}

// SearchSimilarCodeBySnippet chunks a code snippet and searches for similar code in the database
func (ccs *CodeChunkService) SearchSimilarCodeBySnippet(ctx context.Context, collectionName, codeSnippet, language string, limit int, filter map[string]interface{}) ([]*model.CodeChunk, []*model.CodeChunk, []float32, []int, error) {
	// Parse and chunk the code snippet
	queryChunks, err := ccs.parseAndChunk(ctx, "query.snippet", language, []byte(codeSnippet))
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to parse code snippet: %w", err)
	}

	if len(queryChunks) == 0 {
		return nil, nil, nil, nil, fmt.Errorf("no chunks generated from code snippet")
	}

	// For each query chunk, generate embeddings and search
	// We'll aggregate results from all query chunks
	allResults := make(map[string]*resultWithScore)

	for queryChunkIndex, queryChunk := range queryChunks {
		// Generate embedding for the query chunk (with context)
		searchableText := queryChunk.GetSearchableText(true)
		queryVector, err := ccs.embedding.GenerateEmbedding(ctx, searchableText)
		if err != nil {
			ccs.logger.Warn("Failed to generate embedding for query chunk",
				zap.String("chunk_type", string(queryChunk.ChunkType)),
				zap.Error(err))
			continue
		}

		// Search in vector database
		resultChunks, scores, err := ccs.vectorDB.SearchSimilar(ctx, collectionName, queryVector, limit, filter)
		if err != nil {
			ccs.logger.Warn("Failed to search for query chunk",
				zap.String("chunk_type", string(queryChunk.ChunkType)),
				zap.Error(err))
			continue
		}

		// Aggregate results (keep highest score for each unique chunk)
		for i, chunk := range resultChunks {
			if existing, ok := allResults[chunk.ID]; ok {
				// Keep the higher score and update query chunk index
				if scores[i] > existing.score {
					existing.score = scores[i]
					existing.queryChunkIndex = queryChunkIndex
				}
			} else {
				allResults[chunk.ID] = &resultWithScore{
					chunk:           chunk,
					score:           scores[i],
					queryChunkIndex: queryChunkIndex,
				}
			}
		}
	}

	// Convert map to slices and sort by score
	chunks := make([]*model.CodeChunk, 0, len(allResults))
	scores := make([]float32, 0, len(allResults))
	queryChunkIndices := make([]int, 0, len(allResults))

	for _, result := range allResults {
		chunks = append(chunks, result.chunk)
		scores = append(scores, result.score)
		queryChunkIndices = append(queryChunkIndices, result.queryChunkIndex)
	}

	// Sort by score descending (keep indices aligned)
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j] > scores[i] {
				scores[i], scores[j] = scores[j], scores[i]
				chunks[i], chunks[j] = chunks[j], chunks[i]
				queryChunkIndices[i], queryChunkIndices[j] = queryChunkIndices[j], queryChunkIndices[i]
			}
		}
	}

	// Limit results
	if len(chunks) > limit {
		chunks = chunks[:limit]
		scores = scores[:limit]
		queryChunkIndices = queryChunkIndices[:limit]
	}

	return queryChunks, chunks, scores, queryChunkIndices, nil
}

type resultWithScore struct {
	chunk           *model.CodeChunk
	score           float32
	queryChunkIndex int
}

// CreateCollection creates a new collection in the vector database
func (ccs *CodeChunkService) CreateCollection(ctx context.Context, collectionName string) error {
	exists, err := ccs.vectorDB.CollectionExists(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if exists {
		ccs.logger.Info("Collection already exists", zap.String("collection", collectionName))
		return nil
	}

	dimension := ccs.embedding.GetDimension()
	if err := ccs.vectorDB.CreateCollection(ctx, collectionName, dimension, DistanceMetricCosine); err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	ccs.logger.Info("Created collection", zap.String("collection", collectionName), zap.Int("dimension", dimension))
	return nil
}

// DeleteCollection deletes a collection from the vector database
func (ccs *CodeChunkService) DeleteCollection(ctx context.Context, collectionName string) error {
	if err := ccs.vectorDB.DeleteCollection(ctx, collectionName); err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	ccs.logger.Info("Deleted collection", zap.String("collection", collectionName))
	return nil
}

// Helper methods

func (ccs *CodeChunkService) parseAndChunk(ctx context.Context, filePath, language string, sourceCode []byte) ([]*model.CodeChunk, error) {
	// Get tree-sitter language
	tsLanguage, err := ccs.getTreeSitterLanguage(language)
	if err != nil {
		return nil, err
	}

	// Lock parser access (tree-sitter is not thread-safe)
	ccs.parserMutex.Lock()
	defer ccs.parserMutex.Unlock()

	// Set parser language
	if err := ccs.parser.SetLanguage(tsLanguage); err != nil {
		return nil, fmt.Errorf("failed to set parser language: %w", err)
	}

	// Parse source code
	tree := ccs.parser.Parse(sourceCode, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse file")
	}
	defer tree.Close()

	// Create chunk visitor
	visitor := chunk.NewChunkVisitor(ccs.logger, language, filePath, sourceCode, ccs.minConditionalLines, ccs.minLoopLines)

	// Traverse syntax tree
	rootNode := tree.RootNode()
	visitor.TraverseNode(ctx, rootNode, nil)

	return visitor.GetChunks(), nil
}

func (ccs *CodeChunkService) generateAndPrepareEmbeddings(ctx context.Context, chunks []*model.CodeChunk) ([]*model.CodeChunk, error) {
	// For conditionals and loops, we generate TWO embeddings: with and without context
	// For other chunk types, we generate ONE embedding with context

	// Separate chunks into two categories
	var needsTwoEmbeddings []*model.CodeChunk
	var needsOneEmbedding []*model.CodeChunk
	var embeddingsWithoutContext [][]float32

	for _, chunk := range chunks {
		if chunk.ChunkType == model.ChunkTypeConditional || chunk.ChunkType == model.ChunkTypeLoop {
			needsTwoEmbeddings = append(needsTwoEmbeddings, chunk)
		} else {
			needsOneEmbedding = append(needsOneEmbedding, chunk)
		}
	}

	// Generate embeddings for chunks that need one embedding (with context)
	if len(needsOneEmbedding) > 0 {
		texts := make([]string, 0, len(needsOneEmbedding))
		validChunks := make([]*model.CodeChunk, 0, len(needsOneEmbedding))

		for _, chunk := range needsOneEmbedding {
			text := chunk.GetSearchableText(true) // with context
			if text != "" {
				texts = append(texts, text)
				validChunks = append(validChunks, chunk)
			} else {
				ccs.logger.Warn("Skipping chunk with empty searchable text",
					zap.String("id", chunk.ID),
					zap.String("type", string(chunk.ChunkType)),
					zap.String("file", chunk.FilePath))
			}
		}

		if len(texts) == 0 {
			ccs.logger.Warn("No valid texts for embedding generation in needsOneEmbedding")
		} else {
			embeddings, err := ccs.embedding.GenerateEmbeddings(ctx, texts)
			if err != nil {
				return nil, fmt.Errorf("failed to generate embeddings for standard chunks: %w", err)
			}

			for i, embedding := range embeddings {
				validChunks[i].Embedding = embedding
				/*
					ccs.logger.Info("Generated embedding for chunk",
						zap.String("id", validChunks[i].ID),
						zap.String("type", string(validChunks[i].ChunkType)),
						zap.String("file", validChunks[i].FilePath),
						zap.String("name", validChunks[i].Name),
						zap.Int("level", validChunks[i].Level),
						zap.Int("start_line", validChunks[i].StartLine),
						zap.Int("end_line", validChunks[i].EndLine),
						zap.String("signature", validChunks[i].Signature),
						zap.Int("embedding_dim", len(embedding)),
						zap.Int("content_length", len(validChunks[i].Content)),
						zap.Bool("with_context", true))
				*/
			}
		}
	}

	// Generate TWO embeddings for conditionals/loops
	if len(needsTwoEmbeddings) > 0 {
		// First: with context - filter empty texts
		textsWithContext := make([]string, 0, len(needsTwoEmbeddings))
		validTwoEmbeddingChunks := make([]*model.CodeChunk, 0, len(needsTwoEmbeddings))

		for _, chunk := range needsTwoEmbeddings {
			text := chunk.GetSearchableText(true)
			if text != "" {
				textsWithContext = append(textsWithContext, text)
				validTwoEmbeddingChunks = append(validTwoEmbeddingChunks, chunk)
			} else {
				ccs.logger.Warn("Skipping chunk with empty searchable text (with context)",
					zap.String("id", chunk.ID),
					zap.String("type", string(chunk.ChunkType)),
					zap.String("file", chunk.FilePath))
			}
		}

		if len(textsWithContext) == 0 {
			ccs.logger.Warn("No valid texts for embedding generation in needsTwoEmbeddings")
		} else {
			embeddingsWithContext, err := ccs.embedding.GenerateEmbeddings(ctx, textsWithContext)
			if err != nil {
				return nil, fmt.Errorf("failed to generate embeddings with context: %w", err)
			}

			// Second: without context
			textsWithoutContext := make([]string, 0, len(validTwoEmbeddingChunks))
			for _, chunk := range validTwoEmbeddingChunks {
				text := chunk.GetSearchableText(false)
				if text != "" {
					textsWithoutContext = append(textsWithoutContext, text)
				} else {
					// This shouldn't happen if with-context wasn't empty, but handle it
					textsWithoutContext = append(textsWithoutContext, chunk.Content)
				}
			}

			embeddingsWithoutContext, err = ccs.embedding.GenerateEmbeddings(ctx, textsWithoutContext)
			if err != nil {
				return nil, fmt.Errorf("failed to generate embeddings without context: %w", err)
			}

			// Store and log both embeddings
			for i := range validTwoEmbeddingChunks {
				// Store the with-context embedding as the primary one
				validTwoEmbeddingChunks[i].Embedding = embeddingsWithContext[i]

				// Generate the no-context ID for logging
				//noContextID := ccs.generateNoContextID(validTwoEmbeddingChunks[i].ID)

				/*
					ccs.logger.Info("Generated embedding for chunk (with context)",
						zap.String("id", validTwoEmbeddingChunks[i].ID),
						zap.String("type", string(validTwoEmbeddingChunks[i].ChunkType)),
						zap.String("file", validTwoEmbeddingChunks[i].FilePath),
						zap.String("name", validTwoEmbeddingChunks[i].Name),
						zap.Int("level", validTwoEmbeddingChunks[i].Level),
						zap.Int("start_line", validTwoEmbeddingChunks[i].StartLine),
						zap.Int("end_line", validTwoEmbeddingChunks[i].EndLine),
						zap.Int("embedding_dim", len(embeddingsWithContext[i])),
						zap.Bool("with_context", true))

					ccs.logger.Info("Generated embedding for chunk (without context)",
						zap.String("id", noContextID),
						zap.String("type", string(validTwoEmbeddingChunks[i].ChunkType)),
						zap.String("file", validTwoEmbeddingChunks[i].FilePath),
						zap.String("name", validTwoEmbeddingChunks[i].Name),
						zap.Int("level", validTwoEmbeddingChunks[i].Level),
						zap.Int("start_line", validTwoEmbeddingChunks[i].StartLine),
						zap.Int("end_line", validTwoEmbeddingChunks[i].EndLine),
						zap.Int("embedding_dim", len(embeddingsWithoutContext[i])),
						zap.Bool("with_context", false))
				*/
			}

			// Update needsTwoEmbeddings to only include valid chunks
			needsTwoEmbeddings = validTwoEmbeddingChunks
		}
	}

	// Build final list of chunks to store
	// For conditionals/loops, create duplicate chunks with different IDs and embeddings
	result := make([]*model.CodeChunk, 0, len(chunks)+len(needsTwoEmbeddings))

	// Add all single-embedding chunks
	result = append(result, needsOneEmbedding...)

	// Add dual-embedding chunks (both with-context and without-context versions)
	for i, chunk := range needsTwoEmbeddings {
		// Add with-context version (already has embedding)
		result = append(result, chunk)

		// Create without-context version as a duplicate with modified ID
		// Generate a proper UUID by hashing the original ID with a suffix
		noContextID := ccs.generateNoContextID(chunk.ID)

		chunkNoContext := &model.CodeChunk{
			ID:         noContextID,
			ChunkType:  chunk.ChunkType,
			Level:      chunk.Level,
			ParentID:   chunk.ParentID,
			Content:    chunk.Content,
			Language:   chunk.Language,
			FilePath:   chunk.FilePath,
			StartLine:  chunk.StartLine,
			EndLine:    chunk.EndLine,
			Range:      chunk.Range,
			Name:       chunk.Name,
			Signature:  chunk.Signature,
			Docstring:  chunk.Docstring,
			ModuleName: "", // No context
			ClassName:  "", // No context
			Embedding:  embeddingsWithoutContext[i],
			Metadata:   map[string]interface{}{"context_mode": "nocontext", "original_id": chunk.ID},
		}
		result = append(result, chunkNoContext)
	}

	return result, nil
}

func (ccs *CodeChunkService) detectLanguage(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".go":
		return "go"
	case ".py", ".pyw":
		return "python"
	case ".java":
		return "java"
	case ".js", ".jsx", ".mjs":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	default:
		return ""
	}
}

func (ccs *CodeChunkService) getTreeSitterLanguage(language string) (*tree_sitter.Language, error) {
	switch language {
	case "go":
		return tree_sitter.NewLanguage(golang.Language()), nil
	case "python":
		return tree_sitter.NewLanguage(python.Language()), nil
	case "java":
		return tree_sitter.NewLanguage(java.Language()), nil
	case "javascript":
		return tree_sitter.NewLanguage(javascript.Language()), nil
	case "typescript":
		return tree_sitter.NewLanguage(typescript.LanguageTypescript()), nil
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
}

func (ccs *CodeChunkService) readFile(filePath string) ([]byte, error) {
	// Use os.ReadFile which opens, reads, and closes in one operation
	// This is more efficient and ensures file descriptors are released immediately
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return content, nil
}

// ReadCodeFromFile reads specific lines from a file
func (ccs *CodeChunkService) ReadCodeFromFile(filePath string, startLine, endLine int) (string, error) {
	content, err := ccs.readFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Validate line numbers (0-indexed internally)
	if startLine < 0 || startLine >= len(lines) {
		return "", fmt.Errorf("invalid start line: %d", startLine)
	}
	if endLine < 0 || endLine >= len(lines) {
		endLine = len(lines) - 1
	}
	if startLine > endLine {
		return "", fmt.Errorf("start line (%d) greater than end line (%d)", startLine, endLine)
	}

	// Extract lines (inclusive)
	codeLines := lines[startLine : endLine+1]
	return strings.Join(codeLines, "\n"), nil
}

// Close closes all resources
func (ccs *CodeChunkService) Close() error {
	if ccs.vectorDB != nil {
		return ccs.vectorDB.Close()
	}
	return nil
}

// GetVectorDB returns the vector database instance
func (ccs *CodeChunkService) GetVectorDB() VectorDatabase {
	return ccs.vectorDB
}

// GetEmbeddingModel returns the embedding model instance
func (ccs *CodeChunkService) GetEmbeddingModel() EmbeddingModel {
	return ccs.embedding
}

// shouldSkipDirectory checks if a directory should be excluded from processing
func (ccs *CodeChunkService) shouldSkipDirectory(path, name string) bool {
	// Common directories to skip
	skipDirs := []string{
		".git",
		".env",
		".venv",
		"venv",
		"env",
		"node_modules",
		"vendor",
		"target",
		"build",
		"dist",
		"bin",
		"obj",
		"__pycache__",
		".idea",
		".vscode",
		".pytest_cache",
		".mypy_cache",
		".tox",
		"coverage",
		".next",
		".nuxt",
		"out",
	}

	for _, skipDir := range skipDirs {
		if name == skipDir {
			return true
		}
	}

	// Skip hidden directories (starting with .)
	if len(name) > 0 && name[0] == '.' {
		return true
	}

	return false
}

// generateNoContextID generates a proper UUID for the no-context version of a chunk
// by hashing the original ID with a suffix
func (ccs *CodeChunkService) generateNoContextID(originalID string) string {
	input := originalID + ":nocontext"
	hash := sha256.Sum256([]byte(input))
	hashStr := hex.EncodeToString(hash[:])

	// Convert hash to UUID format (8-4-4-4-12)
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hashStr[0:8],
		hashStr[8:12],
		hashStr[12:16],
		hashStr[16:20],
		hashStr[20:32],
	)
}

// MethodSignatureData holds information for indexing a method signature
type MethodSignatureData struct {
	MethodName     string
	ClassName      string
	ReturnType     string
	ParameterTypes []string
	ParameterNames []string
	FilePath       string
	StartLine      int
	EndLine        int
	FileID         int32
}

// IndexMethodSignatures indexes method signatures for semantic search
// The normalized signature text is embedded and stored separately from code content
func (ccs *CodeChunkService) IndexMethodSignatures(ctx context.Context, collectionName string, signatures []MethodSignatureData) error {
	if len(signatures) == 0 {
		return nil
	}

	// Create chunks for each signature
	var chunks []*model.CodeChunk
	var textsToEmbed []string

	for _, sig := range signatures {
		// Build SignatureInfo for normalization
		sigInfo := util.BuildSignatureInfo(sig.ClassName, sig.MethodName, sig.ReturnType, sig.ParameterNames, sig.ParameterTypes)

		// Generate normalized text for embedding
		normalizedText := util.NormalizeSignatureForEmbedding(sigInfo)
		if normalizedText == "" {
			continue
		}

		// Generate unique ID for this signature chunk
		chunkID := ccs.generateSignatureChunkID(sig.FilePath, sig.ClassName, sig.MethodName, uint(sig.StartLine))

		// Format human-readable signature
		formattedSig := util.FormatSignatureString(sigInfo)

		chunk := &model.CodeChunk{
			ID:        chunkID,
			FileID:    sig.FileID,
			ChunkType: model.ChunkTypeMethodSignature,
			Level:     3, // Same level as function
			Content:   normalizedText,
			Language:  "", // Not language-specific for signatures
			FilePath:  sig.FilePath,
			StartLine: sig.StartLine,
			EndLine:   sig.EndLine,
			Name:      sig.MethodName,
			Signature: formattedSig,
			ClassName: sig.ClassName,
			Metadata: map[string]interface{}{
				"return_type":     sig.ReturnType,
				"parameter_types": strings.Join(sig.ParameterTypes, ","),
				"parameter_names": strings.Join(sig.ParameterNames, ","),
				"normalized_text": normalizedText,
			},
		}

		chunks = append(chunks, chunk)
		textsToEmbed = append(textsToEmbed, normalizedText)
	}

	if len(chunks) == 0 {
		return nil
	}

	// Generate embeddings for normalized signature texts
	embeddings, err := ccs.embedding.GenerateEmbeddings(ctx, textsToEmbed)
	if err != nil {
		return fmt.Errorf("failed to generate signature embeddings: %w", err)
	}

	// Assign embeddings to chunks
	for i, embedding := range embeddings {
		chunks[i].Embedding = embedding
	}

	// Store in vector database
	if err := ccs.vectorDB.UpsertChunks(ctx, collectionName, chunks); err != nil {
		return fmt.Errorf("failed to store signature chunks: %w", err)
	}

	ccs.logger.Info("Indexed method signatures",
		zap.String("collection", collectionName),
		zap.Int("count", len(chunks)))

	return nil
}

// SearchMethodSignatures searches for methods by natural language query on their signatures
func (ccs *CodeChunkService) SearchMethodSignatures(ctx context.Context, collectionName, query string, limit int) ([]*model.CodeChunk, []float32, error) {
	// Normalize the query text similarly to how signatures are normalized
	// This helps match queries like "find user by email" to "findByEmail"
	queryForEmbedding := query

	// Generate embedding for query
	queryVector, err := ccs.embedding.GenerateEmbedding(ctx, queryForEmbedding)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Create filter to only search method_signature chunks
	filter := map[string]interface{}{
		"chunk_type": string(model.ChunkTypeMethodSignature),
	}

	// Search in vector database
	chunks, scores, err := ccs.vectorDB.SearchSimilar(ctx, collectionName, queryVector, limit, filter)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to search signatures: %w", err)
	}

	return chunks, scores, nil
}

// generateSignatureChunkID generates a unique ID for a method signature chunk
func (ccs *CodeChunkService) generateSignatureChunkID(filePath, className, methodName string, line uint) string {
	input := fmt.Sprintf("%s:%s:%s:%d:signature", filePath, className, methodName, line)
	hash := sha256.Sum256([]byte(input))
	hashStr := hex.EncodeToString(hash[:])

	// Convert hash to UUID format (8-4-4-4-12)
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hashStr[0:8],
		hashStr[8:12],
		hashStr[12:16],
		hashStr[16:20],
		hashStr[20:32],
	)
}
