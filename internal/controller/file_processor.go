package controller

import (
	"github.com/armchr/codeapi/internal/config"
	"context"
)

// FileContext contains metadata about a file being processed
type FileContext struct {
	// FileID is the unique identifier for this file version from MySQL
	FileID int32

	// FilePath is the absolute path to the file
	FilePath string

	// RelativePath is the path relative to the repository root
	RelativePath string

	// Content is the file content (already read to avoid multiple I/O)
	Content []byte

	// FileSHA is the SHA256 hash of the file content
	FileSHA string

	// CommitID is the git commit SHA if the file is committed (nil if ephemeral)
	CommitID *string

	// Ephemeral indicates if this is an uncommitted/working directory version
	Ephemeral bool
}

// FileProcessor defines the interface for processing individual files
// and performing repository-level post-processing operations
type FileProcessor interface {
	// ProcessFile processes a single file in the repository
	// ctx: context for cancellation
	// repo: repository configuration
	// fileCtx: file context containing FileID, path, content, and metadata
	// Returns an error if processing fails
	ProcessFile(ctx context.Context, repo *config.Repository, fileCtx *FileContext) error

	// PostProcess performs repository-level operations after all files are processed
	// This is called once after all files have been processed
	PostProcess(ctx context.Context, repo *config.Repository) error

	// Name returns the name of this processor (for logging purposes)
	Name() string
}
