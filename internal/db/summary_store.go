package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/armchr/codeapi/internal/service/summary"
	"go.uber.org/zap"
)

// SummaryStore manages code summary storage in MySQL
type SummaryStore struct {
	db       *sql.DB
	repoName string
	logger   *zap.Logger
}

// NewSummaryStore creates a new summary store for a repository
func NewSummaryStore(db *sql.DB, repoName string, logger *zap.Logger) (*SummaryStore, error) {
	store := &SummaryStore{
		db:       db,
		repoName: repoName,
		logger:   logger,
	}

	// Ensure the table exists
	if err := store.EnsureTable(); err != nil {
		return nil, fmt.Errorf("failed to ensure table: %w", err)
	}

	return store, nil
}

// tableName returns the sanitized table name for this repository
func (s *SummaryStore) tableName() string {
	sanitized := sanitizeTableName(s.repoName)
	return fmt.Sprintf("`%s_code_summaries`", sanitized)
}

// EnsureTable creates the code_summaries table if it doesn't exist
func (s *SummaryStore) EnsureTable() error {
	tableName := s.tableName()
	s.logger.Info("Ensuring code_summaries table exists", zap.String("table", tableName))

	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			entity_id VARCHAR(255) NOT NULL,
			entity_type VARCHAR(50) NOT NULL,
			entity_name VARCHAR(255),
			file_path VARCHAR(500),
			summary TEXT NOT NULL,
			context_hash VARCHAR(64),
			llm_provider VARCHAR(50),
			llm_model VARCHAR(100),
			prompt_tokens INT DEFAULT 0,
			output_tokens INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY idx_entity (entity_id, entity_type),
			INDEX idx_file_path (file_path),
			INDEX idx_entity_type (entity_type),
			INDEX idx_context_hash (context_hash)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`, tableName)

	if _, err := s.db.Exec(query); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	s.logger.Info("Table ready", zap.String("table", tableName))
	return nil
}

// SaveSummary saves or updates a code summary
func (s *SummaryStore) SaveSummary(cs *summary.CodeSummary) error {
	tableName := s.tableName()

	query := fmt.Sprintf(`
		INSERT INTO %s (entity_id, entity_type, entity_name, file_path, summary, context_hash, llm_provider, llm_model, prompt_tokens, output_tokens)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			entity_name = VALUES(entity_name),
			file_path = VALUES(file_path),
			summary = VALUES(summary),
			context_hash = VALUES(context_hash),
			llm_provider = VALUES(llm_provider),
			llm_model = VALUES(llm_model),
			prompt_tokens = VALUES(prompt_tokens),
			output_tokens = VALUES(output_tokens),
			updated_at = CURRENT_TIMESTAMP
	`, tableName)

	_, err := s.db.Exec(query,
		cs.EntityID,
		cs.EntityType.String(),
		cs.EntityName,
		cs.FilePath,
		cs.Summary,
		cs.ContextHash,
		cs.LLMProvider,
		cs.LLMModel,
		cs.PromptTokens,
		cs.OutputTokens,
	)

	if err != nil {
		return fmt.Errorf("failed to save summary: %w", err)
	}

	s.logger.Debug("Saved summary",
		zap.String("entity_id", cs.EntityID),
		zap.String("entity_type", cs.EntityType.String()),
		zap.String("entity_name", cs.EntityName))

	return nil
}

// SaveSummaries saves multiple summaries in a batch
func (s *SummaryStore) SaveSummaries(summaries []*summary.CodeSummary) error {
	if len(summaries) == 0 {
		return nil
	}

	tableName := s.tableName()

	// Build batch insert query
	valueStrings := make([]string, 0, len(summaries))
	valueArgs := make([]any, 0, len(summaries)*10)

	for _, cs := range summaries {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		valueArgs = append(valueArgs,
			cs.EntityID,
			cs.EntityType.String(),
			cs.EntityName,
			cs.FilePath,
			cs.Summary,
			cs.ContextHash,
			cs.LLMProvider,
			cs.LLMModel,
			cs.PromptTokens,
			cs.OutputTokens,
		)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (entity_id, entity_type, entity_name, file_path, summary, context_hash, llm_provider, llm_model, prompt_tokens, output_tokens)
		VALUES %s
		ON DUPLICATE KEY UPDATE
			entity_name = VALUES(entity_name),
			file_path = VALUES(file_path),
			summary = VALUES(summary),
			context_hash = VALUES(context_hash),
			llm_provider = VALUES(llm_provider),
			llm_model = VALUES(llm_model),
			prompt_tokens = VALUES(prompt_tokens),
			output_tokens = VALUES(output_tokens),
			updated_at = CURRENT_TIMESTAMP
	`, tableName, strings.Join(valueStrings, ","))

	_, err := s.db.Exec(query, valueArgs...)
	if err != nil {
		return fmt.Errorf("failed to save summaries batch: %w", err)
	}

	s.logger.Debug("Saved summaries batch", zap.Int("count", len(summaries)))
	return nil
}

