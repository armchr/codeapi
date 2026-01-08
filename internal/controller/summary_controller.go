package controller

import (
	"database/sql"
	"net/http"

	"github.com/armchr/codeapi/internal/db"
	"github.com/armchr/codeapi/internal/service/summary"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SummaryController handles HTTP requests for code summary queries
type SummaryController struct {
	mysqlDB *sql.DB
	logger  *zap.Logger
}

// NewSummaryController creates a new SummaryController
func NewSummaryController(mysqlDB *sql.DB, logger *zap.Logger) *SummaryController {
	return &SummaryController{
		mysqlDB: mysqlDB,
		logger:  logger,
	}
}

// -----------------------------------------------------------------------------
// Request/Response Types
// -----------------------------------------------------------------------------

// GetFileSummariesRequest is the request for getting all summaries for a file
type GetFileSummariesRequest struct {
	RepoName   string `json:"repo_name" binding:"required"`
	FilePath   string `json:"file_path" binding:"required"`
	EntityType string `json:"entity_type"` // Optional: "function", "class", or empty for all
}

// GetFileSummariesResponse is the response for GetFileSummaries
type GetFileSummariesResponse struct {
	FilePath  string                  `json:"file_path"`
	Summaries []*summary.CodeSummary `json:"summaries"`
	Count     int                     `json:"count"`
}

// GetEntitySummaryRequest is the request for getting a specific entity summary
type GetEntitySummaryRequest struct {
	RepoName   string `json:"repo_name" binding:"required"`
	FilePath   string `json:"file_path" binding:"required"`
	EntityType string `json:"entity_type" binding:"required"` // "function" or "class"
	EntityName string `json:"entity_name" binding:"required"`
}

// GetFileSummaryRequest is the request for getting a file-level summary
type GetFileSummaryRequest struct {
	RepoName string `json:"repo_name" binding:"required"`
	FilePath string `json:"file_path" binding:"required"`
}

// GetSummaryStatsRequest is the request for getting summary statistics
type GetSummaryStatsRequest struct {
	RepoName string `json:"repo_name" binding:"required"`
}

// GetSummaryStatsResponse is the response for GetSummaryStats
type GetSummaryStatsResponse struct {
	RepoName string           `json:"repo_name"`
	Stats    *db.SummaryStats `json:"stats"`
}

// -----------------------------------------------------------------------------
// Handlers
// -----------------------------------------------------------------------------

// getStore returns a SummaryStore for the given repository
func (c *SummaryController) getStore(repoName string) (*db.SummaryStore, error) {
	return db.NewSummaryStore(c.mysqlDB, repoName, c.logger)
}

// GetFileSummaries returns all summaries for a file, optionally filtered by entity type
func (c *SummaryController) GetFileSummaries(ctx *gin.Context) {
	var req GetFileSummariesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	store, err := c.getStore(req.RepoName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access summary store: " + err.Error()})
		return
	}

	var summaries []*summary.CodeSummary

	if req.EntityType != "" {
		// Filter by entity type
		entityType := summary.ParseSummaryLevel(req.EntityType)
		if entityType == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid entity_type: must be 'function', 'class', 'file', 'folder', or 'project'"})
			return
		}
		summaries, err = store.GetSummariesByFileAndType(req.FilePath, entityType)
	} else {
		// Get all summaries for the file
		summaries, err = store.GetSummariesByFile(req.FilePath)
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query summaries: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, GetFileSummariesResponse{
		FilePath:  req.FilePath,
		Summaries: summaries,
		Count:     len(summaries),
	})
}

// GetEntitySummary returns a specific function or class summary
func (c *SummaryController) GetEntitySummary(ctx *gin.Context) {
	var req GetEntitySummaryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entityType := summary.ParseSummaryLevel(req.EntityType)
	if entityType == 0 || (entityType != summary.LevelFunction && entityType != summary.LevelClass) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid entity_type: must be 'function' or 'class'"})
		return
	}

	store, err := c.getStore(req.RepoName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access summary store: " + err.Error()})
		return
	}

	result, err := store.GetSummaryByFileAndName(req.FilePath, entityType, req.EntityName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query summary: " + err.Error()})
		return
	}

	if result == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "summary not found"})
		return
	}

	ctx.JSON(http.StatusOK, result)
}

// GetFileSummary returns the file-level summary for a file
func (c *SummaryController) GetFileSummary(ctx *gin.Context) {
	var req GetFileSummaryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	store, err := c.getStore(req.RepoName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access summary store: " + err.Error()})
		return
	}

	result, err := store.GetFileSummary(req.FilePath)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query file summary: " + err.Error()})
		return
	}

	if result == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "file summary not found"})
		return
	}

	ctx.JSON(http.StatusOK, result)
}

// GetSummaryStats returns statistics about summaries for a repository
func (c *SummaryController) GetSummaryStats(ctx *gin.Context) {
	var req GetSummaryStatsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	store, err := c.getStore(req.RepoName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access summary store: " + err.Error()})
		return
	}

	stats, err := store.GetStats()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get summary stats: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, GetSummaryStatsResponse{
		RepoName: req.RepoName,
		Stats:    stats,
	})
}
