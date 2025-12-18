package db

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
)

// FileVersion represents a versioned file in the repository
type FileVersion struct {
	FileID       int32     `db:"file_id"`
	FileSHA      string    `db:"file_sha"`
	RelativePath string    `db:"relative_path"`
	Ephemeral    bool      `db:"ephemeral"`
	CommitID     *string   `db:"commit_id"`
	Status       string    `db:"status"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// FileVersionRepository manages file version operations
type FileVersionRepository struct {
	db       *sql.DB
	repoName string
	logger   *zap.Logger
}

var (
	// Regex to match characters that are not alphanumeric or underscore
	invalidTableNameChars = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

// sanitizeTableName converts a repository name to a valid SQL table name
// Replaces all special characters (hyphens, spaces, etc.) with underscores
func sanitizeTableName(repoName string) string {
	// Replace all invalid characters with underscore
	sanitized := invalidTableNameChars.ReplaceAllString(repoName, "_")

	// Remove leading/trailing underscores
	sanitized = regexp.MustCompile(`^_+|_+$`).ReplaceAllString(sanitized, "")

	// Replace multiple consecutive underscores with single underscore
	sanitized = regexp.MustCompile(`_+`).ReplaceAllString(sanitized, "_")

	return sanitized
}

// NewFileVersionRepository creates a new repository for managing file versions
func NewFileVersionRepository(db *sql.DB, repoName string, logger *zap.Logger) (*FileVersionRepository, error) {
	repo := &FileVersionRepository{
		db:       db,
		repoName: repoName,
		logger:   logger,
	}

	// Ensure the table exists
	if err := repo.EnsureTable(); err != nil {
		return nil, fmt.Errorf("failed to ensure table: %w", err)
	}

	return repo, nil
}

// tableName returns the sanitized table name for this repository with backticks for SQL safety
func (r *FileVersionRepository) tableName() string {
	sanitized := sanitizeTableName(r.repoName)
	return fmt.Sprintf("`%s_file_versions`", sanitized)
}

// EnsureTable creates the file_versions table if it doesn't exist
// and ensures all required columns are present (handles schema migrations)
func (r *FileVersionRepository) EnsureTable() error {
	tableName := r.tableName()
	r.logger.Info("Ensuring file_versions table exists", zap.String("table", tableName))

	// Create table if it doesn't exist
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			file_id INT AUTO_INCREMENT PRIMARY KEY,
			file_sha VARCHAR(64) NOT NULL,
			relative_path VARCHAR(512) NOT NULL,
			ephemeral BOOLEAN NOT NULL DEFAULT FALSE,
			commit_id VARCHAR(40),
			status VARCHAR(255) NOT NULL DEFAULT 'processing',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY unique_sha_path_commit (file_sha, relative_path, commit_id),
			INDEX idx_file_sha (file_sha),
			INDEX idx_relative_path (relative_path),
			INDEX idx_commit_id (commit_id),
			INDEX idx_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`, tableName)

	if _, err := r.db.Exec(query); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Check if status column exists, add if missing (for existing tables)
	// Extract the bare table name without backticks for information_schema query
	bareTableName := strings.Trim(tableName, "`")
	checkColumnQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		AND TABLE_NAME = '%s'
		AND COLUMN_NAME = 'status'
	`, bareTableName)

	var columnCount int
	err := r.db.QueryRow(checkColumnQuery).Scan(&columnCount)
	if err != nil {
		return fmt.Errorf("failed to check for status column: %w", err)
	}

	if columnCount == 0 {
		r.logger.Info("Adding missing status column", zap.String("table", tableName))
		alterQuery := fmt.Sprintf(`
			ALTER TABLE %s
			ADD COLUMN status VARCHAR(255) NOT NULL DEFAULT 'processing',
			ADD INDEX idx_status (status)
		`, tableName)

		if _, err := r.db.Exec(alterQuery); err != nil {
			return fmt.Errorf("failed to add status column: %w", err)
		}
		r.logger.Info("Status column added successfully", zap.String("table", tableName))
	}

	r.logger.Info("Table ready", zap.String("table", tableName))
	return nil
}

// GetOrCreateFileID retrieves existing FileID or creates a new one
// This is the core method for FileID management
func (r *FileVersionRepository) GetOrCreateFileID(fileSHA, relativePath string, ephemeral bool, commitID *string) (int32, error) {
	tableName := r.tableName()

	// Try to find existing file version
	existing, err := r.findFileVersion(fileSHA, relativePath, commitID)
	if err == nil {
		// Found existing version
		r.logger.Debug("Found existing FileID",
			zap.Int32("file_id", existing.FileID),
			zap.String("sha", fileSHA),
			zap.String("path", relativePath))
		return existing.FileID, nil
	}

	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("error checking for existing file version: %w", err)
	}

	// No existing version found, create new one
	r.logger.Debug("Creating new FileID",
		zap.String("sha", fileSHA),
		zap.String("path", relativePath),
		zap.Bool("ephemeral", ephemeral))

	query := fmt.Sprintf(`
		INSERT INTO %s (file_sha, relative_path, ephemeral, commit_id)
		VALUES (?, ?, ?, ?)
	`, tableName)

	result, err := r.db.Exec(query, fileSHA, relativePath, ephemeral, commitID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert file version: %w", err)
	}

	fileID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	r.logger.Info("Created new FileID",
		zap.Int32("file_id", int32(fileID)),
		zap.String("sha", fileSHA),
		zap.String("path", relativePath),
		zap.Bool("ephemeral", ephemeral))

	return int32(fileID), nil
}

// findFileVersion finds a file version by SHA, path, and commit
func (r *FileVersionRepository) findFileVersion(fileSHA, relativePath string, commitID *string) (*FileVersion, error) {
	tableName := r.tableName()

	query := fmt.Sprintf(`
		SELECT file_id, file_sha, relative_path, ephemeral, commit_id, status, created_at, updated_at
		FROM %s
		WHERE file_sha = ? AND relative_path = ? AND commit_id <=> ?
		LIMIT 1
	`, tableName)

	var fv FileVersion
	err := r.db.QueryRow(query, fileSHA, relativePath, commitID).Scan(
		&fv.FileID,
		&fv.FileSHA,
		&fv.RelativePath,
		&fv.Ephemeral,
		&fv.CommitID,
		&fv.Status,
		&fv.CreatedAt,
		&fv.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &fv, nil
}

// GetFileByID retrieves a file version by its ID
func (r *FileVersionRepository) GetFileByID(fileID int32) (*FileVersion, error) {
	tableName := r.tableName()

	query := fmt.Sprintf(`
		SELECT file_id, file_sha, relative_path, ephemeral, commit_id, status, created_at, updated_at
		FROM %s
		WHERE file_id = ?
	`, tableName)

	var fv FileVersion
	err := r.db.QueryRow(query, fileID).Scan(
		&fv.FileID,
		&fv.FileSHA,
		&fv.RelativePath,
		&fv.Ephemeral,
		&fv.CommitID,
		&fv.Status,
		&fv.CreatedAt,
		&fv.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &fv, nil
}

// GetFilesBySHA retrieves all file versions with a specific SHA
func (r *FileVersionRepository) GetFilesBySHA(fileSHA string) ([]*FileVersion, error) {
	tableName := r.tableName()

	query := fmt.Sprintf(`
		SELECT file_id, file_sha, relative_path, ephemeral, commit_id, status, created_at, updated_at
		FROM %s
		WHERE file_sha = ?
		ORDER BY created_at DESC
	`, tableName)

	rows, err := r.db.Query(query, fileSHA)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*FileVersion
	for rows.Next() {
		var fv FileVersion
		err := rows.Scan(
			&fv.FileID,
			&fv.FileSHA,
			&fv.RelativePath,
			&fv.Ephemeral,
			&fv.CommitID,
			&fv.Status,
			&fv.CreatedAt,
			&fv.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, &fv)
	}

	return files, rows.Err()
}

// GetFilesByPath retrieves all file versions for a specific path
func (r *FileVersionRepository) GetFilesByPath(relativePath string) ([]*FileVersion, error) {
	tableName := r.tableName()

	query := fmt.Sprintf(`
		SELECT file_id, file_sha, relative_path, ephemeral, commit_id, status, created_at, updated_at
		FROM %s
		WHERE relative_path = ?
		ORDER BY created_at DESC
	`, tableName)

	rows, err := r.db.Query(query, relativePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*FileVersion
	for rows.Next() {
		var fv FileVersion
		err := rows.Scan(
			&fv.FileID,
			&fv.FileSHA,
			&fv.RelativePath,
			&fv.Ephemeral,
			&fv.CommitID,
			&fv.Status,
			&fv.CreatedAt,
			&fv.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, &fv)
	}

	return files, rows.Err()
}

// DeleteEphemeralVersions deletes all ephemeral file versions
func (r *FileVersionRepository) DeleteEphemeralVersions() (int64, error) {
	tableName := r.tableName()

	r.logger.Info("Deleting ephemeral file versions", zap.String("table", tableName))

	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE ephemeral = TRUE
	`, tableName)

	result, err := r.db.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete ephemeral versions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	r.logger.Info("Deleted ephemeral file versions",
		zap.Int64("count", rowsAffected),
		zap.String("table", tableName))

	return rowsAffected, nil
}

