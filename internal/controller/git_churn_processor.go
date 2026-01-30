package controller

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/internal/model/ast"
	"github.com/armchr/codeapi/internal/service/codegraph"
	"go.uber.org/zap"
)

// GitChurnProcessor computes git churn metrics and stores them on graph nodes
type GitChurnProcessor struct {
	codeGraph *codegraph.CodeGraph
	config    config.GitChurnConfig
	logger    *zap.Logger

	// Cached data per repository
	repoPath string
	gitLog   *GitLogCache
}

// Ensure interface compliance
var _ FileProcessor = (*GitChurnProcessor)(nil)

// NewGitChurnProcessor creates a new GitChurnProcessor
func NewGitChurnProcessor(
	codeGraph *codegraph.CodeGraph,
	cfg *config.GitChurnConfig,
	logger *zap.Logger,
) *GitChurnProcessor {
	return &GitChurnProcessor{
		codeGraph: codeGraph,
		config:    cfg.GetDefaults(),
		logger:    logger,
	}
}

// Name returns the processor name
func (gcp *GitChurnProcessor) Name() string {
	return "GitChurn"
}

// Init initializes the processor for a repository
func (gcp *GitChurnProcessor) Init(ctx context.Context, repo *config.Repository) error {
	if !gcp.config.Enabled {
		return nil
	}

	gcp.repoPath = repo.Path
	gcp.gitLog = NewGitLogCache(repo.Path, &gcp.config, gcp.logger)
	return nil
}

// ProcessFile is a no-op for churn analysis (all work done in PostProcess)
func (gcp *GitChurnProcessor) ProcessFile(
	ctx context.Context,
	repo *config.Repository,
	fileCtx *FileContext,
) error {
	// No-op during file processing phase
	// Churn analysis happens in PostProcess after all files are indexed
	return nil
}

// PostProcess performs git churn analysis on the repository
func (gcp *GitChurnProcessor) PostProcess(ctx context.Context, repo *config.Repository) error {
	if !gcp.config.Enabled {
		return nil
	}

	gcp.logger.Info("Starting git churn analysis",
		zap.String("repo", repo.Name),
		zap.Int("timeWindowDays", gcp.config.TimeWindowDays))

	startTime := time.Now()

	// Step 1: Build git history cache
	if err := gcp.gitLog.Build(ctx); err != nil {
		return fmt.Errorf("failed to build git log cache: %w", err)
	}

	// Step 2: Process file-level churn
	if gcp.config.EnableFileLevel {
		if err := gcp.processFileLevelChurn(ctx, repo); err != nil {
			return fmt.Errorf("failed to process file-level churn: %w", err)
		}
	}

	// Step 3: Process function-level churn
	if gcp.config.EnableFunctionLevel {
		if err := gcp.processFunctionLevelChurn(ctx, repo); err != nil {
			return fmt.Errorf("failed to process function-level churn: %w", err)
		}
	}

	gcp.logger.Info("Completed git churn analysis",
		zap.String("repo", repo.Name),
		zap.Duration("duration", time.Since(startTime)))

	return nil
}

// processFileLevelChurn updates all FileScope nodes with churn metrics
func (gcp *GitChurnProcessor) processFileLevelChurn(ctx context.Context, repo *config.Repository) error {
	// Get all FileScope nodes for this repository
	fileScopes, err := gcp.codeGraph.FindFileScopes(ctx, repo.Name, "")
	if err != nil {
		return fmt.Errorf("failed to get file scopes: %w", err)
	}

	gcp.logger.Debug("Processing file-level churn",
		zap.Int("fileCount", len(fileScopes)))

	// Process files concurrently
	sem := make(chan struct{}, gcp.config.MaxConcurrency)
	var wg sync.WaitGroup
	errChan := make(chan error, len(fileScopes))

	for _, fileScope := range fileScopes {
		wg.Add(1)
		go func(fs *ast.Node) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := gcp.updateFileChurnMetrics(ctx, fs); err != nil {
				errChan <- fmt.Errorf("file %s: %w", fs.Name, err)
			}
		}(fileScope)
	}

	wg.Wait()
	close(errChan)

	// Log any errors (don't fail the whole operation)
	errorCount := 0
	for err := range errChan {
		errorCount++
		gcp.logger.Warn("File churn processing error", zap.Error(err))
	}

	if errorCount > 0 {
		gcp.logger.Warn("Some files failed churn processing",
			zap.Int("errorCount", errorCount))
	}

	return nil
}

