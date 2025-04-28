// Package util provides common utilities for firmware operations.
package util

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// IsArchive checks if a file is an archive based on its extension.
func IsArchive(filename, ext string) bool {
	if ext == "" {
		ext = filepath.Ext(filename)
	}
	return ext == ".zip" || ext == ".tar" || ext == ".tgz" || ext == ".tar.gz"
}

// ExtractArchive extracts an archive to the specified directory.
func ExtractArchive(archivePath, destDir string) error {
	if !IsArchive(archivePath, "") {
		return fmt.Errorf("not an archive file: %s", archivePath)
	}

	ext := filepath.Ext(archivePath)
	switch ext {
	case ".zip":
		return extractZip(archivePath, destDir)
	case ".tar", ".tgz", ".gz":
		return extractTarGz(archivePath, destDir)
	default:
		return fmt.Errorf("unsupported archive format: %s", ext)
	}
}

// extractZip extracts a zip archive to the specified directory.
func extractZip(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		filePath := filepath.Join(destDir, file.Name)

		// Check for path traversal
		if !strings.HasPrefix(filePath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", filePath)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filePath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory structure for %s: %w", filePath, err)
		}

		outFile, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", filePath, err)
		}

		zipFile, err := file.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to open file in zip: %w", err)
		}

		_, err = io.Copy(outFile, zipFile)
		outFile.Close()
		zipFile.Close()

		if err != nil {
			return fmt.Errorf("failed to extract file %s: %w", filePath, err)
		}
	}

	return nil
}

// extractTarGz extracts a tar or tar.gz archive to the specified directory.
func extractTarGz(tarPath, destDir string) error {
	file, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	var tarReader *tar.Reader

	if strings.HasSuffix(tarPath, ".gz") || strings.HasSuffix(tarPath, ".tgz") {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		tarReader = tar.NewReader(gzipReader)
	} else {
		tarReader = tar.NewReader(file)
	}

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		filePath := filepath.Join(destDir, header.Name)

		// Check for path traversal
		if !strings.HasPrefix(filePath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", filePath)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filePath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filePath, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
				return fmt.Errorf("failed to create directory structure for %s: %w", filePath, err)
			}

			outFile, err := os.Create(filePath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", filePath, err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to extract file %s: %w", filePath, err)
			}
			outFile.Close()
		}
	}

	return nil
}

// CopyFile copies a file from src to dst.
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}
