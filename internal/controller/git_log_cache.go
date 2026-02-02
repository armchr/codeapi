package controller

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/armchr/codeapi/internal/config"
	"go.uber.org/zap"
)

// GitLogCache caches parsed git log data for efficient querying
type GitLogCache struct {
	repoPath string
	config   *config.GitChurnConfig
	logger   *zap.Logger

	// Per-file aggregated metrics
	fileMetrics map[string]*FileChurnData

	// Per-commit data for function-level attribution
	commits []CommitData
}

// FileChurnData holds aggregated churn metrics for a file
type FileChurnData struct {
	LinesAdded   int
	LinesDeleted int
	CommitCount  int
	Authors      map[string]bool
	FirstCommit  time.Time
	LastCommit   time.Time
}

// CommitData holds data for a single commit
type CommitData struct {
	SHA       string
	Author    string
	Date      time.Time
	IsMerge   bool
	FileStats []FileStatData
}

// FileStatData holds file-level statistics from a commit
type FileStatData struct {
	FilePath     string
	LinesAdded   int
	LinesDeleted int
	// For function-level attribution
	HunkRanges []HunkRange
}

// HunkRange represents a changed range in a diff
type HunkRange struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
}

// NewGitLogCache creates a new GitLogCache for the given repository
func NewGitLogCache(repoPath string, config *config.GitChurnConfig, logger *zap.Logger) *GitLogCache {
	return &GitLogCache{
		repoPath:    repoPath,
		config:      config,
		logger:      logger,
		fileMetrics: make(map[string]*FileChurnData),
	}
}

