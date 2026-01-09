package controller

import (
	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/internal/db"
	"github.com/armchr/codeapi/internal/service/vector"
	"github.com/armchr/codeapi/internal/util"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/armchr/codeapi/internal/model"
	"github.com/armchr/codeapi/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type RepoController struct {
	repoService  *service.RepoService
	chunkService *vector.CodeChunkService
	processors   []FileProcessor
	mysqlConn    *db.MySQLConnection
	config       *config.Config
	logger       *zap.Logger
}

func NewRepoController(repoService *service.RepoService, chunkService *vector.CodeChunkService, processors []FileProcessor, mysqlConn *db.MySQLConnection, config *config.Config, logger *zap.Logger) *RepoController {
	return &RepoController{
		repoService:  repoService,
		chunkService: chunkService,
		processors:   processors,
		mysqlConn:    mysqlConn,
		config:       config,
		logger:       logger,
	}
}

type BuildIndexRequest struct {
	RepoName string `json:"repo_name" binding:"required"`
	UseHead  bool   `json:"use_head"` // Use git HEAD version instead of working directory
}

type BuildIndexResponse struct {
	RepoName string `json:"repo_name"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

func (rc *RepoController) BuildIndex(c *gin.Context) {
	var request BuildIndexRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		rc.logger.Error("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
		return
	}

	rc.logger.Info("Processing repository",
		zap.String("repo_name", request.RepoName),
		zap.Bool("use_head", request.UseHead))

	ctx := c.Request.Context()

	// Validate repository exists in config
	repo, err := rc.config.GetRepository(request.RepoName)
	if err != nil {
		rc.logger.Error("Repository not found in configuration",
			zap.String("repo_name", request.RepoName),
			zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Repository not found",
			"details": err.Error(),
		})
		return
	}

	// Check if MySQL connection is available
	if rc.mysqlConn == nil {
		rc.logger.Error("MySQL connection not available")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "MySQL connection not available for file tracking",
		})
		return
	}

	// Create FileVersionRepository for this repository
	fileVersionRepo, err := db.NewFileVersionRepository(rc.mysqlConn.GetDB(), repo.Name, rc.logger)
	if err != nil {
		rc.logger.Error("Failed to create file version repository",
			zap.String("repo_name", repo.Name),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to initialize file tracking",
			"details": err.Error(),
		})
		return
	}

	// Create index builder with processors
	indexBuilder := NewIndexBuilder(rc.config, rc.processors, fileVersionRepo, rc.logger)

	// Get git info if using HEAD mode
	var gitInfo *util.GitInfo
	if request.UseHead {
		gitInfo, err = util.GetGitInfo(repo.Path)
		if err != nil {
			rc.logger.Error("Failed to get git info",
				zap.String("repo_name", repo.Name),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to get git information",
				"details": err.Error(),
			})
			return
		}
		if !gitInfo.IsGitRepo {
			rc.logger.Error("Repository is not a git repository, cannot use use_head flag",
				zap.String("repo_name", repo.Name),
				zap.String("path", repo.Path))
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Repository is not a git repository, cannot use use_head flag",
			})
			return
		}
	}

	// Build indexes
	if err := indexBuilder.BuildIndexWithGitInfo(ctx, repo, request.UseHead, gitInfo); err != nil {
		rc.logger.Error("Failed to build indexes for repository",
			zap.String("repo_name", repo.Name),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process repository",
			"details": err.Error(),
		})
		return
	}

	rc.logger.Info("Successfully processed repository",
		zap.String("repo_name", repo.Name),
		zap.Bool("use_head", request.UseHead))

	c.JSON(http.StatusOK, BuildIndexResponse{
		RepoName: repo.Name,
		Status:   "completed",
		Message:  "Repository indexed successfully",
	})
}

func (rc *RepoController) GetFunctionsInFile(c *gin.Context) {
	var request model.GetFunctionsInFileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		rc.logger.Error("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
		return
	}

	rc.logger.Info("Getting functions in file",
		zap.String("repo_name", request.RepoName),
		zap.String("relative_path", request.RelativePath))

	/*response, err := rc.repoService.GetFunctionsInFile(request.RepoName, request.RelativePath)
	if err != nil {
		rc.logger.Error("Failed to get functions in file",
			zap.String("repo_name", request.RepoName),
			zap.String("relative_path", request.RelativePath),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get functions in file",
			"details": err.Error(),
		})
		return
	}

	rc.logger.Info("Successfully got functions in file",
		zap.String("repo_name", request.RepoName),
		zap.String("relative_path", request.RelativePath),
		zap.Int("function_count", len(response.Functions)))

	rc.logger.Debug("About to send JSON response")
	c.JSON(http.StatusOK, response)
	*/
	c.JSON(http.StatusOK, nil)
	rc.logger.Debug("JSON response sent successfully")
}

func (rc *RepoController) GetFunctionDetails(c *gin.Context) {
	var request model.GetFunctionDetailsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		rc.logger.Error("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
		return
	}

	rc.logger.Info("Getting function details",
		zap.String("repo_name", request.RepoName),
		zap.String("relative_path", request.RelativePath),
		zap.String("function_name", request.FunctionName))

	response, err := rc.repoService.GetFunctionDetails(request.RepoName, request.RelativePath, request.FunctionName)
	if err != nil {
		rc.logger.Error("Failed to get function details",
			zap.String("repo_name", request.RepoName),
			zap.String("relative_path", request.RelativePath),
			zap.String("function_name", request.FunctionName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get function details",
			"details": err.Error(),
		})
		return
	}

	rc.logger.Info("Successfully got function details",
		zap.String("repo_name", request.RepoName),
		zap.String("relative_path", request.RelativePath),
		zap.String("function_name", request.FunctionName))

	rc.logger.Debug("About to send JSON response")
	c.JSON(http.StatusOK, response)
	rc.logger.Debug("JSON response sent successfully")
}

func (rc *RepoController) GetFunctionDependencies(c *gin.Context) {
	request := model.GetFunctionDependenciesRequest{
		Depth: 2, // Default depth
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		rc.logger.Error("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
		return
	}

	rc.logger.Info("Getting function dependencies",
		zap.String("repo_name", request.RepoName),
		zap.String("relative_path", request.RelativePath),
		zap.String("function_name", request.FunctionName),
		zap.Int("depth", request.Depth))

	response, err := rc.repoService.GetFunctionDependencies(c, request.RepoName, request.RelativePath, request.FunctionName, request.Depth)
	if err != nil {
		rc.logger.Error("Failed to get function dependencies",
			zap.String("repo_name", request.RepoName),
			zap.String("relative_path", request.RelativePath),
			zap.String("function_name", request.FunctionName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get function dependencies",
			"details": err.Error(),
		})
		return
	}

	rc.logger.Info("Successfully got function dependencies",
		zap.String("repo_name", request.RepoName),
		zap.String("relative_path", request.RelativePath),
		zap.String("function_name", request.FunctionName))

	rc.logger.Debug("About to send JSON response")
	c.JSON(http.StatusOK, response)
	rc.logger.Debug("JSON response sent successfully")
}

func (rc *RepoController) ProcessDirectory(c *gin.Context) {
	var request model.ProcessDirectoryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		rc.logger.Error("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
		return
	}

	// Check if chunk service is available
	if rc.chunkService == nil {
		rc.logger.Error("Code chunk service not available")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Code chunk service not available",
		})
		return
	}

	// Get repository configuration
	repo, err := rc.repoService.GetConfig().GetRepository(request.RepoName)
	if err != nil {
		rc.logger.Error("Repository not found",
			zap.String("repo_name", request.RepoName),
			zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Repository not found",
			"details": err.Error(),
		})
		return
	}

	// Use repo name as collection name if not provided
	collectionName := request.CollectionName
	if collectionName == "" {
		collectionName = request.RepoName
	}

	rc.logger.Info("Processing directory for code chunking",
		zap.String("repo_name", request.RepoName),
		zap.String("path", repo.Path),
		zap.String("collection", collectionName))

	// Create collection if it doesn't exist
	if err := rc.chunkService.CreateCollection(c.Request.Context(), collectionName); err != nil {
		rc.logger.Error("Failed to create collection",
			zap.String("collection", collectionName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create collection",
			"details": err.Error(),
		})
		return
	}

	// Process directory with repository configuration
	totalChunks, err := rc.chunkService.ProcessDirectory(c.Request.Context(), repo.Path, collectionName, repo)
	if err != nil {
		rc.logger.Error("Failed to process directory",
			zap.String("repo_name", request.RepoName),
			zap.String("path", repo.Path),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.ProcessDirectoryResponse{
			RepoName:       request.RepoName,
			CollectionName: collectionName,
			TotalChunks:    totalChunks,
			Success:        false,
			Message:        fmt.Sprintf("Failed to process directory: %v", err),
		})
		return
	}

	rc.logger.Info("Successfully processed directory",
		zap.String("repo_name", request.RepoName),
		zap.String("collection", collectionName),
		zap.Int("total_chunks", totalChunks))

	response := model.ProcessDirectoryResponse{
		RepoName:       request.RepoName,
		CollectionName: collectionName,
		TotalChunks:    totalChunks,
		Success:        true,
		Message:        "Directory processed successfully",
	}

	c.JSON(http.StatusOK, response)
}

// SearchSimilarCode handles searching for similar code using a code snippet
func (rc *RepoController) SearchSimilarCode(c *gin.Context) {
	var request model.SearchSimilarCodeRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		rc.logger.Error("Invalid request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Check if chunk service is available
	if rc.chunkService == nil {
		rc.logger.Error("Code chunk service not available")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Code chunk service not available",
		})
		return
	}

	// Validate language
	validLanguages := map[string]bool{
		"go":         true,
		"python":     true,
		"java":       true,
		"javascript": true,
		"typescript": true,
	}
	if !validLanguages[request.Language] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unsupported language. Supported: go, python, java, javascript, typescript",
		})
		return
	}

	// Use repo name as collection name if not provided
	collectionName := request.CollectionName
	if collectionName == "" {
		collectionName = request.RepoName
	}

	// Set default limit
	limit := request.Limit
	if limit <= 0 {
		limit = 10
	}

	rc.logger.Info("Searching for similar code",
		zap.String("repo_name", request.RepoName),
		zap.String("collection", collectionName),
		zap.String("language", request.Language),
		zap.Int("limit", limit))

	// Search for similar code
	queryChunks, resultChunks, scores, queryChunkIndices, err := rc.chunkService.SearchSimilarCodeBySnippet(
		c.Request.Context(),
		collectionName,
		request.CodeSnippet,
		request.Language,
		limit,
		nil, // no filter
	)
	if err != nil {
		rc.logger.Error("Failed to search for similar code",
			zap.String("repo_name", request.RepoName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.SearchSimilarCodeResponse{
			RepoName:       request.RepoName,
			CollectionName: collectionName,
			Query: model.QueryInfo{
				CodeSnippet: request.CodeSnippet,
				Language:    request.Language,
				ChunksFound: 0,
			},
			Results: []model.SimilarCodeResult{},
			Success: false,
			Message: fmt.Sprintf("Failed to search: %v", err),
		})
		return
	}

	// Build results
	results := make([]model.SimilarCodeResult, len(resultChunks))
	for i, chunk := range resultChunks {
		result := model.SimilarCodeResult{
			Chunk:           chunk,
			Score:           scores[i],
			QueryChunkIndex: queryChunkIndices[i],
		}

		// Fetch code from file if requested
		if request.IncludeCode {
			code, err := rc.chunkService.ReadCodeFromFile(chunk.FilePath, chunk.StartLine, chunk.EndLine)
			if err != nil {
				rc.logger.Warn("Failed to read code from file",
					zap.String("file", chunk.FilePath),
					zap.Int("start_line", chunk.StartLine),
					zap.Int("end_line", chunk.EndLine),
					zap.Error(err))
				// Continue without code rather than failing the entire request
			} else {
				result.Code = code
			}
		}

		results[i] = result
	}

	rc.logger.Info("Successfully found similar code",
		zap.String("repo_name", request.RepoName),
		zap.String("collection", collectionName),
		zap.Int("query_chunks", len(queryChunks)),
		zap.Int("results", len(results)),
		zap.Bool("include_code", request.IncludeCode))

	response := model.SearchSimilarCodeResponse{
		RepoName:       request.RepoName,
		CollectionName: collectionName,
		Query: model.QueryInfo{
			CodeSnippet: request.CodeSnippet,
			Language:    request.Language,
			ChunksFound: len(queryChunks),
			Chunks:      queryChunks,
		},
		Results: results,
		Success: true,
		Message: "Search completed successfully",
	}

	c.JSON(http.StatusOK, response)
}

// SearchMethodsBySignatureRequest represents the request for semantic signature search
type SearchMethodsBySignatureRequest struct {
	RepoName string `json:"repo_name" binding:"required"`
	Query    string `json:"query" binding:"required"` // Natural language query like "find user by email"
	Limit    int    `json:"limit"`                    // Max results (default 10)
}

// SearchMethodsBySignatureResponse represents the response from signature search
type SearchMethodsBySignatureResponse struct {
	RepoName string                   `json:"repo_name"`
	Query    string                   `json:"query"`
	Results  []MethodSignatureResult  `json:"results"`
	Success  bool                     `json:"success"`
	Message  string                   `json:"message,omitempty"`
}

// MethodSignatureResult represents a single method found by signature search
type MethodSignatureResult struct {
	MethodName     string   `json:"method_name"`
	ClassName      string   `json:"class_name,omitempty"`
	Signature      string   `json:"signature"`
	ReturnType     string   `json:"return_type,omitempty"`
	ParameterTypes []string `json:"parameter_types,omitempty"`
	ParameterNames []string `json:"parameter_names,omitempty"`
	FilePath       string   `json:"file_path"`
	StartLine      int      `json:"start_line"`
	EndLine        int      `json:"end_line"`
	Score          float32  `json:"score"`
	NormalizedText string   `json:"normalized_text,omitempty"` // The normalized text used for embedding
}

// SearchMethodsBySignature searches for methods using natural language queries on signatures
func (rc *RepoController) SearchMethodsBySignature(c *gin.Context) {
	var request SearchMethodsBySignatureRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		rc.logger.Error("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
		return
	}

	// Check if chunk service is available
	if rc.chunkService == nil {
		rc.logger.Error("Code chunk service not available")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Code chunk service not available",
		})
		return
	}

	// Set default limit
	limit := request.Limit
	if limit <= 0 {
		limit = 10
	}

	// Use repo name as collection name
	collectionName := request.RepoName

	rc.logger.Info("Searching methods by signature",
		zap.String("repo_name", request.RepoName),
		zap.String("query", request.Query),
		zap.Int("limit", limit))

	// Search using the chunk service
	chunks, scores, err := rc.chunkService.SearchMethodSignatures(
		c.Request.Context(),
		collectionName,
		request.Query,
		limit,
	)
	if err != nil {
		rc.logger.Error("Failed to search method signatures",
			zap.String("repo_name", request.RepoName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, SearchMethodsBySignatureResponse{
			RepoName: request.RepoName,
			Query:    request.Query,
			Results:  []MethodSignatureResult{},
			Success:  false,
			Message:  fmt.Sprintf("Search failed: %v", err),
		})
		return
	}

	// Build results
	results := make([]MethodSignatureResult, len(chunks))
	for i, chunk := range chunks {
		result := MethodSignatureResult{
			MethodName: chunk.Name,
			ClassName:  chunk.ClassName,
			Signature:  chunk.Signature,
			FilePath:   chunk.FilePath,
			StartLine:  chunk.StartLine,
			EndLine:    chunk.EndLine,
			Score:      scores[i],
		}

		// Extract metadata if available
		if chunk.Metadata != nil {
			if rt, ok := chunk.Metadata["return_type"].(string); ok {
				result.ReturnType = rt
			}
			if pt, ok := chunk.Metadata["parameter_types"].([]string); ok {
				result.ParameterTypes = pt
			} else if ptInterface, ok := chunk.Metadata["parameter_types"].([]interface{}); ok {
				for _, p := range ptInterface {
					if ps, ok := p.(string); ok {
						result.ParameterTypes = append(result.ParameterTypes, ps)
					}
				}
			}
			if pn, ok := chunk.Metadata["parameter_names"].([]string); ok {
				result.ParameterNames = pn
			} else if pnInterface, ok := chunk.Metadata["parameter_names"].([]interface{}); ok {
				for _, p := range pnInterface {
					if ps, ok := p.(string); ok {
						result.ParameterNames = append(result.ParameterNames, ps)
					}
				}
			}
			if nt, ok := chunk.Metadata["normalized_text"].(string); ok {
				result.NormalizedText = nt
			}
		}

		results[i] = result
	}

	rc.logger.Info("Successfully found methods by signature",
		zap.String("repo_name", request.RepoName),
		zap.String("query", request.Query),
		zap.Int("results", len(results)))

	c.JSON(http.StatusOK, SearchMethodsBySignatureResponse{
		RepoName: request.RepoName,
		Query:    request.Query,
		Results:  results,
		Success:  true,
		Message:  fmt.Sprintf("Found %d matching methods", len(results)),
	})
}

// IndexFileRequest represents the request to index a single file
type IndexFileRequest struct {
	RepoName      string   `json:"repo_name" binding:"required"`
	RelativePaths []string `json:"relative_paths" binding:"required"`
}

// IndexFileResponse represents the response after indexing files
type IndexFileResponse struct {
	RepoName string              `json:"repo_name"`
	Files    []IndexedFileResult `json:"files"`
	Message  string              `json:"message"`
}

// IndexedFileResult represents the result of indexing a single file
type IndexedFileResult struct {
	RelativePath string   `json:"relative_path"`
	FileID       int32    `json:"file_id,omitempty"`
	FileSHA      string   `json:"file_sha,omitempty"`
	Processors   []string `json:"processors_run,omitempty"`
	Success      bool     `json:"success"`
	Error        string   `json:"error,omitempty"`
}

// IndexFile indexes multiple files through all registered processors in parallel
func (rc *RepoController) IndexFile(c *gin.Context) {
	var request IndexFileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		rc.logger.Error("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
		return
	}

	// Validate that we have files to process
	if len(request.RelativePaths) == 0 {
		rc.logger.Error("No files specified in request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No files specified. Please provide at least one file path.",
		})
		return
	}

	// Check if processors are available
	if len(rc.processors) == 0 {
		rc.logger.Error("No processors available - processors may not be enabled")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "No processors available. Ensure processors are enabled in configuration.",
		})
		return
	}

	// Check if MySQL is available (needed for file version tracking)
	if rc.mysqlConn == nil {
		rc.logger.Error("MySQL connection not available")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "MySQL connection not available. File indexing requires MySQL.",
		})
		return
	}

	ctx := c.Request.Context()

	// Get repository configuration
	repo, err := rc.config.GetRepository(request.RepoName)
	if err != nil {
		rc.logger.Error("Repository not found", zap.String("repo_name", request.RepoName), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Repository not found",
			"details": err.Error(),
		})
		return
	}

	// Create FileVersionRepository for this repository (shared across all files)
	fileVersionRepo, err := db.NewFileVersionRepository(rc.mysqlConn.GetDB(), repo.Name, rc.logger)
	if err != nil {
		rc.logger.Error("Failed to create file version repository",
			zap.String("repo_name", repo.Name),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create file version repository",
			"details": err.Error(),
		})
		return
	}

	// Get concurrency limit from config, default to 5
	maxConcurrent := rc.config.App.MaxConcurrentFileProcessing
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}

	rc.logger.Info("Starting parallel file indexing",
		zap.String("repo_name", request.RepoName),
		zap.Int("file_count", len(request.RelativePaths)),
		zap.Int("max_concurrent", maxConcurrent))

	// Process files in parallel using worker pool
	results := rc.processFilesInParallel(ctx, repo, request.RelativePaths, fileVersionRepo, maxConcurrent)

	// Count successes and failures
	successCount := 0
	failureCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	rc.logger.Info("Completed parallel file indexing",
		zap.String("repo_name", request.RepoName),
		zap.Int("total_files", len(request.RelativePaths)),
		zap.Int("successes", successCount),
		zap.Int("failures", failureCount))

	response := IndexFileResponse{
		RepoName: request.RepoName,
		Files:    results,
		Message:  fmt.Sprintf("Processed %d file(s): %d succeeded, %d failed", len(results), successCount, failureCount),
	}

	c.JSON(http.StatusOK, response)
}

// processFilesInParallel processes multiple files concurrently using a worker pool
func (rc *RepoController) processFilesInParallel(ctx context.Context, repo *config.Repository, relativePaths []string, fileVersionRepo *db.FileVersionRepository, maxConcurrent int) []IndexedFileResult {
	type fileJob struct {
		relativePath string
		index        int
	}

	// Create channels
	jobs := make(chan fileJob, len(relativePaths))
	results := make(chan IndexedFileResult, len(relativePaths))

	// Start worker goroutines
	for w := 0; w < maxConcurrent; w++ {
		go func(workerID int) {
			for job := range jobs {
				rc.logger.Debug("Worker processing file",
					zap.Int("worker_id", workerID),
					zap.String("file", job.relativePath))

				result := rc.processSingleFile(ctx, repo, job.relativePath, fileVersionRepo)
				results <- result
			}
		}(w)
	}

	// Send jobs to workers
	for i, relativePath := range relativePaths {
		jobs <- fileJob{relativePath: relativePath, index: i}
	}
	close(jobs)

	// Collect results
	fileResults := make([]IndexedFileResult, len(relativePaths))
	for i := 0; i < len(relativePaths); i++ {
		result := <-results
		fileResults[i] = result
	}

	return fileResults
}

// processSingleFile processes a single file through all processors
func (rc *RepoController) processSingleFile(ctx context.Context, repo *config.Repository, relativePath string, fileVersionRepo *db.FileVersionRepository) IndexedFileResult {
	// Build absolute file path
	filePath := relativePath
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(repo.Path, relativePath)
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		rc.logger.Error("File not found", zap.String("file_path", filePath))
		return IndexedFileResult{
			RelativePath: relativePath,
			Success:      false,
			Error:        fmt.Sprintf("File does not exist: %s", relativePath),
		}
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		rc.logger.Error("Failed to read file", zap.String("file_path", filePath), zap.Error(err))
		return IndexedFileResult{
			RelativePath: relativePath,
			Success:      false,
			Error:        fmt.Sprintf("Failed to read file: %v", err),
		}
	}

	// Calculate file SHA256
	fileSHA := util.CalculateFileSHA256(content)

	// Get or create FileID from MySQL
	fileID, err := fileVersionRepo.GetOrCreateFileID(fileSHA, relativePath, true, nil)
	if err != nil {
		rc.logger.Error("Failed to create file ID", zap.String("file_path", filePath), zap.Error(err))
		return IndexedFileResult{
			RelativePath: relativePath,
			Success:      false,
			Error:        fmt.Sprintf("Failed to create file ID: %v", err),
		}
	}

	// Create FileContext
	fileCtx := &FileContext{
		FileID:       fileID,
		FilePath:     filePath,
		RelativePath: relativePath,
		Content:      content,
		FileSHA:      fileSHA,
		CommitID:     nil,
		Ephemeral:    true,
	}

	// Process through all processors
	processorsRun := []string{}
	for _, processor := range rc.processors {
		rc.logger.Debug("Processing file with processor",
			zap.String("processor", processor.Name()),
			zap.String("file_path", relativePath),
			zap.Int32("file_id", fileID))

		err := processor.ProcessFile(ctx, repo, fileCtx)
		if err != nil {
			rc.logger.Error("Processor failed to process file",
				zap.String("processor", processor.Name()),
				zap.String("file_path", filePath),
				zap.Error(err))
			return IndexedFileResult{
				RelativePath: relativePath,
				FileID:       fileID,
				FileSHA:      fileSHA,
				Success:      false,
				Error:        fmt.Sprintf("Processor '%s' failed: %v", processor.Name(), err),
			}
		}

		processorsRun = append(processorsRun, processor.Name())

		// Update status to indicate this processor completed
		processorStatus := fmt.Sprintf("%s_done", processor.Name())
		if err := fileVersionRepo.UpdateStatus(fileID, processorStatus); err != nil {
			rc.logger.Warn("Failed to update processor status",
				zap.String("processor", processor.Name()),
				zap.Int32("file_id", fileID),
				zap.Error(err))
		}
	}

	// Mark file as fully processed
	if err := fileVersionRepo.UpdateStatus(fileID, "done"); err != nil {
		rc.logger.Warn("Failed to update final status",
			zap.Int32("file_id", fileID),
			zap.Error(err))
	}

	rc.logger.Info("Successfully indexed file",
		zap.String("repo_name", repo.Name),
		zap.String("relative_path", relativePath),
		zap.Int32("file_id", fileID),
		zap.Strings("processors", processorsRun))

	return IndexedFileResult{
		RelativePath: relativePath,
		FileID:       fileID,
		FileSHA:      fileSHA,
		Processors:   processorsRun,
		Success:      true,
	}
}