// updateFileChurnMetrics updates a single FileScope node with churn metrics
func (gcp *GitChurnProcessor) updateFileChurnMetrics(ctx context.Context, fileScope *ast.Node) error {
	// Get relative path from file scope metadata
	relativePath, ok := fileScope.MetaData["path"].(string)
	if !ok || relativePath == "" {
		return fmt.Errorf("file scope missing path metadata")
	}

	// Look up churn data from cache
	churnData := gcp.gitLog.GetFileMetrics(relativePath)
	if churnData == nil {
		// No commits for this file in time window - set zero metrics
		churnData = &FileChurnData{Authors: make(map[string]bool)}
	}

	// Get current LOC for density calculation
	loc := gcp.getFileLineCount(fileScope)

	// Calculate composite score
	linesChanged := churnData.LinesAdded + churnData.LinesDeleted
	authorCount := len(churnData.Authors)

	score := float64(linesChanged)*gcp.config.Weights.LinesChanged +
		float64(churnData.CommitCount)*gcp.config.Weights.CommitCount +
		float64(authorCount)*gcp.config.Weights.AuthorCount

	density := 0.0
	if loc > 0 {
		density = score / float64(loc)
	}

	// Prepare metadata updates
	metadata := map[string]any{
		"churn_lines_added":   churnData.LinesAdded,
		"churn_lines_deleted": churnData.LinesDeleted,
		"churn_lines_changed": linesChanged,
		"churn_commit_count":  churnData.CommitCount,
		"churn_author_count":  authorCount,
		"churn_score":         score,
		"churn_density":       density,
		"churn_window_days":   gcp.config.TimeWindowDays,
	}

	if !churnData.FirstCommit.IsZero() {
		metadata["churn_first_commit"] = churnData.FirstCommit.Format(time.RFC3339)
		metadata["churn_last_commit"] = churnData.LastCommit.Format(time.RFC3339)
	}

	// Average change size
	if churnData.CommitCount > 0 {
		metadata["churn_avg_change_size"] = float64(linesChanged) / float64(churnData.CommitCount)
	}

	// Update node in Neo4j
	return gcp.codeGraph.UpdateNodeMetaData(ctx, fileScope.ID, fileScope.FileID, metadata)
}

// getFileLineCount gets the line count for a file from its range or metadata
func (gcp *GitChurnProcessor) getFileLineCount(fileScope *ast.Node) int {
	// Try to get from existing metadata
	if loc, ok := fileScope.MetaData["line_count"].(int); ok {
		return loc
	}

	// Calculate from range if available
	if fileScope.Range.End.Line > 0 {
		return fileScope.Range.End.Line
	}

	return 0
}

// processFunctionLevelChurn updates Function nodes with churn metrics
// Only processes files with high churn (hybrid approach for performance)
func (gcp *GitChurnProcessor) processFunctionLevelChurn(ctx context.Context, repo *config.Repository) error {
	// Get all files with high churn
	highChurnFiles, err := gcp.getHighChurnFiles(ctx, repo)
	if err != nil {
		return fmt.Errorf("failed to identify high churn files: %w", err)
	}

	if len(highChurnFiles) == 0 {
		gcp.logger.Debug("No high churn files found for function-level analysis")
		return nil
	}

	gcp.logger.Debug("Processing function-level churn for high-churn files",
		zap.Int("fileCount", len(highChurnFiles)))

	for _, filePath := range highChurnFiles {
		if err := gcp.processFunctionsInFile(ctx, repo, filePath); err != nil {
			gcp.logger.Warn("Failed to process functions in file",
				zap.String("file", filePath),
				zap.Error(err))
		}
	}

	return nil
}

// getHighChurnFiles returns files with churn score in the top N%
func (gcp *GitChurnProcessor) getHighChurnFiles(ctx context.Context, repo *config.Repository) ([]string, error) {
	// Get all file metrics from the cache
	allMetrics := gcp.gitLog.GetAllFileMetrics()
	if len(allMetrics) == 0 {
		return nil, nil
	}

	// Calculate scores and sort
	type fileScore struct {
		path  string
		score float64
	}

	scores := make([]fileScore, 0, len(allMetrics))
	for path, metrics := range allMetrics {
		linesChanged := metrics.LinesAdded + metrics.LinesDeleted
		authorCount := len(metrics.Authors)
		score := float64(linesChanged)*gcp.config.Weights.LinesChanged +
			float64(metrics.CommitCount)*gcp.config.Weights.CommitCount +
			float64(authorCount)*gcp.config.Weights.AuthorCount
		scores = append(scores, fileScore{path: path, score: score})
	}

	// Sort by score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// Take top N%
	threshold := int(float64(len(scores)) * (gcp.config.FunctionChurnThreshold / 100.0))
	if threshold < 1 {
		threshold = 1
	}

	result := make([]string, 0, threshold)
	for i := 0; i < threshold && i < len(scores); i++ {
		result = append(result, scores[i].path)
	}

	return result, nil
}

