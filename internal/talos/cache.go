package talos

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// CacheManager handles caching of Talos images.
type CacheManager struct {
	cacheDir string
	maxSize  int64
	logger   *slog.Logger
}

// NewCacheManager creates a new cache manager.
func NewCacheManager(cacheDir string, logger *slog.Logger) (*CacheManager, error) {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &CacheManager{
		cacheDir: cacheDir,
		logger:   logger,
	}, nil
}

// SetMaxSize sets the maximum cache size.
func (cm *CacheManager) SetMaxSize(maxSize int64) {
	cm.maxSize = maxSize
}

// Exists checks if a cache entry exists and has a valid checksum.
func (cm *CacheManager) Exists(key string) bool {
	filePath := cm.getFilePath(key)
	checksumPath := filePath + ".sha256"

	// Both files must exist
	if _, err := os.Stat(filePath); err != nil {
		return false
	}
	if _, err := os.Stat(checksumPath); err != nil {
		return false
	}

	return true
}

// Get returns a reader for the cached file.
func (cm *CacheManager) Get(key string) (*os.File, error) {
	filePath := cm.getFilePath(key)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cached file: %w", err)
	}
	return file, nil
}

// CreateCacheFile creates a temporary file for writing cache data.
func (cm *CacheManager) CreateCacheFile(key string) (*os.File, error) {
	filePath := cm.getFilePath(key)
	tempPath := filePath + ".tmp"

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	file, err := os.Create(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary cache file: %w", err)
	}

	return file, nil
}

// FinalizeCacheFile atomically moves the temporary file to the final location and stores checksum.
func (cm *CacheManager) FinalizeCacheFile(key, checksum string) error {
	filePath := cm.getFilePath(key)
	tempPath := filePath + ".tmp"
	checksumPath := filePath + ".sha256"
	checksumTempPath := checksumPath + ".tmp"

	// Verify the temp file exists
	if _, err := os.Stat(tempPath); err != nil {
		return fmt.Errorf("temporary file not found: %w", err)
	}

	// Write checksum to temporary checksum file
	if err := os.WriteFile(checksumTempPath, []byte(checksum+"\n"), 0o644); err != nil {
		return fmt.Errorf("failed to write checksum file: %w", err)
	}

	// Atomically rename the main file
	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(checksumTempPath) // Clean up checksum temp file on failure
		return fmt.Errorf("failed to finalize cache file: %w", err)
	}

	// Atomically rename the checksum file
	if err := os.Rename(checksumTempPath, checksumPath); err != nil {
		os.Remove(filePath) // Clean up main file on checksum failure
		return fmt.Errorf("failed to finalize checksum file: %w", err)
	}

	cm.logger.Debug("Cache file finalized", "key", key, "checksum", checksum)
	return nil
}

// DeleteCacheFile removes a cache file and its checksum.
func (cm *CacheManager) DeleteCacheFile(key string) error {
	filePath := cm.getFilePath(key)
	tempPath := filePath + ".tmp"
	checksumPath := filePath + ".sha256"
	checksumTempPath := checksumPath + ".tmp"

	// Remove all related files (ignore errors since files might not exist)
	os.Remove(filePath)
	os.Remove(tempPath)
	os.Remove(checksumPath)
	os.Remove(checksumTempPath)

	return nil
}

// Delete removes a cache entry.
func (cm *CacheManager) Delete(key string) error {
	return cm.DeleteCacheFile(key)
}

// Set stores data in the cache (for simple use cases).
func (cm *CacheManager) Set(key string, reader io.Reader) error {
	file, err := cm.CreateCacheFile(key)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	multiWriter := io.MultiWriter(file, hash)

	_, err = io.Copy(multiWriter, reader)
	if err != nil {
		cm.DeleteCacheFile(key)
		return fmt.Errorf("failed to write to cache: %w", err)
	}

	checksum := fmt.Sprintf("%x", hash.Sum(nil))
	return cm.FinalizeCacheFile(key, checksum)
}

// GetChecksum returns the stored checksum for a cache entry.
func (cm *CacheManager) GetChecksum(key string) (string, error) {
	filePath := cm.getFilePath(key)
	checksumPath := filePath + ".sha256"

	data, err := os.ReadFile(checksumPath)
	if err != nil {
		return "", fmt.Errorf("failed to read checksum file: %w", err)
	}

	// Remove trailing newline if present
	checksum := strings.TrimSpace(string(data))
	return checksum, nil
}

// VerifyChecksum calculates and verifies the checksum of a cached file.
func (cm *CacheManager) VerifyChecksum(key string) error {
	filePath := cm.getFilePath(key)

	// Get stored checksum
	storedChecksum, err := cm.GetChecksum(key)
	if err != nil {
		return fmt.Errorf("failed to get stored checksum: %w", err)
	}

	// Calculate current file checksum
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open cached file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	currentChecksum := fmt.Sprintf("%x", hash.Sum(nil))

	// Compare checksums
	if currentChecksum != storedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", storedChecksum, currentChecksum)
	}

	return nil
}

// GetVerified returns a reader for the cached file after verifying its checksum.
func (cm *CacheManager) GetVerified(key string) (*os.File, error) {
	// First verify the checksum
	if err := cm.VerifyChecksum(key); err != nil {
		cm.logger.Warn(
			"Cache integrity check failed, removing corrupted entry",
			"key",
			key,
			"error",
			err,
		)
		cm.DeleteCacheFile(key) // Remove corrupted cache entry
		return nil, fmt.Errorf("cache integrity check failed: %w", err)
	}

	// If verification passes, return the file
	return cm.Get(key)
}

// Size returns the current cache size (excluding checksum files).
func (cm *CacheManager) Size() (int64, error) {
	var totalSize int64

	err := filepath.Walk(cm.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip temporary files and checksum files
		if strings.HasSuffix(path, ".tmp") || strings.HasSuffix(path, ".sha256") {
			return nil
		}

		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return totalSize, err
}

// Cleanup removes old entries when cache exceeds limits.
func (cm *CacheManager) Cleanup() error {
	if cm.maxSize <= 0 {
		return nil // No size limit
	}

	currentSize, err := cm.Size()
	if err != nil {
		return fmt.Errorf("failed to get cache size: %w", err)
	}

	if currentSize <= cm.maxSize {
		return nil // Within limits
	}

	cm.logger.Info("Cache cleanup needed", "current_size", currentSize, "max_size", cm.maxSize)

	// TODO: Implement LRU cleanup logic if needed
	// For now, we'll just log that cleanup is needed

	return nil
}

// getFilePath returns the full file path for a cache key.
func (cm *CacheManager) getFilePath(key string) string {
	// Sanitize the key to be filesystem-safe
	safePath := strings.ReplaceAll(key, "/", "_")
	return filepath.Join(cm.cacheDir, safePath)
}
