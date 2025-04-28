package util_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmcpi/pibmc/internal/firmware/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDirs creates temporary directories for testing
func setupTestDirs(t *testing.T) (string, string) {
	srcDir, err := os.MkdirTemp("", "firmware-archive-src-*")
	require.NoError(t, err)

	destDir, err := os.MkdirTemp("", "firmware-archive-dest-*")
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(srcDir)
		os.RemoveAll(destDir)
	})

	return srcDir, destDir
}

// createTestFiles creates test files in the source directory
func createTestFiles(t *testing.T, dir string) []string {
	filePaths := []string{
		filepath.Join(dir, "file1.txt"),
		filepath.Join(dir, "file2.bin"),
		filepath.Join(dir, "subdir", "file3.txt"),
	}

	err := os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	require.NoError(t, err)

	for i, path := range filePaths {
		err := os.MkdirAll(filepath.Dir(path), 0755)
		require.NoError(t, err)

		content := []byte("test content " + string(rune('A'+i)))
		err = os.WriteFile(path, content, 0644)
		require.NoError(t, err)
	}

	return filePaths
}

// createZipArchive creates a test zip archive
func createZipArchive(t *testing.T, testFiles []string, baseDir, archivePath string) {
	zipFile, err := os.Create(archivePath)
	require.NoError(t, err)
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, file := range testFiles {
		data, err := os.ReadFile(file)
		require.NoError(t, err)

		relPath, err := filepath.Rel(baseDir, file)
		require.NoError(t, err)

		f, err := zipWriter.Create(relPath)
		require.NoError(t, err)

		_, err = f.Write(data)
		require.NoError(t, err)
	}
}

// createTarGzArchive creates a test tar.gz archive
func createTarGzArchive(t *testing.T, testFiles []string, baseDir, archivePath string) {
	tarFile, err := os.Create(archivePath)
	require.NoError(t, err)
	defer tarFile.Close()

	gzipWriter := gzip.NewWriter(tarFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, file := range testFiles {
		data, err := os.ReadFile(file)
		require.NoError(t, err)

		relPath, err := filepath.Rel(baseDir, file)
		require.NoError(t, err)

		header := &tar.Header{
			Name: relPath,
			Mode: 0644,
			Size: int64(len(data)),
		}

		err = tarWriter.WriteHeader(header)
		require.NoError(t, err)

		_, err = tarWriter.Write(data)
		require.NoError(t, err)
	}
}

func TestIsArchive(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
		expected bool
	}{
		{"ZipArchive", "file.zip", true},
		{"TarArchive", "file.tar", true},
		{"TgzArchive", "file.tgz", true},
		{"TarGzArchive", "file.tar.gz", true},
		{"BinFile", "file.bin", false},
		{"NoExtension", "file", false},
		{"EmptyString", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := util.IsArchive(tc.filename, "")
			assert.Equal(t, tc.expected, result)
		})
	}

	// Test with ext parameter
	assert.True(t, util.IsArchive("file", ".zip"))
	assert.False(t, util.IsArchive("file", ".bin"))
}

func TestExtractZip(t *testing.T) {
	srcDir, destDir := setupTestDirs(t)
	testFiles := createTestFiles(t, srcDir)

	// Create a test zip archive
	zipPath := filepath.Join(srcDir, "test.zip")
	createZipArchive(t, testFiles, srcDir, zipPath)

	// Extract the archive
	err := util.ExtractArchive(zipPath, destDir)
	require.NoError(t, err)

	// Verify extracted files
	for _, file := range testFiles {
		relPath, err := filepath.Rel(srcDir, file)
		require.NoError(t, err)

		destFile := filepath.Join(destDir, relPath)
		_, err = os.Stat(destFile)
		assert.NoError(t, err, "Extracted file should exist: %s", destFile)

		originalData, err := os.ReadFile(file)
		require.NoError(t, err)

		extractedData, err := os.ReadFile(destFile)
		require.NoError(t, err)

		assert.Equal(t, originalData, extractedData, "Extracted file content should match original")
	}
}

