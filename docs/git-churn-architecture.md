# Git Churn Metrics: Architecture and Implementation Plan

**Version:** 1.0
**Date:** January 2026
**Status:** Design
**Author:** Claude Code

---

## 1. Executive Summary

This document outlines the architecture and implementation plan for adding git churn metrics to CodeAPI. Git churn measures how frequently and substantially code changes over time, serving as an indicator of technical debt and code hotspots.

### 1.1 Scope

- **File-level churn**: Metrics aggregated per file
- **Function-level churn**: Metrics aggregated per function/method
- **Integration**: Post-processing phase in index building pipeline
- **Storage**: Metrics stored as metadata on graph nodes (FileScope, Function)

### 1.2 Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Processing Phase | Post-processing | Requires complete code graph before attribution |
| Storage Location | Neo4j node metadata | Aligns with existing patterns; queryable via Cypher |
| Git Interface | Go git library | Already used in codebase (`util/git.go`) |
| Configuration | YAML-based toggle | Matches existing `IndexBuildingConfig` pattern |

---

## 2. Architecture Overview

### 2.1 Component Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Index Building Pipeline                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐        │
│  │ CodeGraph       │   │ Embedding       │   │ Summary         │        │
│  │ Processor       │   │ Processor       │   │ Processor       │        │
│  └────────┬────────┘   └────────┬────────┘   └────────┬────────┘        │
│           │                     │                     │                  │
│           └─────────────────────┼─────────────────────┘                  │
│                                 │                                        │
│                                 ▼                                        │
│                    ┌────────────────────────┐                            │
│                    │   PostProcess Phase    │                            │
│                    └────────────┬───────────┘                            │
│                                 │                                        │
│           ┌─────────────────────┼─────────────────────┐                  │
│           ▼                     ▼                     ▼                  │
│  ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐        │
│  │ LSP Call        │   │ Inheritance     │   │ Git Churn       │  NEW   │
│  │ Resolution      │   │ Processing      │   │ Processor       │◄───────│
│  └─────────────────┘   └─────────────────┘   └────────┬────────┘        │
│                                                       │                  │
└───────────────────────────────────────────────────────┼──────────────────┘
                                                        │
                                                        ▼
                                              ┌─────────────────┐
                                              │   Neo4j Graph   │
                                              │  (Node Metadata)│
                                              └─────────────────┘
```

### 2.2 Data Flow

```
1. Code graph is built (files, functions, classes already exist in Neo4j)
                                ↓
2. GitChurnProcessor.PostProcess() is invoked
                                ↓
3. Extract file list from Neo4j (FileScope nodes)
                                ↓
4. Query git log for commit history within time window
                                ↓
5. For each commit:
   ├─ Parse numstat output (file-level line changes)
   └─ Parse diff output (line ranges for function attribution)
                                ↓
6. Aggregate metrics by file path
                                ↓
7. For function-level: Map changed lines → function ranges
                                ↓
