package util

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"go.uber.org/zap"
)

// WalkFunc is called for each file or directory visited during the walk.
// If the function returns an error, the walk will stop.
type WalkFunc func(path string, err error) error
type SkipFunc func(path string, isDir bool) bool

type walkItem struct {
	path string
	//info fs.FileInfo
}

// Walk traverses a directory tree concurrently, calling walkFn for each file
// and directory. Unlike filepath.Walk, this implementation does not guarantee
// any particular ordering of children directories and files.
// gcThreshold controls how often to trigger GC (every N files). Set to 0 to disable.
// Uses 2 worker goroutines to process files concurrently.
func WalkDirTree(root string, walkFn WalkFunc, skipPath SkipFunc, logger *zap.Logger, gcThreshold int64, numThreads int) error {
	processedCount := int64(0)
	// Create channels for work distribution
	workQueue := make(chan walkItem, 2)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Start 2 worker goroutines
	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range workQueue {
				// Increment processed count
				mu.Lock()
				processedCount++
				mu.Unlock()

				// Trigger GC if needed
				if gcThreshold > 0 && processedCount%gcThreshold == 0 {
					logger.Info("WalkDirTree - Triggering GC after processing files",
						zap.Int64("files_processed", processedCount))
					runtime.GC()
				}

				// Call the walk function
				err := walkFn(item.path, nil)
				if err != nil {
					logger.Error("WalkDirTree - Failed to process file", zap.String("path", item.path), zap.Error(err))
					if err != filepath.SkipDir {
						/*select {
						case errorChan <- err:
						default:
						}*/
						// Continue processing other files even if one fails
					}
				}
			}
		}()
	}

	// Walk the directory tree and send items to workers
	_, err := os.Lstat(root)
	if err != nil {
		logger.Error("WalkDirTree - Failed to stat root", zap.String("path", root), zap.Error(err))
		// Send error item to work queue
		return nil
	}

	processedCount = 0

	err = walk(root, workQueue, skipPath, &processedCount, gcThreshold, logger)
	close(workQueue)

	// Wait for all workers to finish
	wg.Wait()

	// Check for errors
	if err != nil {
		return err
	}

	return nil
}

// walk recursively traverses the directory tree and sends items to the work queue
func walk(path string, fileQueue chan<- walkItem, skipPath SkipFunc, processedCount *int64, gcThreshold int64, logger *zap.Logger) error {
	// This must be a directory. Don't call for files
	if skipPath(path, true) {
		logger.Info("WalkDirTree - Skipping path", zap.String("path", path))
		return filepath.SkipDir
	}

	// Read directory entries
	entries, err := os.ReadDir(path)
	if err != nil {
		logger.Error("WalkDirTree - Failed to read directory", zap.String("path", path), zap.Error(err))
		return nil
	}

	// Recursively walk child entries
	for _, entry := range entries {
		childPath := filepath.Join(path, entry.Name())
		*processedCount++

		if gcThreshold > 0 && *processedCount%gcThreshold == 0 {
			logger.Info("WalkDirTree - Triggering GC after processing files",
				zap.Int64("files_processed", *processedCount))
			runtime.GC()
		}

		if !entry.IsDir() {
			if skipPath(childPath, false) {
				logger.Info("WalkDirTree - Skipping file", zap.String("path", childPath))
				continue
			}
			fileQueue <- walkItem{path: childPath}
		} else {
			walk(childPath, fileQueue, skipPath, processedCount, gcThreshold, logger)
		}
	}

	return nil
}
