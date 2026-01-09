package init

import (
	"context"
	"fmt"

	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/internal/controller"
	"github.com/armchr/codeapi/internal/db"
	"github.com/armchr/codeapi/internal/service"
	"github.com/armchr/codeapi/internal/service/codegraph"
	"github.com/armchr/codeapi/internal/service/llm"
	"github.com/armchr/codeapi/internal/service/summary"
	"github.com/armchr/codeapi/internal/service/vector"

	"go.uber.org/zap"
)

// ServiceContainer holds all initialized services and their lifecycle management
type ServiceContainer struct {
	// Database connections
	MySQLConn *db.MySQLConnection

	// Core services
	CodeGraph      *codegraph.CodeGraph
	VectorDB       vector.VectorDatabase
	EmbeddingModel vector.EmbeddingModel
	ChunkService   *vector.CodeChunkService
	RepoService    *service.RepoService

	// Summary services
	LLMService       llm.LLMService
	PromptManager    *summary.PromptManager
	SummaryProcessor *controller.SummaryProcessor // Exposed for on-demand generation

	// Processors
	Processors []controller.FileProcessor

	logger *zap.Logger
}

// ServiceInitOptions configures which services to initialize
type ServiceInitOptions struct {
	EnableMySQL       bool
	EnableCodeGraph   bool
	EnableEmbeddings  bool
	EnableRepoService bool
	EnableSummary     bool // Enable hierarchical code summarization

	// For index building CLI mode
	RequireMySQL bool // If true, fail if MySQL is not available
}

// NewServiceContainer initializes all requested services based on options
func NewServiceContainer(cfg *config.Config, opts ServiceInitOptions, logger *zap.Logger) (*ServiceContainer, error) {
	container := &ServiceContainer{
		logger: logger,
	}

	var err error

	// Initialize MySQL if enabled
	if opts.EnableMySQL && cfg.MySQL.Host != "" {
		container.MySQLConn, err = initMySQL(cfg, logger, opts.RequireMySQL)
		if err != nil {
			if opts.RequireMySQL {
				return nil, fmt.Errorf("MySQL initialization failed (required): %w", err)
			}
			logger.Warn("MySQL initialization failed, continuing without it", zap.Error(err))
		}
	} else if opts.RequireMySQL {
		return nil, fmt.Errorf("MySQL configuration is required but not provided")
	}

	// Initialize RepoService if enabled (needed for LSP operations)
	if opts.EnableRepoService {
		container.RepoService = service.NewRepoService(cfg, logger)
		logger.Info("RepoService initialized")
	}

	// Initialize CodeGraph if enabled
	if opts.EnableCodeGraph {
		container.CodeGraph, err = initCodeGraph(cfg, logger)
		if err != nil {
			return nil, fmt.Errorf("CodeGraph initialization failed: %w", err)
		}
		logger.Info("CodeGraph initialized")
	}

	// Initialize Vector DB and Embeddings if enabled
	if opts.EnableEmbeddings {
		container.VectorDB, container.EmbeddingModel, container.ChunkService, err = initVectorServices(cfg, logger)
		if err != nil {
			return nil, fmt.Errorf("Vector services initialization failed: %w", err)
		}
		logger.Info("Vector services initialized")
	}

	// Initialize Summary services if enabled
	if opts.EnableSummary {
		container.LLMService, container.PromptManager, err = initSummaryServices(cfg, logger)
		if err != nil {
			// Summary is optional, log warning but don't fail
			logger.Warn("Summary services initialization failed, summarization will be disabled", zap.Error(err))
		} else {
			logger.Info("Summary services initialized",
				zap.String("llm_provider", container.LLMService.Name()),
				zap.String("llm_model", container.LLMService.ModelName()))
		}
	}

	return container, nil
}