8. Update Neo4j nodes with churn metadata
```

---

## 3. Data Model

### 3.1 Churn Metrics Schema

Metrics will be stored in the `MetaData` field of AST nodes, following existing patterns.

#### File-Level Metrics (FileScope nodes)

```go
// Stored in FileScope.MetaData
type FileChurnMetrics struct {
    // Core metrics
    LinesAdded      int   `json:"churn_lines_added"`
    LinesDeleted    int   `json:"churn_lines_deleted"`
    LinesChanged    int   `json:"churn_lines_changed"`    // adds + deletes
    CommitCount     int   `json:"churn_commit_count"`
    AuthorCount     int   `json:"churn_author_count"`

    // Derived metrics
    ChurnScore      float64 `json:"churn_score"`          // Composite score
    ChurnDensity    float64 `json:"churn_density"`        // Score / LOC

    // Time context
    FirstCommitDate string  `json:"churn_first_commit"`   // ISO8601
    LastCommitDate  string  `json:"churn_last_commit"`    // ISO8601
    TimeWindowDays  int     `json:"churn_window_days"`

    // Additional signals
    AvgChangeSize   float64 `json:"churn_avg_change_size"` // Avg lines per commit
}
```

#### Function-Level Metrics (Function nodes)

```go
// Stored in Function.MetaData
type FunctionChurnMetrics struct {
    LinesAdded      int     `json:"churn_lines_added"`
    LinesDeleted    int     `json:"churn_lines_deleted"`
    LinesChanged    int     `json:"churn_lines_changed"`
    CommitCount     int     `json:"churn_commit_count"`
    AuthorCount     int     `json:"churn_author_count"`
    ChurnScore      float64 `json:"churn_score"`
    ChurnDensity    float64 `json:"churn_density"`

    // Function stability indicators
    SignatureChanged bool   `json:"churn_sig_changed"`    // Was signature modified?
    BodyOnlyChanges  int    `json:"churn_body_changes"`   // Changes inside body only
}
```

### 3.2 Neo4j Storage Format

Metrics are stored as node properties with `md_churn_` prefix:

```cypher
// Example FileScope node with churn metrics
(:FileScope {
    id: 12345,
    name: "user_service.go",
    md_path: "/internal/service/user_service.go",
    md_language: "go",
    // Churn metrics
    md_churn_lines_added: 245,
    md_churn_lines_deleted: 112,
    md_churn_lines_changed: 357,
    md_churn_commit_count: 23,
    md_churn_author_count: 4,
    md_churn_score: 892.4,
    md_churn_density: 1.78,
    md_churn_first_commit: "2025-09-15T10:30:00Z",
    md_churn_last_commit: "2026-01-25T14:22:00Z",
    md_churn_window_days: 180
})
```

### 3.3 Churn Score Formula

Following the requirements document:

```go
// Configurable weights (defaults from requirements)
type ChurnWeights struct {
    LinesChanged float64 // 0.5
    CommitCount  float64 // 0.3
    AuthorCount  float64 // 0.2
}

func CalculateChurnScore(metrics *ChurnMetrics, weights ChurnWeights) float64 {
    return float64(metrics.LinesChanged) * weights.LinesChanged +
           float64(metrics.CommitCount) * weights.CommitCount +
           float64(metrics.AuthorCount) * weights.AuthorCount
}

func CalculateChurnDensity(score float64, linesOfCode int) float64 {
    if linesOfCode == 0 {
        return 0
    }
    return score / float64(linesOfCode)
}
```

---

## 4. Configuration

### 4.1 Config Structure

**File:** `internal/config/config.go`

```go
// Add to existing Config struct
type Config struct {
    // ... existing fields ...
    GitChurn GitChurnConfig `yaml:"git_churn"`
}

type GitChurnConfig struct {
    // Master toggle
    Enabled bool `yaml:"enabled" default:"false"`

    // Time window configuration
    TimeWindowDays  int    `yaml:"time_window_days" default:"180"`
    LookbackCommits int    `yaml:"lookback_commits" default:"0"` // 0 = use time window

    // Granularity toggles
    EnableFileLevel     bool `yaml:"enable_file_level" default:"true"`
    EnableFunctionLevel bool `yaml:"enable_function_level" default:"true"`

    // Score weights
    Weights ChurnWeights `yaml:"weights"`

    // Filtering
    ExcludePatterns []string `yaml:"exclude_patterns"` // e.g., ["*.test.go", "vendor/**"]
    ExcludeAuthors  []string `yaml:"exclude_authors"`  // e.g., ["dependabot[bot]"]
    ExcludeMerges   bool     `yaml:"exclude_merges" default:"true"`

    // Performance
    MaxConcurrency int `yaml:"max_concurrency" default:"4"`
}

type ChurnWeights struct {
    LinesChanged float64 `yaml:"lines_changed" default:"0.5"`
    CommitCount  float64 `yaml:"commit_count" default:"0.3"`
    AuthorCount  float64 `yaml:"author_count" default:"0.2"`
}
```

### 4.2 Example YAML Configuration

**File:** `config/app.yaml`

```yaml
git_churn:
  enabled: true
  time_window_days: 180

  enable_file_level: true
  enable_function_level: true

  weights:
    lines_changed: 0.5
    commit_count: 0.3
    author_count: 0.2

  exclude_patterns:
    - "vendor/**"
    - "**/*_test.go"
    - "**/testdata/**"
    - "**/*.pb.go"  # Generated protobuf

  exclude_authors:
    - "dependabot[bot]"
    - "renovate[bot]"

  exclude_merges: true
  max_concurrency: 4
