package controller

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/internal/model"
	"github.com/armchr/codeapi/internal/service/vector"
	"github.com/armchr/codeapi/internal/util"

	"go.uber.org/zap"
)

// EmbeddingProcessor implements FileProcessor for code chunk embeddings
type EmbeddingProcessor struct {
	chunkService          *vector.CodeChunkService
	logger                *zap.Logger
	chunkCount            atomic.Int64
	collectionInitialized map[string]bool // Track which collections have been created
	collectionMu          sync.Mutex      // Protects collectionInitialized map
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
	// Check if we've already initialized this collection (with lock)
	ep.collectionMu.Lock()
	if ep.collectionInitialized[collectionName] {
		ep.collectionMu.Unlock()
		return nil
	}
	// Keep lock while we check/create collection to prevent race conditions
	defer ep.collectionMu.Unlock()

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

	// Index method signatures for semantic signature search
	ep.indexMethodSignatures(ctx, repo.Language, collectionName, chunks, fileCtx.FileID)

	ep.logger.Debug("Successfully processed file for embeddings",
		zap.String("path", fileCtx.FilePath),
		zap.Int32("file_id", fileCtx.FileID),
		zap.Int("chunks", len(chunks)))
	return nil
}

// indexMethodSignatures extracts and indexes method signatures from function chunks
func (ep *EmbeddingProcessor) indexMethodSignatures(ctx context.Context, language, collectionName string, chunks []*model.CodeChunk, fileID int32) {
	var signatures []vector.MethodSignatureData

	for _, chunk := range chunks {
		// Only process function chunks that have signatures
		if chunk.ChunkType != model.ChunkTypeFunction || chunk.Signature == "" {
			continue
		}

		// Parse the signature string to extract components
		sigInfo := util.ParseSignatureByLanguage(chunk.Signature, chunk.Name, chunk.ClassName, language)

		// Create signature data for indexing
		sigData := vector.MethodSignatureData{
			MethodName:     chunk.Name,
			ClassName:      chunk.ClassName,
			ReturnType:     sigInfo.ReturnType,
			ParameterTypes: sigInfo.ParameterTypes,
			ParameterNames: sigInfo.ParameterNames,
			FilePath:       chunk.FilePath,
			StartLine:      chunk.StartLine,
			EndLine:        chunk.EndLine,
			FileID:         fileID,
		}

		signatures = append(signatures, sigData)
	}

	if len(signatures) == 0 {
		return
	}

	// Index the signatures
	if err := ep.chunkService.IndexMethodSignatures(ctx, collectionName, signatures); err != nil {
		ep.logger.Warn("Failed to index method signatures",
			zap.String("collection", collectionName),
			zap.Error(err))
	}
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