// UpdateStatus updates the processing status of a file version
func (r *FileVersionRepository) UpdateStatus(fileID int32, status string) error {
	tableName := r.tableName()

	query := fmt.Sprintf(`
		UPDATE %s
		SET status = ?
		WHERE file_id = ?
	`, tableName)

	_, err := r.db.Exec(query, status, fileID)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	r.logger.Debug("Updated file status",
		zap.Int32("file_id", fileID),
		zap.String("status", status))

	return nil
}

// GetStats returns statistics about the file versions
func (r *FileVersionRepository) GetStats() (total int64, ephemeral int64, committed int64, err error) {
	tableName := r.tableName()

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN ephemeral = TRUE THEN 1 ELSE 0 END) as ephemeral,
			SUM(CASE WHEN ephemeral = FALSE THEN 1 ELSE 0 END) as committed
		FROM %s
	`, tableName)

	err = r.db.QueryRow(query).Scan(&total, &ephemeral, &committed)
	return
}

// DropTable drops the file_versions table for this repository.
// This permanently deletes all file version tracking data for the repository.
func (r *FileVersionRepository) DropTable() error {
	tableName := r.tableName()

	r.logger.Info("Dropping file versions table", zap.String("table", tableName))

	query := fmt.Sprintf(`DROP TABLE IF EXISTS %s`, tableName)

	_, err := r.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to drop table %s: %w", tableName, err)
	}

	r.logger.Info("File versions table dropped successfully", zap.String("table", tableName))
	return nil
}