```

---

## 5. Implementation Components

### 5.1 GitChurnProcessor

**File:** `internal/controller/git_churn_processor.go`

```go
package controller

import (
    "context"
    "time"

    "codeapi/internal/config"
    "codeapi/internal/model/ast"
    "codeapi/internal/service/codegraph"
    "codeapi/internal/util"
    "go.uber.org/zap"
)

// GitChurnProcessor computes git churn metrics and stores them on graph nodes
type GitChurnProcessor struct {
    codeGraph *codegraph.CodeGraph
    config    *config.GitChurnConfig
    logger    *zap.Logger

    // Cached data per repository
    gitLog    *GitLogCache
}

// Ensure interface compliance
var _ FileProcessor = (*GitChurnProcessor)(nil)

func NewGitChurnProcessor(
    codeGraph *codegraph.CodeGraph,
    config *config.GitChurnConfig,
    logger *zap.Logger,
) *GitChurnProcessor {
    return &GitChurnProcessor{
        codeGraph: codeGraph,
        config:    config,
        logger:    logger,
    }
}

func (gcp *GitChurnProcessor) Name() string {
    return "GitChurn"
}

func (gcp *GitChurnProcessor) Init(ctx context.Context, repo *config.Repository) error {
    // Initialize git log cache for this repository
    gcp.gitLog = NewGitLogCache(repo.LocalPath, gcp.config)
    return nil
}

func (gcp *GitChurnProcessor) ProcessFile(
    ctx context.Context,
    repo *config.Repository,
    fileCtx *FileContext,
) error {
    // No-op during file processing phase
    // Churn analysis happens in PostProcess after all files are indexed
    return nil
}

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
```

### 5.2 Git Log Cache

**File:** `internal/controller/git_log_cache.go`

```go
package controller

import (
    "bufio"
    "context"
    "fmt"
    "os/exec"
    "strconv"
    "strings"
    "time"
)

// GitLogCache caches parsed git log data for efficient querying
type GitLogCache struct {
    repoPath string
    config   *config.GitChurnConfig

    // Per-file aggregated metrics
    fileMetrics map[string]*FileChurnData

    // Per-commit data for function-level attribution
    commits []CommitData
}

type FileChurnData struct {
    LinesAdded    int
    LinesDeleted  int
    CommitCount   int
    Authors       map[string]bool
    FirstCommit   time.Time
    LastCommit    time.Time
}

type CommitData struct {
    SHA       string
    Author    string
    Date      time.Time
    IsMerge   bool
    FileStats []FileStatData
}

type FileStatData struct {
    FilePath     string
    LinesAdded   int
    LinesDeleted int
    // For function-level attribution
    HunkRanges   []HunkRange
}

type HunkRange struct {
    OldStart int
    OldCount int
    NewStart int
    NewCount int
}

func NewGitLogCache(repoPath string, config *config.GitChurnConfig) *GitLogCache {
    return &GitLogCache{
        repoPath:    repoPath,
        config:      config,
        fileMetrics: make(map[string]*FileChurnData),
    }
}

func (glc *GitLogCache) Build(ctx context.Context) error {
    // Calculate time window
    since := time.Now().AddDate(0, 0, -glc.config.TimeWindowDays)

    // Build git log command
    args := []string{
        "log",
        "--since=" + since.Format("2006-01-02"),
        "--numstat",
        "--format=%H|%an|%aI|%P",  // SHA|Author|Date|Parents
    }

    if glc.config.ExcludeMerges {
        args = append(args, "--no-merges")
    }

    cmd := exec.CommandContext(ctx, "git", args...)
    cmd.Dir = glc.repoPath

    output, err := cmd.Output()
    if err != nil {
        return fmt.Errorf("git log failed: %w", err)
    }

    return glc.parseGitLog(string(output))
}