// GetSummary retrieves a summary by entity ID and type
func (s *SummaryStore) GetSummary(entityID string, entityType summary.SummaryLevel) (*summary.CodeSummary, error) {
	tableName := s.tableName()

	query := fmt.Sprintf(`
		SELECT id, entity_id, entity_type, entity_name, file_path, summary, context_hash, llm_provider, llm_model, prompt_tokens, output_tokens, created_at, updated_at
		FROM %s
		WHERE entity_id = ? AND entity_type = ?
	`, tableName)

	var cs summary.CodeSummary
	var entityTypeStr string
	err := s.db.QueryRow(query, entityID, entityType.String()).Scan(
		&cs.ID,
		&cs.EntityID,
		&entityTypeStr,
		&cs.EntityName,
		&cs.FilePath,
		&cs.Summary,
		&cs.ContextHash,
		&cs.LLMProvider,
		&cs.LLMModel,
		&cs.PromptTokens,
		&cs.OutputTokens,
		&cs.CreatedAt,
		&cs.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get summary: %w", err)
	}

	cs.EntityType = summary.ParseSummaryLevel(entityTypeStr)
	return &cs, nil
}

// GetSummariesByFile retrieves all summaries for a file path
func (s *SummaryStore) GetSummariesByFile(filePath string) ([]*summary.CodeSummary, error) {
	tableName := s.tableName()

	query := fmt.Sprintf(`
		SELECT id, entity_id, entity_type, entity_name, file_path, summary, context_hash, llm_provider, llm_model, prompt_tokens, output_tokens, created_at, updated_at
		FROM %s
		WHERE file_path = ?
		ORDER BY entity_type, entity_name
	`, tableName)

	return s.querySummaries(query, filePath)
}

// GetSummariesByType retrieves all summaries of a specific type
func (s *SummaryStore) GetSummariesByType(entityType summary.SummaryLevel) ([]*summary.CodeSummary, error) {
	tableName := s.tableName()

	query := fmt.Sprintf(`
		SELECT id, entity_id, entity_type, entity_name, file_path, summary, context_hash, llm_provider, llm_model, prompt_tokens, output_tokens, created_at, updated_at
		FROM %s
		WHERE entity_type = ?
		ORDER BY entity_name
	`, tableName)

	return s.querySummaries(query, entityType.String())
}

// GetAllSummaries retrieves all summaries
func (s *SummaryStore) GetAllSummaries() ([]*summary.CodeSummary, error) {
	tableName := s.tableName()

	query := fmt.Sprintf(`
		SELECT id, entity_id, entity_type, entity_name, file_path, summary, context_hash, llm_provider, llm_model, prompt_tokens, output_tokens, created_at, updated_at
		FROM %s
		ORDER BY entity_type, entity_name
	`, tableName)

	return s.querySummaries(query)
}