// InitProcessors creates FileProcessor instances based on enabled services
func (sc *ServiceContainer) InitProcessors(cfg *config.Config) error {
	var processors []controller.FileProcessor

	// Add CodeGraph processor if available
	if sc.CodeGraph != nil {
		if sc.RepoService == nil {
			return fmt.Errorf("CodeGraph processor requires RepoService but it's not initialized")
		}
		codeGraphProcessor := controller.NewCodeGraphProcessor(cfg, sc.CodeGraph, sc.RepoService, sc.logger)
		processors = append(processors, codeGraphProcessor)
		sc.logger.Info("CodeGraph processor added to pipeline")
	}

	// Add Embedding processor if available
	if sc.ChunkService != nil {
		embeddingProcessor := controller.NewEmbeddingProcessor(sc.ChunkService, sc.logger)
		processors = append(processors, embeddingProcessor)
		sc.logger.Info("Embedding processor added to pipeline")
	}

	// Add Summary processor if LLM service is available
	// Note: Summary processor requires CodeGraph to be available for entity queries
	if sc.LLMService != nil && sc.PromptManager != nil && sc.CodeGraph != nil && sc.MySQLConn != nil {
		summaryConfig := &controller.SummaryProcessorConfig{
			Enabled:      cfg.IndexBuilding.EnableSummary,
			WorkerCount:  cfg.Summary.WorkerCount,
			SkipIfExists: cfg.Summary.SkipIfExists,
			BatchSize:    cfg.Summary.BatchSize,
		}
		if summaryConfig.WorkerCount <= 0 {
			summaryConfig.WorkerCount = 4
		}
		if summaryConfig.BatchSize <= 0 {
			summaryConfig.BatchSize = 50
		}

		// Pass MySQL DB for creating per-repo summary stores
		summaryProcessor := controller.NewSummaryProcessor(
			sc.LLMService,
			sc.PromptManager,
			sc.MySQLConn.GetDB(), // MySQL DB for per-repo stores
			sc.CodeGraph,
			summaryConfig,
			sc.logger,
		)
		processors = append(processors, summaryProcessor)
		sc.SummaryProcessor = summaryProcessor // Store for on-demand API access
		sc.logger.Info("Summary processor added to pipeline")
	}

	sc.Processors = processors
	return nil
}

// Close cleans up all resources
func (sc *ServiceContainer) Close(ctx context.Context) {
	if sc.MySQLConn != nil {
		sc.MySQLConn.Close()
		sc.logger.Info("MySQL connection closed")
	}

	if sc.CodeGraph != nil {
		sc.CodeGraph.Close(ctx)
		sc.logger.Info("CodeGraph closed")
	}

	if sc.VectorDB != nil {
		sc.VectorDB.Close()
		sc.logger.Info("Vector DB closed")
	}
}

// initMySQL initializes MySQL connection and ensures database exists
func initMySQL(cfg *config.Config, logger *zap.Logger, required bool) (*db.MySQLConnection, error) {
	mysqlConn, err := db.NewMySQLConnection(cfg.MySQL, logger)
	if err != nil {
		if required {
			return nil, fmt.Errorf("failed to initialize MySQL connection: %w", err)
		}
		logger.Error("Failed to initialize MySQL connection, FileID tracking will be disabled", zap.Error(err))
		return nil, err
	}

	// Ensure armchair database exists
	if err := mysqlConn.EnsureDatabase("armchair"); err != nil {
		mysqlConn.Close()
		if required {
			return nil, fmt.Errorf("failed to ensure armchair database: %w", err)
		}
		logger.Error("Failed to ensure armchair database, FileID tracking will be disabled", zap.Error(err))
		return nil, err
	}

	logger.Info("MySQL connection established and armchair database verified")
	return mysqlConn, nil
}

// initCodeGraph initializes the CodeGraph service
func initCodeGraph(cfg *config.Config, logger *zap.Logger) (*codegraph.CodeGraph, error) {
	codeGraph, err := codegraph.NewCodeGraph(
		cfg.Neo4j.URI,
		cfg.Neo4j.Username,
		cfg.Neo4j.Password,
		cfg,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize CodeGraph: %w", err)
	}

	return codeGraph, nil
}