func (glc *GitLogCache) parseGitLog(output string) error {
    scanner := bufio.NewScanner(strings.NewReader(output))
    var currentCommit *CommitData

    for scanner.Scan() {
        line := scanner.Text()

        if line == "" {
            continue
        }

        // Check if this is a commit header line
        if strings.Contains(line, "|") && len(strings.Split(line, "|")) >= 3 {
            parts := strings.Split(line, "|")
            commitDate, _ := time.Parse(time.RFC3339, parts[2])
            isMerge := len(strings.Fields(parts[3])) > 1

            currentCommit = &CommitData{
                SHA:     parts[0],
                Author:  parts[1],
                Date:    commitDate,
                IsMerge: isMerge,
            }

            // Skip if author is excluded
            if glc.isExcludedAuthor(currentCommit.Author) {
                currentCommit = nil
                continue
            }

            glc.commits = append(glc.commits, *currentCommit)
            continue
        }

        // Parse numstat line: "added\tdeleted\tfilepath"
        if currentCommit != nil {
            parts := strings.Split(line, "\t")
            if len(parts) == 3 {
                added, _ := strconv.Atoi(parts[0])
                deleted, _ := strconv.Atoi(parts[1])
                filePath := parts[2]

                // Skip excluded patterns
                if glc.isExcludedPath(filePath) {
                    continue
                }

                // Aggregate file metrics
                glc.aggregateFileMetrics(filePath, added, deleted, currentCommit)
            }
        }
    }

    return scanner.Err()
}

func (glc *GitLogCache) aggregateFileMetrics(
    filePath string,
    added, deleted int,
    commit *CommitData,
) {
    metrics, exists := glc.fileMetrics[filePath]
    if !exists {
        metrics = &FileChurnData{
            Authors:     make(map[string]bool),
            FirstCommit: commit.Date,
            LastCommit:  commit.Date,
        }
        glc.fileMetrics[filePath] = metrics
    }

    metrics.LinesAdded += added
    metrics.LinesDeleted += deleted
    metrics.CommitCount++
    metrics.Authors[commit.Author] = true

    if commit.Date.Before(metrics.FirstCommit) {
        metrics.FirstCommit = commit.Date
    }
    if commit.Date.After(metrics.LastCommit) {
        metrics.LastCommit = commit.Date
    }
}

func (glc *GitLogCache) isExcludedAuthor(author string) bool {
    for _, excluded := range glc.config.ExcludeAuthors {
        if author == excluded {
            return true
        }
    }
    return false
}

func (glc *GitLogCache) isExcludedPath(path string) bool {
    for _, pattern := range glc.config.ExcludePatterns {
        if matched, _ := filepath.Match(pattern, path); matched {
            return true
        }
        // Also check with doublestar for ** patterns
        if matched, _ := doublestar.Match(pattern, path); matched {
            return true
        }
    }
    return false
}

// GetFileMetrics returns churn metrics for a specific file
func (glc *GitLogCache) GetFileMetrics(relativePath string) *FileChurnData {
    return glc.fileMetrics[relativePath]
}
```

### 5.3 File-Level Churn Processing

**File:** `internal/controller/git_churn_processor.go` (continued)

```go
func (gcp *GitChurnProcessor) processFileLevelChurn(
    ctx context.Context,
    repo *config.Repository,
) error {
    // Get all FileScope nodes for this repository
    fileScopes, err := gcp.codeGraph.FindFileScopesByRepo(ctx, repo.Name)
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

    // Collect any errors
    var errs []error
    for err := range errChan {
        errs = append(errs, err)
    }

    if len(errs) > 0 {
        gcp.logger.Warn("Some files failed churn processing",
            zap.Int("errorCount", len(errs)))
    }

    return nil
}

