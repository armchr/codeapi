package util

import (
	"github.com/armchr/codeapi/internal/config"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/bits-and-blooms/bloom/v3"
	"go.uber.org/zap"
)

// BloomFilterManager manages bloom filters for repositories with disk persistence
type BloomFilterManager struct {
	config     config.BloomFilterConfig
	filters    map[string]*bloom.BloomFilter
	mu         sync.RWMutex
	logger     *zap.Logger
	storageDir string
}

// NewBloomFilterManager creates a new bloom filter manager
func NewBloomFilterManager(cfg config.BloomFilterConfig, logger *zap.Logger) (*BloomFilterManager, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("bloom filter is disabled in config")
	}

	// Set defaults
	if cfg.ExpectedItems == 0 {
		cfg.ExpectedItems = 1000000 // Default: 1 million items
	}
	if cfg.FalsePositiveRate == 0 {
		cfg.FalsePositiveRate = 0.01 // Default: 1% false positive rate
	}
	if cfg.StorageDir == "" {
		cfg.StorageDir = "./bloom_filters"
	}

	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(cfg.StorageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create bloom filter storage directory: %w", err)
	}

	return &BloomFilterManager{
		config:     cfg,
		filters:    make(map[string]*bloom.BloomFilter),
		logger:     logger,
		storageDir: cfg.StorageDir,
	}, nil
}

// GetOrCreateFilter gets or creates a bloom filter for a repository
func (bfm *BloomFilterManager) GetOrCreateFilter(repoName string) (*bloom.BloomFilter, error) {
	bfm.mu.RLock()
	filter, exists := bfm.filters[repoName]
	bfm.mu.RUnlock()

	if exists {
		return filter, nil
	}

	// Try to load from disk first
	bfm.mu.Lock()
	defer bfm.mu.Unlock()

	// Double-check after acquiring write lock
	if filter, exists := bfm.filters[repoName]; exists {
		return filter, nil
	}

	filterPath := bfm.getFilterPath(repoName)
	filter, err := bfm.loadFromDisk(filterPath)
	if err != nil {
		// Create new filter if load fails
		bfm.logger.Info("Creating new bloom filter for repository",
			zap.String("repo", repoName),
			zap.Uint("expected_items", bfm.config.ExpectedItems),
			zap.Float64("false_positive_rate", bfm.config.FalsePositiveRate))

		filter = bloom.NewWithEstimates(bfm.config.ExpectedItems, bfm.config.FalsePositiveRate)
	} else {
		bfm.logger.Info("Loaded bloom filter from disk",
			zap.String("repo", repoName),
			zap.String("path", filterPath))
	}

	bfm.filters[repoName] = filter
	return filter, nil
}

// Add adds data to the bloom filter for a repository
func (bfm *BloomFilterManager) Add(repoName string, data string) error {
	filter, err := bfm.GetOrCreateFilter(repoName)
	if err != nil {
		return err
	}

	filter.AddString(data)
	return nil
}

// Test checks if data might exist in the bloom filter for a repository
// Returns true if data might exist (or false positive), false if definitely doesn't exist
func (bfm *BloomFilterManager) Test(repoName string, data string) (bool, error) {
	filter, err := bfm.GetOrCreateFilter(repoName)
	if err != nil {
		return false, err
	}

	return filter.TestString(data), nil
}

// Save persists the bloom filter for a repository to disk
func (bfm *BloomFilterManager) Save(repoName string) error {
	bfm.mu.RLock()
	filter, exists := bfm.filters[repoName]
	bfm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no bloom filter found for repository: %s", repoName)
	}

	filterPath := bfm.getFilterPath(repoName)
	return bfm.saveToDisk(filter, filterPath)
}

// SaveAll persists all bloom filters to disk
func (bfm *BloomFilterManager) SaveAll() error {
	bfm.mu.RLock()
	defer bfm.mu.RUnlock()

	for repoName, filter := range bfm.filters {
		filterPath := bfm.getFilterPath(repoName)
		if err := bfm.saveToDisk(filter, filterPath); err != nil {
			bfm.logger.Error("Failed to save bloom filter",
				zap.String("repo", repoName),
				zap.Error(err))
			return err
		}
		bfm.logger.Info("Saved bloom filter to disk",
			zap.String("repo", repoName),
			zap.String("path", filterPath))
	}

	return nil
}

// getFilterPath returns the file path for a repository's bloom filter
func (bfm *BloomFilterManager) getFilterPath(repoName string) string {
	return filepath.Join(bfm.storageDir, fmt.Sprintf("%s.bloom", repoName))
}

// saveToDisk saves a bloom filter to disk
func (bfm *BloomFilterManager) saveToDisk(filter *bloom.BloomFilter, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create bloom filter file: %w", err)
	}
	defer file.Close()

	_, err = filter.WriteTo(file)
	if err != nil {
		return fmt.Errorf("failed to write bloom filter: %w", err)
	}

	return nil
}

// loadFromDisk loads a bloom filter from disk
func (bfm *BloomFilterManager) loadFromDisk(path string) (*bloom.BloomFilter, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open bloom filter file: %w", err)
	}
	defer file.Close()

	filter := &bloom.BloomFilter{}
	_, err = filter.ReadFrom(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read bloom filter: %w", err)
	}

	return filter, nil
}

// Clear removes the bloom filter for a repository from memory
func (bfm *BloomFilterManager) Clear(repoName string) {
	bfm.mu.Lock()
	defer bfm.mu.Unlock()

	delete(bfm.filters, repoName)
	bfm.logger.Info("Cleared bloom filter from memory", zap.String("repo", repoName))
}

// ClearAll removes all bloom filters from memory
func (bfm *BloomFilterManager) ClearAll() {
	bfm.mu.Lock()
	defer bfm.mu.Unlock()

	bfm.filters = make(map[string]*bloom.BloomFilter)
	bfm.logger.Info("Cleared all bloom filters from memory")
}

// Delete removes the bloom filter for a repository from both memory and disk
func (bfm *BloomFilterManager) Delete(repoName string) error {
	bfm.Clear(repoName)

	filterPath := bfm.getFilterPath(repoName)
	if err := os.Remove(filterPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete bloom filter file: %w", err)
	}

	bfm.logger.Info("Deleted bloom filter", zap.String("repo", repoName))
	return nil
}