// querySummaries is a helper to execute a query and return summaries
func (s *SummaryStore) querySummaries(query string, args ...any) ([]*summary.CodeSummary, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query summaries: %w", err)
	}
	defer rows.Close()

	var summaries []*summary.CodeSummary
	for rows.Next() {
		var cs summary.CodeSummary
		var entityTypeStr string
		err := rows.Scan(
			&cs.ID,
			&cs.EntityID,
			&entityTypeStr,
			&cs.EntityName,
			&cs.FilePath,
			&cs.Summary,
			&cs.ContextHash,
			&cs.LLMProvider,
			&cs.LLMModel,
			&cs.PromptTokens,
			&cs.OutputTokens,
			&cs.CreatedAt,
			&cs.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan summary: %w", err)
		}
		cs.EntityType = summary.ParseSummaryLevel(entityTypeStr)
		summaries = append(summaries, &cs)
	}

	return summaries, rows.Err()
}

// NeedsUpdate checks if a summary needs to be regenerated based on context hash
func (s *SummaryStore) NeedsUpdate(entityID string, entityType summary.SummaryLevel, contextHash string) (bool, error) {
	existing, err := s.GetSummary(entityID, entityType)
	if err != nil {
		return false, err
	}

	// No existing summary, needs creation
	if existing == nil {
		return true, nil
	}

	// Context changed, needs update
	return existing.ContextHash != contextHash, nil
}

// DeleteByFile deletes all summaries for a file path
func (s *SummaryStore) DeleteByFile(filePath string) (int64, error) {
	tableName := s.tableName()

	query := fmt.Sprintf(`DELETE FROM %s WHERE file_path = ?`, tableName)
	result, err := s.db.Exec(query, filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to delete summaries: %w", err)
	}

	return result.RowsAffected()
}

// DeleteByType deletes all summaries of a specific type
func (s *SummaryStore) DeleteByType(entityType summary.SummaryLevel) (int64, error) {
	tableName := s.tableName()

	query := fmt.Sprintf(`DELETE FROM %s WHERE entity_type = ?`, tableName)
	result, err := s.db.Exec(query, entityType.String())
	if err != nil {
		return 0, fmt.Errorf("failed to delete summaries: %w", err)
	}

	return result.RowsAffected()
}

// DeleteAll deletes all summaries for the repository
func (s *SummaryStore) DeleteAll() (int64, error) {
	tableName := s.tableName()

	query := fmt.Sprintf(`DELETE FROM %s`, tableName)
	result, err := s.db.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete all summaries: %w", err)
	}

	return result.RowsAffected()
}