func (gcp *GitChurnProcessor) updateFileChurnMetrics(
    ctx context.Context,
    fileScope *ast.Node,
) error {
    // Get relative path from file scope metadata
    relativePath, ok := fileScope.MetaData["path"].(string)
    if !ok {
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

    score := float64(linesChanged) * gcp.config.Weights.LinesChanged +
             float64(churnData.CommitCount) * gcp.config.Weights.CommitCount +
             float64(authorCount) * gcp.config.Weights.AuthorCount

    density := 0.0
    if loc > 0 {
        density = score / float64(loc)
    }

    // Prepare metadata updates
    if fileScope.MetaData == nil {
        fileScope.MetaData = make(map[string]any)
    }

    fileScope.MetaData["churn_lines_added"] = churnData.LinesAdded
    fileScope.MetaData["churn_lines_deleted"] = churnData.LinesDeleted
    fileScope.MetaData["churn_lines_changed"] = linesChanged
    fileScope.MetaData["churn_commit_count"] = churnData.CommitCount
    fileScope.MetaData["churn_author_count"] = authorCount
    fileScope.MetaData["churn_score"] = score
    fileScope.MetaData["churn_density"] = density
    fileScope.MetaData["churn_window_days"] = gcp.config.TimeWindowDays

    if !churnData.FirstCommit.IsZero() {
        fileScope.MetaData["churn_first_commit"] = churnData.FirstCommit.Format(time.RFC3339)
        fileScope.MetaData["churn_last_commit"] = churnData.LastCommit.Format(time.RFC3339)
    }

    // Average change size
    if churnData.CommitCount > 0 {
        fileScope.MetaData["churn_avg_change_size"] =
            float64(linesChanged) / float64(churnData.CommitCount)
    }

    // Update node in Neo4j
    return gcp.codeGraph.UpdateNodeMetadata(ctx, fileScope)
}

func (gcp *GitChurnProcessor) getFileLineCount(fileScope *ast.Node) int {
    // Try to get from existing metadata
    if loc, ok := fileScope.MetaData["line_count"].(int); ok {
        return loc
    }

    // Calculate from range if available
    if fileScope.Range.End.Line > 0 {
        return int(fileScope.Range.End.Line)
    }

    return 0
}
```

### 5.4 Function-Level Churn Processing

**File:** `internal/controller/git_churn_processor.go` (continued)

```go
func (gcp *GitChurnProcessor) processFunctionLevelChurn(
    ctx context.Context,
    repo *config.Repository,
) error {
    // Get all files with high churn (optimization: skip stable files)
    highChurnFiles, err := gcp.getHighChurnFiles(ctx, repo)
    if err != nil {
        return fmt.Errorf("failed to identify high churn files: %w", err)
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

func (gcp *GitChurnProcessor) getHighChurnFiles(
    ctx context.Context,
    repo *config.Repository,
) ([]string, error) {
    // Query Neo4j for files with churn score above threshold
    // Using hybrid approach: only deep-dive into top 10% by churn
    query := `
        MATCH (f:FileScope)
        WHERE f.repo = $repo AND f.md_churn_score IS NOT NULL
        WITH f, f.md_churn_score AS score
        ORDER BY score DESC
        WITH collect(f.md_path) AS allPaths, count(f) AS total
        RETURN allPaths[0..toInteger(total * 0.1)] AS highChurnPaths
    `

    result, err := gcp.codeGraph.ExecuteQuery(ctx, query, map[string]any{
        "repo": repo.Name,
    })
    if err != nil {
        return nil, err
    }

    return extractPathsFromResult(result), nil
}

func (gcp *GitChurnProcessor) processFunctionsInFile(
    ctx context.Context,
    repo *config.Repository,
    filePath string,
) error {
    // Get all Function nodes in this file
    functions, err := gcp.codeGraph.FindFunctionsByFilePath(ctx, repo.Name, filePath)
    if err != nil {
        return err
    }

    // Get detailed diff data for this file
    diffData, err := gcp.getDiffDataForFile(ctx, filePath)
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

func (gcp *GitChurnProcessor) attributeChangesToFunction(
    fn *ast.Node,
    diffData *FileDiffData,
) *FunctionChurnMetrics {
    metrics := &FunctionChurnMetrics{
        Authors: make(map[string]bool),
    }

    fnStartLine := int(fn.Range.Start.Line)
    fnEndLine := int(fn.Range.End.Line)

    // For each commit that touched this file
    for _, commit := range diffData.Commits {
        touchedFunction := false

        for _, hunk := range commit.Hunks {
            // Check if hunk overlaps with function range
            // Note: We use the NEW file line numbers since that's the current state
            hunkStart := hunk.NewStart
            hunkEnd := hunk.NewStart + hunk.NewCount

            if overlaps(fnStartLine, fnEndLine, hunkStart, hunkEnd) {
                touchedFunction = true

                // Count lines within function range
                overlapStart := max(fnStartLine, hunkStart)
                overlapEnd := min(fnEndLine, hunkEnd)
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

func (gcp *GitChurnProcessor) updateFunctionChurnMetrics(
    ctx context.Context,
    fn *ast.Node,
    metrics *FunctionChurnMetrics,
) error {
    // Calculate composite score
    linesChanged := metrics.LinesAdded + metrics.LinesDeleted
    authorCount := len(metrics.Authors)

    score := float64(linesChanged) * gcp.config.Weights.LinesChanged +
             float64(metrics.CommitCount) * gcp.config.Weights.CommitCount +
             float64(authorCount) * gcp.config.Weights.AuthorCount

    // Calculate function LOC for density
    fnLOC := int(fn.Range.End.Line - fn.Range.Start.Line + 1)
    density := 0.0
    if fnLOC > 0 {
        density = score / float64(fnLOC)
    }

    // Update metadata
    if fn.MetaData == nil {
        fn.MetaData = make(map[string]any)
    }

    fn.MetaData["churn_lines_added"] = metrics.LinesAdded
    fn.MetaData["churn_lines_deleted"] = metrics.LinesDeleted
    fn.MetaData["churn_lines_changed"] = linesChanged
    fn.MetaData["churn_commit_count"] = metrics.CommitCount
    fn.MetaData["churn_author_count"] = authorCount
    fn.MetaData["churn_score"] = score
    fn.MetaData["churn_density"] = density

    return gcp.codeGraph.UpdateNodeMetadata(ctx, fn)
}

// Helper functions
func overlaps(aStart, aEnd, bStart, bEnd int) bool {
    return aStart <= bEnd && bStart <= aEnd
}

func max(a, b int) int {
    if a > b { return a }
    return b
}

func min(a, b int) int {
    if a < b { return a }
    return b
}
```

### 5.5 Processor Registration

**File:** `internal/init/service_init.go` (modification)

```go
func (sc *ServiceContainer) InitProcessors(cfg *config.Config) error {
    var processors []controller.FileProcessor

    // Existing processors
    if sc.CodeGraph != nil {
        processors = append(processors, controller.NewCodeGraphProcessor(
            sc.CodeGraph,
            sc.RepoService,
            cfg,
            sc.Logger,
        ))
    }

    if sc.ChunkService != nil {
        processors = append(processors, controller.NewEmbeddingProcessor(
            sc.ChunkService,
            sc.Logger,
        ))
    }

    if sc.LLMService != nil {
        processors = append(processors, controller.NewSummaryProcessor(
            sc.LLMService,
            sc.PromptManager,
            sc.MySQLConn,
            cfg,
            sc.Logger,
        ))
    }

    // NEW: Git Churn Processor
    if cfg.GitChurn.Enabled && sc.CodeGraph != nil {
        processors = append(processors, controller.NewGitChurnProcessor(
            sc.CodeGraph,
            &cfg.GitChurn,
            sc.Logger,
        ))
    }

    sc.Processors = processors
    return nil
}
```

---

## 6. Required CodeGraph Additions

### 6.1 New Query Methods

**File:** `internal/service/codegraph/code_graph.go` (additions)

```go
// FindFileScopesByRepo returns all FileScope nodes for a repository
func (cg *CodeGraph) FindFileScopesByRepo(ctx context.Context, repoName string) ([]*ast.Node, error) {
    query := `
        MATCH (f:FileScope)
        WHERE f.repo = $repo
        RETURN f
    `
    return cg.executeNodeQuery(ctx, query, map[string]any{"repo": repoName})
}

// FindFunctionsByFilePath returns all Function nodes in a specific file
func (cg *CodeGraph) FindFunctionsByFilePath(
    ctx context.Context,
    repoName string,
    filePath string,
) ([]*ast.Node, error) {
    query := `
        MATCH (file:FileScope)-[:CONTAINS*]->(fn:Function)
        WHERE file.repo = $repo AND file.md_path = $path
        RETURN fn
    `
    return cg.executeNodeQuery(ctx, query, map[string]any{
        "repo": repoName,
        "path": filePath,
    })
}

// UpdateNodeMetadata updates the metadata properties of a node
func (cg *CodeGraph) UpdateNodeMetadata(ctx context.Context, node *ast.Node) error {
    // Build SET clause for metadata fields
    setClauses := []string{}
    params := map[string]any{"id": node.ID}

    for key, value := range node.MetaData {
        paramName := "md_" + key
        setClauses = append(setClauses, fmt.Sprintf("n.md_%s = $%s", key, paramName))
        params[paramName] = value
    }

    if len(setClauses) == 0 {
        return nil
    }

    query := fmt.Sprintf(`
        MATCH (n)
        WHERE n.id = $id
        SET %s
    `, strings.Join(setClauses, ", "))

    return cg.executeWrite(ctx, query, params)
}
```

---

## 7. API Extensions

### 7.1 Churn Query Endpoints

**File:** `internal/handler/churn_handler.go` (new)

```go
package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

type ChurnHandler struct {
    codeGraph *codegraph.CodeGraph
}

// GET /codeapi/v1/repos/:repo/churn/files
// Returns file churn rankings
func (h *ChurnHandler) GetFileChurn(c *gin.Context) {
    repo := c.Param("repo")
    limit := c.DefaultQuery("limit", "50")

    query := `
        MATCH (f:FileScope)
        WHERE f.repo = $repo AND f.md_churn_score IS NOT NULL
        RETURN f.md_path AS path,
               f.md_churn_score AS score,
               f.md_churn_density AS density,
               f.md_churn_commit_count AS commits,
               f.md_churn_author_count AS authors,
               f.md_churn_lines_changed AS linesChanged
        ORDER BY f.md_churn_score DESC
        LIMIT $limit
    `

    // Execute and return results...
}

// GET /codeapi/v1/repos/:repo/churn/functions
// Returns function churn rankings
func (h *ChurnHandler) GetFunctionChurn(c *gin.Context) {
    repo := c.Param("repo")
    limit := c.DefaultQuery("limit", "50")
    filePath := c.Query("file") // Optional filter

    query := `
        MATCH (file:FileScope)-[:CONTAINS*]->(fn:Function)
        WHERE file.repo = $repo
          AND fn.md_churn_score IS NOT NULL
          AND ($filePath IS NULL OR file.md_path = $filePath)
        RETURN fn.name AS name,
               file.md_path AS file,
               fn.md_churn_score AS score,
               fn.md_churn_density AS density,
               fn.md_churn_commit_count AS commits
        ORDER BY fn.md_churn_score DESC
        LIMIT $limit
    `

    // Execute and return results...
}

// GET /codeapi/v1/repos/:repo/churn/hotspots
// Returns combined hotspot analysis
func (h *ChurnHandler) GetHotspots(c *gin.Context) {
    repo := c.Param("repo")
    threshold := c.DefaultQuery("threshold", "0.9") // Top 10%

    // Return files and functions above threshold...
}
```

---

## 8. Implementation Phases

### Phase 1: Foundation (Core Infrastructure)

**Estimated scope:** Basic file-level churn

| Task | File | Description |
|------|------|-------------|
| 1.1 | `config/config.go` | Add `GitChurnConfig` struct |
| 1.2 | `controller/git_churn_processor.go` | Create processor skeleton |
| 1.3 | `controller/git_log_cache.go` | Implement git log parsing |
| 1.4 | `init/service_init.go` | Register processor |
| 1.5 | `codegraph/code_graph.go` | Add `UpdateNodeMetadata` |

**Deliverable:** File-level churn scores visible in Neo4j

### Phase 2: Function-Level Attribution

**Estimated scope:** Function granularity

| Task | File | Description |
|------|------|-------------|
| 2.1 | `controller/git_log_cache.go` | Add hunk-level diff parsing |
| 2.2 | `controller/git_churn_processor.go` | Implement function attribution |
| 2.3 | `codegraph/code_graph.go` | Add `FindFunctionsByFilePath` |

**Deliverable:** Function-level churn scores

### Phase 3: API and Optimization

**Estimated scope:** Query endpoints, performance

| Task | File | Description |
|------|------|-------------|
| 3.1 | `handler/churn_handler.go` | Create REST endpoints |
| 3.2 | `router.go` | Register routes |
| 3.3 | `controller/git_churn_processor.go` | Optimize with concurrency |
| 3.4 | Config | Add exclude patterns, author filters |

**Deliverable:** Production-ready churn analysis

---

## 9. Testing Strategy

### 9.1 Unit Tests

| Component | Test Cases |
|-----------|------------|
| `GitLogCache` | Parse numstat output; handle renames; filter patterns |
| `ChurnScore` | Weight calculation; density normalization; edge cases |
| `FunctionAttribution` | Hunk-to-function mapping; overlapping ranges |

### 9.2 Integration Tests

| Scenario | Validation |
|----------|------------|
| Full pipeline | Index repo → verify churn in Neo4j |
| Empty history | Handle repos with no commits in window |
| Large repo | Performance within targets (10K files < 10min) |

### 9.3 Test Fixtures

Create test repository with known churn patterns:
- `stable_file.go` - No changes in window
- `hotspot_file.go` - Many commits, many authors
- `single_function_churn.go` - One function with high churn

---

## 10. Example Queries

### Find Top Churning Files

```cypher
MATCH (f:FileScope)
WHERE f.repo = "myrepo" AND f.md_churn_score IS NOT NULL
RETURN f.md_path AS file,
       f.md_churn_score AS score,
       f.md_churn_density AS density,
       f.md_churn_commit_count AS commits,
       f.md_churn_author_count AS authors
ORDER BY score DESC
LIMIT 20
```

### Find Hotspot Functions

```cypher
MATCH (file:FileScope)-[:CONTAINS*]->(fn:Function)
WHERE file.repo = "myrepo"
  AND fn.md_churn_density > 2.0
RETURN fn.name AS function,
       file.md_path AS file,
       fn.md_churn_density AS density,
       fn.md_churn_score AS score
ORDER BY density DESC
```

### Compare File Churn vs Function Churn

```cypher
MATCH (file:FileScope)-[:CONTAINS*]->(fn:Function)
WHERE file.repo = "myrepo"
  AND file.md_churn_score > 500
WITH file,
     file.md_churn_score AS fileChurn,
     collect({name: fn.name, score: fn.md_churn_score}) AS functions
RETURN file.md_path AS file,
       fileChurn,
       [f IN functions WHERE f.score > 100 | f.name] AS hotFunctions
```

---

## 11. Future Enhancements

### 11.1 Incremental Updates

- Track last processed commit per repository
- Only process new commits since last run
- Delta updates to existing metrics

### 11.2 Trend Analysis

- Store historical snapshots (weekly/monthly)
- Track churn trajectory over time
- Alert on deteriorating metrics

### 11.3 Correlation Features

- Link churn to code complexity metrics
- Correlate with test coverage data
- Integration with issue tracking systems

---

## Appendix A: File Reference

| File | Purpose |
|------|---------|
| `internal/config/config.go` | Configuration structs |
| `internal/controller/git_churn_processor.go` | Main processor |
| `internal/controller/git_log_cache.go` | Git history caching |
| `internal/service/codegraph/code_graph.go` | Neo4j extensions |
| `internal/handler/churn_handler.go` | API endpoints |
| `internal/init/service_init.go` | Processor registration |

---

## Appendix B: Revision History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | January 2026 | Initial design |