// processFunctionsInFile processes all functions in a file for churn metrics
func (gcp *GitChurnProcessor) processFunctionsInFile(ctx context.Context, repo *config.Repository, filePath string) error {
	// Get all Function nodes in this file
	functions, err := gcp.codeGraph.FindFunctionsByFilePath(ctx, repo.Name, filePath)
	if err != nil {
		return err
	}

	if len(functions) == 0 {
		return nil
	}

	// Get detailed diff data for this file
	diffData, err := gcp.gitLog.BuildDiffData(ctx, filePath)
	if err != nil {
		return err
	}

	for _, fn := range functions {
		metrics := gcp.attributeChangesToFunction(fn, diffData)
		if err := gcp.updateFunctionChurnMetrics(ctx, fn, metrics); err != nil {
			gcp.logger.Warn("Failed to update function churn",
				zap.String("function", fn.Name),
				zap.Error(err))
		}
	}

	return nil
}

// FunctionChurnMetrics holds churn metrics for a function
type FunctionChurnMetrics struct {
	LinesAdded   int
	LinesDeleted int
	CommitCount  int
	Authors      map[string]bool
}

// attributeChangesToFunction calculates churn metrics for a function based on diff data
func (gcp *GitChurnProcessor) attributeChangesToFunction(fn *ast.Node, diffData *FileDiffData) *FunctionChurnMetrics {
	metrics := &FunctionChurnMetrics{
		Authors: make(map[string]bool),
	}

	fnStartLine := fn.Range.Start.Line
	fnEndLine := fn.Range.End.Line

	// For each commit that touched this file
	for _, commit := range diffData.Commits {
		touchedFunction := false

		for _, hunk := range commit.Hunks {
			// Check if hunk overlaps with function range
			// We use the NEW file line numbers since that's the current state
			hunkStart := hunk.NewStart
			hunkEnd := hunk.NewStart + hunk.NewCount

			if overlaps(fnStartLine, fnEndLine, hunkStart, hunkEnd) {
				touchedFunction = true

				// Count lines within function range
				overlapStart := maxInt(fnStartLine, hunkStart)
				overlapEnd := minInt(fnEndLine, hunkEnd)
				overlapLines := overlapEnd - overlapStart

				// Proportionally attribute added/deleted lines
				if hunk.NewCount > 0 {
					ratio := float64(overlapLines) / float64(hunk.NewCount)
					metrics.LinesAdded += int(float64(hunk.LinesAdded) * ratio)
					metrics.LinesDeleted += int(float64(hunk.LinesDeleted) * ratio)
				}
			}
		}

		if touchedFunction {
			metrics.CommitCount++
			metrics.Authors[commit.Author] = true
		}
	}

	return metrics
}

// updateFunctionChurnMetrics updates a function node with churn metrics
func (gcp *GitChurnProcessor) updateFunctionChurnMetrics(ctx context.Context, fn *ast.Node, metrics *FunctionChurnMetrics) error {
	// Calculate composite score
	linesChanged := metrics.LinesAdded + metrics.LinesDeleted
	authorCount := len(metrics.Authors)

	score := float64(linesChanged)*gcp.config.Weights.LinesChanged +
		float64(metrics.CommitCount)*gcp.config.Weights.CommitCount +
		float64(authorCount)*gcp.config.Weights.AuthorCount

	// Calculate function LOC for density
	fnLOC := fn.Range.End.Line - fn.Range.Start.Line + 1
	density := 0.0
	if fnLOC > 0 {
		density = score / float64(fnLOC)
	}

	// Prepare metadata updates
	metadata := map[string]any{
		"churn_lines_added":   metrics.LinesAdded,
		"churn_lines_deleted": metrics.LinesDeleted,
		"churn_lines_changed": linesChanged,
		"churn_commit_count":  metrics.CommitCount,
		"churn_author_count":  authorCount,
		"churn_score":         score,
		"churn_density":       density,
	}

	return gcp.codeGraph.UpdateNodeMetaData(ctx, fn.ID, fn.FileID, metadata)
}

// Helper functions
func overlaps(aStart, aEnd, bStart, bEnd int) bool {
	return aStart <= bEnd && bStart <= aEnd
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