// GetStats returns statistics about stored summaries
func (s *SummaryStore) GetStats() (*SummaryStats, error) {
	tableName := s.tableName()

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN entity_type = 'function' THEN 1 ELSE 0 END), 0) as functions,
			COALESCE(SUM(CASE WHEN entity_type = 'class' THEN 1 ELSE 0 END), 0) as classes,
			COALESCE(SUM(CASE WHEN entity_type = 'file' THEN 1 ELSE 0 END), 0) as files,
			COALESCE(SUM(CASE WHEN entity_type = 'folder' THEN 1 ELSE 0 END), 0) as folders,
			COALESCE(SUM(CASE WHEN entity_type = 'project' THEN 1 ELSE 0 END), 0) as projects,
			COALESCE(SUM(prompt_tokens), 0) as total_prompt_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens
		FROM %s
	`, tableName)

	var stats SummaryStats
	err := s.db.QueryRow(query).Scan(
		&stats.Total,
		&stats.Functions,
		&stats.Classes,
		&stats.Files,
		&stats.Folders,
		&stats.Projects,
		&stats.TotalPromptTokens,
		&stats.TotalOutputTokens,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &stats, nil
}

// SummaryStats holds statistics about stored summaries
type SummaryStats struct {
	Total             int64 `json:"total"`
	Functions         int64 `json:"functions"`
	Classes           int64 `json:"classes"`
	Files             int64 `json:"files"`
	Folders           int64 `json:"folders"`
	Projects          int64 `json:"projects"`
	TotalPromptTokens int64 `json:"total_prompt_tokens"`
	TotalOutputTokens int64 `json:"total_output_tokens"`
}

// DropTable drops the summaries table for this repository
func (s *SummaryStore) DropTable() error {
	tableName := s.tableName()

	s.logger.Info("Dropping code summaries table", zap.String("table", tableName))

	query := fmt.Sprintf(`DROP TABLE IF EXISTS %s`, tableName)

	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to drop table %s: %w", tableName, err)
	}

	s.logger.Info("Code summaries table dropped successfully", zap.String("table", tableName))
	return nil
}

// GetSummaryMap returns a map of entity ID to summary for quick lookups
func (s *SummaryStore) GetSummaryMap(entityType summary.SummaryLevel) (map[string]*summary.CodeSummary, error) {
	summaries, err := s.GetSummariesByType(entityType)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*summary.CodeSummary, len(summaries))
	for _, cs := range summaries {
		result[cs.EntityID] = cs
	}

	return result, nil
}

// GetRecentSummaries returns summaries updated after a given time
func (s *SummaryStore) GetRecentSummaries(since time.Time) ([]*summary.CodeSummary, error) {
	tableName := s.tableName()

	query := fmt.Sprintf(`
		SELECT id, entity_id, entity_type, entity_name, file_path, summary, context_hash, llm_provider, llm_model, prompt_tokens, output_tokens, created_at, updated_at
		FROM %s
		WHERE updated_at > ?
		ORDER BY updated_at DESC
	`, tableName)

	return s.querySummaries(query, since)
}

// GetSummariesByFileAndType retrieves summaries for a file filtered by entity type
func (s *SummaryStore) GetSummariesByFileAndType(filePath string, entityType summary.SummaryLevel) ([]*summary.CodeSummary, error) {
	tableName := s.tableName()

	query := fmt.Sprintf(`
		SELECT id, entity_id, entity_type, entity_name, file_path, summary, context_hash, llm_provider, llm_model, prompt_tokens, output_tokens, created_at, updated_at
		FROM %s
		WHERE file_path = ? AND entity_type = ?
		ORDER BY entity_name
	`, tableName)

	return s.querySummaries(query, filePath, entityType.String())
}

// GetSummaryByFileAndName retrieves a specific summary by file path, entity type and name
func (s *SummaryStore) GetSummaryByFileAndName(filePath string, entityType summary.SummaryLevel, entityName string) (*summary.CodeSummary, error) {
	tableName := s.tableName()

	query := fmt.Sprintf(`
		SELECT id, entity_id, entity_type, entity_name, file_path, summary, context_hash, llm_provider, llm_model, prompt_tokens, output_tokens, created_at, updated_at
		FROM %s
		WHERE file_path = ? AND entity_type = ? AND entity_name = ?
	`, tableName)

	var cs summary.CodeSummary
	var entityTypeStr string
	err := s.db.QueryRow(query, filePath, entityType.String(), entityName).Scan(
		&cs.ID,
		&cs.EntityID,
		&entityTypeStr,
		&cs.EntityName,
		&cs.FilePath,
		&cs.Summary,
		&cs.ContextHash,
		&cs.LLMProvider,
		&cs.LLMModel,
		&cs.PromptTokens,
		&cs.OutputTokens,
		&cs.CreatedAt,
		&cs.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get summary: %w", err)
	}

	cs.EntityType = summary.ParseSummaryLevel(entityTypeStr)
	return &cs, nil
}

// GetFileSummary retrieves the file-level summary for a file path
func (s *SummaryStore) GetFileSummary(filePath string) (*summary.CodeSummary, error) {
	tableName := s.tableName()

	query := fmt.Sprintf(`
		SELECT id, entity_id, entity_type, entity_name, file_path, summary, context_hash, llm_provider, llm_model, prompt_tokens, output_tokens, created_at, updated_at
		FROM %s
		WHERE file_path = ? AND entity_type = 'file'
	`, tableName)

	var cs summary.CodeSummary
	var entityTypeStr string
	err := s.db.QueryRow(query, filePath).Scan(
		&cs.ID,
		&cs.EntityID,
		&entityTypeStr,
		&cs.EntityName,
		&cs.FilePath,
		&cs.Summary,
		&cs.ContextHash,
		&cs.LLMProvider,
		&cs.LLMModel,
		&cs.PromptTokens,
		&cs.OutputTokens,
		&cs.CreatedAt,
		&cs.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get file summary: %w", err)
	}

	cs.EntityType = summary.ParseSummaryLevel(entityTypeStr)
	return &cs, nil
}