// initVectorServices initializes Vector DB, Embedding model, and CodeChunkService
func initVectorServices(cfg *config.Config, logger *zap.Logger) (vector.VectorDatabase, vector.EmbeddingModel, *vector.CodeChunkService, error) {
	// Validate configuration
	if cfg.Qdrant.Host == "" || cfg.Ollama.URL == "" {
		return nil, nil, nil, fmt.Errorf("Qdrant and Ollama configuration required for vector services")
	}

	// Initialize Qdrant
	vectorDB, err := vector.NewQdrantDatabase(cfg.Qdrant.Host, cfg.Qdrant.Port, cfg.Qdrant.APIKey, logger)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize Qdrant database: %w", err)
	}

	// Initialize Ollama embedding model
	embeddingModel, err := vector.NewOllamaEmbedding(vector.OllamaEmbeddingConfig{
		APIURL:    cfg.Ollama.URL,
		APIKey:    cfg.Ollama.APIKey,
		Model:     cfg.Ollama.Model,
		Dimension: cfg.Ollama.Dimension,
	}, logger)
	if err != nil {
		vectorDB.Close()
		return nil, nil, nil, fmt.Errorf("failed to initialize Ollama embedding model: %w", err)
	}

	// Set default thresholds
	minConditionalLines := cfg.Chunking.MinConditionalLines
	minLoopLines := cfg.Chunking.MinLoopLines
	if minConditionalLines == 0 {
		minConditionalLines = 5
	}
	if minLoopLines == 0 {
		minLoopLines = 5
	}

	gcThreshold := cfg.App.GCThreshold
	if gcThreshold == 0 {
		gcThreshold = 100
	}

	numFileThreads := cfg.App.NumFileThreads
	if numFileThreads == 0 {
		numFileThreads = 2
	}

	// Create CodeChunkService
	chunkService := vector.NewCodeChunkService(
		vectorDB,
		embeddingModel,
		minConditionalLines,
		minLoopLines,
		gcThreshold,
		numFileThreads,
		logger,
	)

	logger.Info("Vector services initialized",
		zap.String("qdrant_host", cfg.Qdrant.Host),
		zap.Int("qdrant_port", cfg.Qdrant.Port),
		zap.String("ollama_url", cfg.Ollama.URL),
		zap.Int("min_conditional_lines", minConditionalLines),
		zap.Int("min_loop_lines", minLoopLines),
		zap.Int64("gc_threshold", gcThreshold))

	return vectorDB, embeddingModel, chunkService, nil
}

// GetIndexBuildingOptions returns ServiceInitOptions configured for index building CLI
func GetIndexBuildingOptions(cfg *config.Config) ServiceInitOptions {
	return ServiceInitOptions{
		EnableMySQL:       true,
		RequireMySQL:      true, // MySQL is required for FileID tracking
		EnableCodeGraph:   cfg.IndexBuilding.EnableCodeGraph,
		EnableEmbeddings:  cfg.IndexBuilding.EnableEmbeddings,
		EnableRepoService: cfg.IndexBuilding.EnableCodeGraph, // Only needed for CodeGraph
		EnableSummary:     cfg.IndexBuilding.EnableSummary,
	}
}

// GetServerModeOptions returns ServiceInitOptions configured for server mode
func GetServerModeOptions(cfg *config.Config) ServiceInitOptions {
	return ServiceInitOptions{
		EnableMySQL:       cfg.MySQL.Host != "",
		RequireMySQL:      false, // Optional in server mode
		EnableCodeGraph:   cfg.App.CodeGraph,
		EnableEmbeddings:  cfg.Qdrant.Host != "" && cfg.Ollama.URL != "",
		EnableRepoService: true,  // Always needed in server mode
		EnableSummary:     false, // Summary is only used during index building
	}
}

// initSummaryServices initializes the LLM service and prompt manager for summarization
func initSummaryServices(cfg *config.Config, logger *zap.Logger) (llm.LLMService, *summary.PromptManager, error) {
	// Build LLM config from summary config
	llmConfig := llm.Config{
		Provider:      llm.Provider(cfg.Summary.LLMProvider),
		Model:         cfg.Summary.LLMModel,
		MaxTokens:     500,
		Temperature:   0.3,
		OllamaURL:     cfg.Summary.OllamaURL,
		ClaudeAPIKey:  cfg.Summary.ClaudeAPIKey,
		OpenAIAPIKey:  cfg.Summary.OpenAIAPIKey,
		OpenAIBaseURL: cfg.Summary.OpenAIBaseURL,
	}

	// Use Ollama URL from main config if not set in summary config
	if llmConfig.OllamaURL == "" && cfg.Ollama.URL != "" {
		llmConfig.OllamaURL = cfg.Ollama.URL
	}

	// Default to Ollama if no provider specified
	if llmConfig.Provider == "" {
		llmConfig.Provider = llm.ProviderOllama
	}

	// Create LLM service
	llmService, err := llm.NewLLMService(llmConfig, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create LLM service: %w", err)
	}

	// Create prompt manager - prompts file is required when summary is enabled
	if cfg.Summary.PromptsFile == "" {
		return nil, nil, fmt.Errorf("summary.prompts_file is required when summary is enabled")
	}

	promptManager, err := summary.NewPromptManager(cfg.Summary.PromptsFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load prompts from %s: %w", cfg.Summary.PromptsFile, err)
	}

	logger.Info("Summary services initialized",
		zap.String("provider", string(llmConfig.Provider)),
		zap.String("model", llmConfig.Model))

	return llmService, promptManager, nil
}