func TestExtractTarGz(t *testing.T) {
	srcDir, destDir := setupTestDirs(t)
	testFiles := createTestFiles(t, srcDir)

	// Create a test tar.gz archive
	tarGzPath := filepath.Join(srcDir, "test.tar.gz")
	createTarGzArchive(t, testFiles, srcDir, tarGzPath)

	// Extract the archive
	err := util.ExtractArchive(tarGzPath, destDir)
	require.NoError(t, err)

	// Verify extracted files
	for _, file := range testFiles {
		relPath, err := filepath.Rel(srcDir, file)
		require.NoError(t, err)

		destFile := filepath.Join(destDir, relPath)
		_, err = os.Stat(destFile)
		assert.NoError(t, err, "Extracted file should exist: %s", destFile)

		originalData, err := os.ReadFile(file)
		require.NoError(t, err)

		extractedData, err := os.ReadFile(destFile)
		require.NoError(t, err)

		assert.Equal(t, originalData, extractedData, "Extracted file content should match original")
	}
}

func TestExtractUnsupportedFormat(t *testing.T) {
	srcDir, destDir := setupTestDirs(t)

	// Create a test file that's not an archive
	binPath := filepath.Join(srcDir, "test.bin")
	err := os.WriteFile(binPath, []byte("test content"), 0644)
	require.NoError(t, err)

	// Try to extract it
	err = util.ExtractArchive(binPath, destDir)
	assert.Error(t, err, "Extracting a non-archive file should return an error")
}

func TestCopyFile(t *testing.T) {
	srcDir, destDir := setupTestDirs(t)

	// Create a test file
	srcFile := filepath.Join(srcDir, "testfile.txt")
	testContent := []byte("test file content")
	err := os.WriteFile(srcFile, testContent, 0644)
	require.NoError(t, err)

	// Copy the file
	destFile := filepath.Join(destDir, "copiedfile.txt")
	err = util.CopyFile(srcFile, destFile)
	require.NoError(t, err)

	// Verify the copied file
	destContent, err := os.ReadFile(destFile)
	require.NoError(t, err)
	assert.Equal(t, testContent, destContent, "Copied file content should match original")

	// Test error cases
	t.Run("SourceNotFound", func(t *testing.T) {
		err := util.CopyFile(filepath.Join(srcDir, "nonexistent.txt"), destFile)
		assert.Error(t, err)
	})

	t.Run("DestinationDirNotFound", func(t *testing.T) {
		err := util.CopyFile(srcFile, filepath.Join(destDir, "nonexistent", "file.txt"))
		assert.Error(t, err)
	})
}

func TestExtractWithPathTraversal(t *testing.T) {
	srcDir, destDir := setupTestDirs(t)

	// Create a zip file with path traversal attempt
	zipPath := filepath.Join(srcDir, "malicious.zip")
	zipFile, err := os.Create(zipPath)
	require.NoError(t, err)
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add a file with path traversal attempt
	f, err := zipWriter.Create("../outside.txt")
	require.NoError(t, err)
	_, err = f.Write([]byte("malicious content"))
	require.NoError(t, err)
	zipWriter.Close()

	// Try to extract it
	err = util.ExtractArchive(zipPath, destDir)
	assert.Error(t, err, "Extracting a file with path traversal should return an error")

	// Verify the file wasn't extracted outside the destination
	outsideFile := filepath.Join(filepath.Dir(destDir), "outside.txt")
	_, err = os.Stat(outsideFile)
	assert.True(t, os.IsNotExist(err), "File should not exist outside the destination directory")
}

func TestExtractEmptyArchive(t *testing.T) {
	srcDir, destDir := setupTestDirs(t)

	// Create an empty zip file
	zipPath := filepath.Join(srcDir, "empty.zip")
	zipFile, err := os.Create(zipPath)
	require.NoError(t, err)
	zipWriter := zip.NewWriter(zipFile)
	zipWriter.Close()
	zipFile.Close()

	// Extract it
	err = util.ExtractArchive(zipPath, destDir)
	assert.NoError(t, err, "Extracting an empty archive should not return an error")
}
