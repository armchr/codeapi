package controller

import (
	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/internal/service/vector"
	"context"
	"sync/atomic"

	"go.uber.org/zap"
)

// EmbeddingProcessor implements FileProcessor for code chunk embeddings
type EmbeddingProcessor struct {
	chunkService         *vector.CodeChunkService
	logger               *zap.Logger
	chunkCount           atomic.Int64
	collectionInitialized map[string]bool // Track which collections have been created
}

// NewEmbeddingProcessor creates a new embedding processor
func NewEmbeddingProcessor(chunkService *vector.CodeChunkService, logger *zap.Logger) *EmbeddingProcessor {
	return &EmbeddingProcessor{
		chunkService:          chunkService,
		logger:                logger,
		collectionInitialized: make(map[string]bool),
	}
}

// Name returns the processor name
func (ep *EmbeddingProcessor) Name() string {
	return "Embedding"
}

// Init initializes the processor for a repository (no-op for EmbeddingProcessor)
func (ep *EmbeddingProcessor) Init(ctx context.Context, repo *config.Repository) error {
	return nil
}

// ensureCollection ensures the Qdrant collection exists for the repository
func (ep *EmbeddingProcessor) ensureCollection(ctx context.Context, collectionName string) error {
	// Check if we've already initialized this collection
	if ep.collectionInitialized[collectionName] {
		return nil
	}

	// Check if collection exists in Qdrant
	exists, err := ep.chunkService.GetVectorDB().CollectionExists(ctx, collectionName)
	if err != nil {
		return err
	}

	if !exists {
		ep.logger.Info("Creating Qdrant collection", zap.String("collection", collectionName))
		// Get embedding dimension from the embedding model
		vectorDim := ep.chunkService.GetEmbeddingModel().GetDimension()
		err = ep.chunkService.GetVectorDB().CreateCollection(ctx, collectionName, vectorDim, vector.DistanceMetricCosine)
		if err != nil {
			return err
		}
		ep.logger.Info("Qdrant collection created successfully", zap.String("collection", collectionName))
	}

	// Mark collection as initialized
	ep.collectionInitialized[collectionName] = true
	return nil
}

// ProcessFile processes a single file for embedding generation
func (ep *EmbeddingProcessor) ProcessFile(ctx context.Context, repo *config.Repository, fileCtx *FileContext) error {
	ep.logger.Debug("Processing file for embeddings",
		zap.String("path", fileCtx.FilePath),
		zap.Int32("file_id", fileCtx.FileID))

	collectionName := repo.Name

	// Ensure collection exists before processing
	if err := ep.ensureCollection(ctx, collectionName); err != nil {
		ep.logger.Error("Failed to ensure collection exists",
			zap.String("collection", collectionName),
			zap.Error(err))
		return nil // Continue processing other files
	}

	chunks, err := ep.chunkService.ProcessFileWithContentAndFileID(
		ctx,
		fileCtx.FilePath,
		repo.Language,
		collectionName,
		fileCtx.Content,
		fileCtx.FileID,
	)
	if err != nil {
		ep.logger.Error("Failed to process file for embeddings",
			zap.String("path", fileCtx.FilePath),
			zap.Int32("file_id", fileCtx.FileID),
			zap.Error(err))
		return nil // Continue processing other files
	}

	// Track total chunks processed
	ep.chunkCount.Add(int64(len(chunks)))

	ep.logger.Debug("Successfully processed file for embeddings",
		zap.String("path", fileCtx.FilePath),
		zap.Int32("file_id", fileCtx.FileID),
		zap.Int("chunks", len(chunks)))
	return nil
}

// PostProcess performs any cleanup or finalization after all files are processed
func (ep *EmbeddingProcessor) PostProcess(ctx context.Context, repo *config.Repository) error {
	totalChunks := ep.chunkCount.Load()
	ep.logger.Info("Embedding processing completed",
		zap.String("repo_name", repo.Name),
		zap.Int64("total_chunks", totalChunks))

	// Reset counter for next repository
	ep.chunkCount.Store(0)
	return nil
}
