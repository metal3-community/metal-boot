package talos

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheManager_Basic(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "talos-cache-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cm, err := NewCacheManager(tempDir, logger)
	require.NoError(t, err)

	// Test that cache directory was created
	assert.DirExists(t, tempDir)

	// Test non-existent cache key
	assert.False(t, cm.Exists("non-existent"))

	// Test setting and getting cache entry
	testData := "test data content"
	reader := strings.NewReader(testData)

	err = cm.Set("test-key", reader)
	require.NoError(t, err)

	assert.True(t, cm.Exists("test-key"))

	// Verify checksum was stored
	checksum, err := cm.GetChecksum("test-key")
	require.NoError(t, err)
	assert.NotEmpty(t, checksum)
	assert.Len(t, checksum, 64) // SHA256 hex length

	// Verify checksum validation passes
	err = cm.VerifyChecksum("test-key")
	require.NoError(t, err)

	// Get the cached data using verified method
	file, err := cm.GetVerified("test-key")
	require.NoError(t, err)
	defer file.Close()

	content, err := io.ReadAll(file)
	require.NoError(t, err)
	assert.Equal(t, testData, string(content))

	// Test deletion
	err = cm.Delete("test-key")
	require.NoError(t, err)
	assert.False(t, cm.Exists("test-key"))
}

func TestCacheManager_CreateAndFinalize(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "talos-cache-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cm, err := NewCacheManager(tempDir, logger)
	require.NoError(t, err)

	// Create cache file
	file, err := cm.CreateCacheFile("test-key")
	require.NoError(t, err)

	// Write some data
	testData := "test data for finalization"
	_, err = file.WriteString(testData)
	require.NoError(t, err)
	file.Close()

	// File should not exist yet (it's still .tmp)
	assert.False(t, cm.Exists("test-key"))

	// Finalize the file
	err = cm.FinalizeCacheFile("test-key", "dummy-checksum")
	require.NoError(t, err)

	// Now it should exist
	assert.True(t, cm.Exists("test-key"))

	// Verify content
	cachedFile, err := cm.Get("test-key")
	require.NoError(t, err)
	defer cachedFile.Close()

	content, err := io.ReadAll(cachedFile)
	require.NoError(t, err)
	assert.Equal(t, testData, string(content))
}

func TestCacheManager_Size(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "talos-cache-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cm, err := NewCacheManager(tempDir, logger)
	require.NoError(t, err)

	// Initial size should be 0
	size, err := cm.Size()
	require.NoError(t, err)
	assert.Equal(t, int64(0), size)

	// Add some data
	testData := "test data content"
	err = cm.Set("test-key", strings.NewReader(testData))
	require.NoError(t, err)

	// Size should now reflect the data
	size, err = cm.Size()
	require.NoError(t, err)
	assert.Equal(t, int64(len(testData)), size)
}

func TestCacheManager_MaxSize(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "talos-cache-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cm, err := NewCacheManager(tempDir, logger)
	require.NoError(t, err)

	// Set max size
	cm.SetMaxSize(100)

	// Cleanup should not fail even if there's nothing to clean
	err = cm.Cleanup()
	assert.NoError(t, err)
}

func TestCacheManager_GetFilePath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "talos-cache-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cm, err := NewCacheManager(tempDir, logger)
	require.NoError(t, err)

	// Test path sanitization
	key := "v1.11.1/amd64/abc123/openstack.raw"
	safePath := cm.getFilePath(key)

	expectedPath := filepath.Join(tempDir, "v1.11.1_amd64_abc123_openstack.raw")
	assert.Equal(t, expectedPath, safePath)
}

func TestCacheManager_ChecksumHandling(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "talos-cache-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cm, err := NewCacheManager(tempDir, logger)
	require.NoError(t, err)

	testData := "test data for checksum verification"

	// Store data and verify checksum storage
	err = cm.Set("checksum-test", strings.NewReader(testData))
	require.NoError(t, err)

	// Verify checksum file exists
	checksumPath := filepath.Join(tempDir, "checksum-test.sha256")
	assert.FileExists(t, checksumPath)

	// Read and verify checksum content
	checksum, err := cm.GetChecksum("checksum-test")
	require.NoError(t, err)
	assert.Len(t, checksum, 64) // SHA256 hex string

	// Verify checksum validation passes
	err = cm.VerifyChecksum("checksum-test")
	assert.NoError(t, err)

	// Test corruption detection by modifying the cached file
	cachePath := filepath.Join(tempDir, "checksum-test")
	err = os.WriteFile(cachePath, []byte("corrupted data"), 0o644)
	require.NoError(t, err)

	// Checksum verification should now fail
	err = cm.VerifyChecksum("checksum-test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")

	// GetVerified should fail and clean up corrupted entry
	_, err = cm.GetVerified("checksum-test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache integrity check failed")

	// After corruption detection, cache entry should be removed
	assert.False(t, cm.Exists("checksum-test"))
}

func TestCacheManager_ChecksumFileAccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "talos-cache-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cm, err := NewCacheManager(tempDir, logger)
	require.NoError(t, err)

	testData := "checksum file access test"
	key := "access-test"

	// Store data
	err = cm.Set(key, strings.NewReader(testData))
	require.NoError(t, err)

	// Verify checksum is accessible via .sha256 file
	checksumPath := filepath.Join(tempDir, key+".sha256")
	checksumData, err := os.ReadFile(checksumPath)
	require.NoError(t, err)

	storedChecksum := strings.TrimSpace(string(checksumData))

	// Get checksum via manager method
	managerChecksum, err := cm.GetChecksum(key)
	require.NoError(t, err)

	// Both should match
	assert.Equal(t, storedChecksum, managerChecksum)
	assert.Len(t, managerChecksum, 64) // SHA256 hex length
}