// Build parses the git log and populates the cache
func (glc *GitLogCache) Build(ctx context.Context) error {
	// Build git log command
	args := []string{
		"log",
		"--numstat",
		"--format=%H|%an|%aI|%P", // SHA|Author|Date|Parents
	}

	// Add time window filter if specified (0 means all history)
	if glc.config.TimeWindowDays > 0 {
		since := time.Now().AddDate(0, 0, -glc.config.TimeWindowDays)
		args = append(args, "--since="+since.Format("2006-01-02"))
		glc.logger.Debug("Building git log cache with time window",
			zap.String("repoPath", glc.repoPath),
			zap.Time("since", since),
			zap.Int("timeWindowDays", glc.config.TimeWindowDays))
	} else {
		glc.logger.Debug("Building git log cache for all history",
			zap.String("repoPath", glc.repoPath))
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

	if err := glc.parseGitLog(string(output)); err != nil {
		return fmt.Errorf("failed to parse git log: %w", err)
	}

	glc.logger.Debug("Git log cache built",
		zap.Int("filesTracked", len(glc.fileMetrics)),
		zap.Int("commitsProcessed", len(glc.commits)))

	return nil
}

// parseGitLog parses the raw git log output
func (glc *GitLogCache) parseGitLog(output string) error {
	scanner := bufio.NewScanner(strings.NewReader(output))
	var currentCommit *CommitData

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		// Check if this is a commit header line (contains | separators)
		if strings.Count(line, "|") >= 3 {
			parts := strings.SplitN(line, "|", 4)
			if len(parts) >= 4 {
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
		}

		// Parse numstat line: "added\tdeleted\tfilepath"
		if currentCommit != nil && strings.Contains(line, "\t") {
			parts := strings.Split(line, "\t")
			if len(parts) >= 3 {
				// Handle binary files which show "-" instead of numbers
				added, err1 := strconv.Atoi(parts[0])
				deleted, err2 := strconv.Atoi(parts[1])
				if err1 != nil || err2 != nil {
					// Binary file, skip
					continue
				}

				filePath := parts[2]

				// Handle renames: "old_path => new_path" or "{old => new}/path"
				if strings.Contains(filePath, " => ") {
					filePath = glc.parseRenamedPath(filePath)
				}

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

// parseRenamedPath extracts the new path from a rename notation
func (glc *GitLogCache) parseRenamedPath(path string) string {
	// Handle simple rename: "old_path => new_path"
	if strings.Contains(path, " => ") && !strings.Contains(path, "{") {
		parts := strings.Split(path, " => ")
		if len(parts) == 2 {
			return strings.TrimSpace(parts[1])
		}
	}

	// Handle complex rename: "{old => new}/path" or "path/{old => new}/file"
	// This format indicates partial path changes
	start := strings.Index(path, "{")
	end := strings.Index(path, "}")
	if start >= 0 && end > start {
		prefix := path[:start]
		suffix := path[end+1:]
		inner := path[start+1 : end]

		parts := strings.Split(inner, " => ")
		if len(parts) == 2 {
			newPart := strings.TrimSpace(parts[1])
			return prefix + newPart + suffix
		}
	}

	return path
}

// aggregateFileMetrics updates file metrics for a change
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

// isExcludedAuthor checks if an author should be excluded
func (glc *GitLogCache) isExcludedAuthor(author string) bool {
	for _, excluded := range glc.config.ExcludeAuthors {
		if author == excluded {
			return true
		}
	}
	return false
}

// isExcludedPath checks if a path matches any exclude pattern
func (glc *GitLogCache) isExcludedPath(path string) bool {
	for _, pattern := range glc.config.ExcludePatterns {
		if matchGlobPattern(pattern, path) {
			return true
		}
	}
	return false
}

// matchGlobPattern matches a path against a glob pattern supporting ** for recursive matching
func matchGlobPattern(pattern, path string) bool {
	// If pattern contains **, use recursive matching
	if strings.Contains(pattern, "**") {
		return matchDoublestar(pattern, path)
	}
	// Otherwise use standard filepath.Match
	matched, _ := filepath.Match(pattern, path)
	return matched
}

// matchDoublestar implements recursive glob matching for ** patterns
func matchDoublestar(pattern, path string) bool {
	// Split pattern by **
	parts := strings.Split(pattern, "**")
	if len(parts) == 0 {
		return false
	}

	// Handle leading **
	if strings.HasPrefix(pattern, "**") {
		// ** at start matches any prefix
		suffix := strings.TrimPrefix(pattern, "**")
		suffix = strings.TrimPrefix(suffix, "/")
		if suffix == "" {
			return true
		}
		// Check if any suffix of path matches the remaining pattern
		pathParts := strings.Split(path, "/")
		for i := 0; i <= len(pathParts); i++ {
			subPath := strings.Join(pathParts[i:], "/")
			if matchGlobPattern(suffix, subPath) {
				return true
			}
		}
		return false
	}

	// Handle trailing **
	if strings.HasSuffix(pattern, "**") {
		prefix := strings.TrimSuffix(pattern, "**")
		prefix = strings.TrimSuffix(prefix, "/")
		// Check if path starts with prefix
		if matched, _ := filepath.Match(prefix+"/*", path); matched {
			return true
		}
		return strings.HasPrefix(path, prefix+"/") || path == prefix
	}

	// Handle ** in the middle
	prefix := parts[0]
	suffix := parts[1]
	prefix = strings.TrimSuffix(prefix, "/")
	suffix = strings.TrimPrefix(suffix, "/")

	// Path must match prefix, then anything, then suffix
	if prefix != "" {
		if !strings.HasPrefix(path, prefix) {
			return false
		}
		path = strings.TrimPrefix(path, prefix)
		path = strings.TrimPrefix(path, "/")
	}

	if suffix != "" {
		// Check if suffix matches any ending of the remaining path
		pathParts := strings.Split(path, "/")
		for i := 0; i <= len(pathParts); i++ {
			subPath := strings.Join(pathParts[i:], "/")
			if matchGlobPattern(suffix, subPath) {
				return true
			}
		}
		return false
	}

	return true
}

// GetFileMetrics returns churn metrics for a specific file
func (glc *GitLogCache) GetFileMetrics(relativePath string) *FileChurnData {
	return glc.fileMetrics[relativePath]
}

// GetAllFileMetrics returns all file metrics
func (glc *GitLogCache) GetAllFileMetrics() map[string]*FileChurnData {
	return glc.fileMetrics
}

// GetCommits returns all parsed commits
func (glc *GitLogCache) GetCommits() []CommitData {
	return glc.commits
}

// BuildDiffData builds detailed diff data for function-level attribution
// This runs git log with -p for detailed diffs on specific files
func (glc *GitLogCache) BuildDiffData(ctx context.Context, filePath string) (*FileDiffData, error) {
	args := []string{
		"log",
		"-p",                    // Show patch (diff)
		"--format=%H|%an|%aI",   // SHA|Author|Date
		"--follow",              // Follow renames
	}

	// Add time window filter if specified (0 means all history)
	if glc.config.TimeWindowDays > 0 {
		since := time.Now().AddDate(0, 0, -glc.config.TimeWindowDays)
		args = append(args, "--since="+since.Format("2006-01-02"))
	}

	if glc.config.ExcludeMerges {
		args = append(args, "--no-merges")
	}

	// Add file path after "--" separator (must be last)
	args = append(args, "--", filePath)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = glc.repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log -p failed for %s: %w", filePath, err)
	}

	return glc.parseDiffOutput(string(output))
}

// FileDiffData holds detailed diff data for function attribution
type FileDiffData struct {
	Commits []CommitDiffData
}

// CommitDiffData holds diff data for a single commit
type CommitDiffData struct {
	SHA    string
	Author string
	Date   time.Time
	Hunks  []HunkData
}

// HunkData holds data for a single diff hunk
type HunkData struct {
	OldStart     int
	OldCount     int
	NewStart     int
	NewCount     int
	LinesAdded   int
	LinesDeleted int
}

// parseDiffOutput parses git log -p output for a single file
func (glc *GitLogCache) parseDiffOutput(output string) (*FileDiffData, error) {
	result := &FileDiffData{}

	scanner := bufio.NewScanner(strings.NewReader(output))
	var currentCommit *CommitDiffData
	var currentHunk *HunkData

	for scanner.Scan() {
		line := scanner.Text()

		// Check for commit header
		if strings.Count(line, "|") >= 2 && len(line) > 40 {
			parts := strings.SplitN(line, "|", 3)
			if len(parts) >= 3 && len(parts[0]) == 40 {
				commitDate, _ := time.Parse(time.RFC3339, parts[2])

				// Skip if author is excluded
				if glc.isExcludedAuthor(parts[1]) {
					currentCommit = nil
					continue
				}

				currentCommit = &CommitDiffData{
					SHA:    parts[0],
					Author: parts[1],
					Date:   commitDate,
				}
				result.Commits = append(result.Commits, *currentCommit)
				currentHunk = nil
				continue
			}
		}

		// Check for hunk header: @@ -start,count +start,count @@
		if currentCommit != nil && strings.HasPrefix(line, "@@") {
			hunk := parseHunkHeader(line)
			if hunk != nil {
				currentHunk = hunk
				// Update the last commit in the result
				if len(result.Commits) > 0 {
					result.Commits[len(result.Commits)-1].Hunks = append(
						result.Commits[len(result.Commits)-1].Hunks,
						*currentHunk,
					)
				}
			}
			continue
		}

		// Count added/deleted lines in current hunk
		if currentHunk != nil && len(result.Commits) > 0 {
			lastCommitIdx := len(result.Commits) - 1
			lastHunkIdx := len(result.Commits[lastCommitIdx].Hunks) - 1
			if lastHunkIdx >= 0 {
				if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
					result.Commits[lastCommitIdx].Hunks[lastHunkIdx].LinesAdded++
				} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
					result.Commits[lastCommitIdx].Hunks[lastHunkIdx].LinesDeleted++
				}
			}
		}
	}

	return result, scanner.Err()
}

// parseHunkHeader parses a unified diff hunk header
// Format: @@ -old_start,old_count +new_start,new_count @@
func parseHunkHeader(line string) *HunkData {
	// Find the positions of @@ markers
	start := strings.Index(line, "@@")
	end := strings.Index(line[start+2:], "@@")
	if start < 0 || end < 0 {
		return nil
	}

	header := strings.TrimSpace(line[start+2 : start+2+end])
	parts := strings.Split(header, " ")
	if len(parts) < 2 {
		return nil
	}

	hunk := &HunkData{}

	// Parse old range: -start,count
	if strings.HasPrefix(parts[0], "-") {
		oldParts := strings.Split(parts[0][1:], ",")
		hunk.OldStart, _ = strconv.Atoi(oldParts[0])
		if len(oldParts) > 1 {
			hunk.OldCount, _ = strconv.Atoi(oldParts[1])
		} else {
			hunk.OldCount = 1
		}
	}

	// Parse new range: +start,count
	if strings.HasPrefix(parts[1], "+") {
		newParts := strings.Split(parts[1][1:], ",")
		hunk.NewStart, _ = strconv.Atoi(newParts[0])
		if len(newParts) > 1 {
			hunk.NewCount, _ = strconv.Atoi(newParts[1])
		} else {
			hunk.NewCount = 1
		}
	}

	return hunk
}
