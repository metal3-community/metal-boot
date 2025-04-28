package firmware

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

type FirmwareSource struct {
	Path string
	URL  string
}

func (f *FirmwareSource) IsArchive() bool {
	return isArchive("", path.Ext(f.URL))
}

type FirmwareUpdater struct {
	RootPath string
	Version  string
	Sources  []*FirmwareSource
}

func (f *FirmwareUpdater) AddSource(path, url string) {
	f.Sources = append(f.Sources, &FirmwareSource{Path: path, URL: url})
}

func (f *FirmwareUpdater) SetRootPath(path string) {
	f.RootPath = path
}

func (f *FirmwareUpdater) SetVersion(version string) {
	f.Version = version
}

// SetSources sets the firmware sources to the provided list
// If replace is true, the sources will be replaced with the provided list
// Otherwise, the sources will be appended to the existing list, replacing any existing sources with the same URL
// If the source is an archive, this function will only replace the source if the URL matches
// This is to allow multiple archive extractions to the same directory if needed
func (f *FirmwareUpdater) SetSources(replace bool, sources ...*FirmwareSource) error {
	if replace {
		f.Sources = sources
		return nil
	}

	type firmwareSourceCompare = func(s *FirmwareSource) bool

	archiveCompare := func(src *FirmwareSource) firmwareSourceCompare {
		return func(s *FirmwareSource) bool {
			if s.IsArchive() {
				return s.URL == src.URL
			}
			return s.Path == src.Path && s.URL == src.URL
		}
	}

	for _, source := range sources {
		ac := archiveCompare(source)
		if slices.ContainsFunc(f.Sources, ac) {
			i := slices.IndexFunc(f.Sources, ac)
			if i != -1 {
				f.Sources[i] = f.Sources[len(f.Sources)-1]
				f.Sources = f.Sources[:len(f.Sources)-1]
			}
		}
		f.Sources = append(f.Sources, source)
	}

	return nil
}

// UpdateFirmware downloads all firmware sources and extracts archives directly
func (f *FirmwareUpdater) UpdateFirmware() error {
	if len(f.Sources) == 0 {
		return errors.New("no firmware sources defined")
	}

	for _, source := range f.Sources {
		if err := f.downloadAndExtract(source.URL, source.Path); err != nil {
			return fmt.Errorf("failed to process source %s: %w", source.URL, err)
		}
	}

	return nil
}

// downloadAndExtract downloads a file from url and extracts it if it's an archive
func (f *FirmwareUpdater) downloadAndExtract(url, destPath string) error {
	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll(destPath, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", destPath, err)
	}

	// Create HTTP request
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	// Determine content type and extension
	contentType := resp.Header.Get("Content-Type")
	urlPath := resp.Request.URL.Path
	extension := strings.ToLower(path.Ext(urlPath))

	// Handle based on file type
	if isArchive(contentType, extension) {
		if err := extractArchive(resp.Body, contentType, extension, destPath); err != nil {
			return fmt.Errorf("failed to extract archive: %w", err)
		}
		return nil
	}

	// Not an archive, save directly to file
	filename := filepath.Base(urlPath)
	if filename == "" || filename == "." {
		filename = "firmware"
	}
	outPath := filepath.Join(destPath, filename)

	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outPath, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to save file %s: %w", outPath, err)
	}

	return nil
}

// isArchive checks if the content is an archive based on content type or extension
func isArchive(contentType, extension string) bool {
	// Check by content type
	if strings.Contains(contentType, "zip") ||
		strings.Contains(contentType, "tar") ||
		strings.Contains(contentType, "gzip") ||
		strings.Contains(contentType, "x-compress") ||
		strings.Contains(contentType, "x-bzip2") {
		return true
	}

	// Check by extension
	switch extension {
	case ".zip", ".tar", ".tgz", ".gz", ".tar.gz", ".bz2", ".tar.bz2", ".xz", ".tar.xz":
		return true
	default:
		return false
	}
}

// extractArchive extracts the archive to the destination path
func extractArchive(reader io.Reader, contentType, extension, destPath string) error {
	// Handle ZIP files
	if strings.Contains(contentType, "zip") || extension == ".zip" {
		// Create a temporary file for zip content
		tmpFile, err := os.CreateTemp("", "firmware-*.zip")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		// Copy zip content to temporary file
		if _, err := io.Copy(tmpFile, reader); err != nil {
			return fmt.Errorf("failed to copy zip content: %w", err)
		}

		// Close the file to ensure all data is written
		tmpFile.Close()

		// Open the zip file for reading
		zipReader, err := zip.OpenReader(tmpFile.Name())
		if err != nil {
			return fmt.Errorf("failed to open zip file: %w", err)
		}
		defer zipReader.Close()

		// Extract each file
		for _, file := range zipReader.File {
			if err := extractZipFile(file, destPath); err != nil {
				return err
			}
		}
		return nil
	}

	// Handle tar and tar.gz files
	if strings.Contains(contentType, "tar") || strings.Contains(contentType, "gzip") ||
		extension == ".tar" || extension == ".tgz" || extension == ".tar.gz" || extension == ".gz" {

		var tarReader *tar.Reader
		if strings.Contains(contentType, "gzip") || extension == ".tgz" || extension == ".tar.gz" ||
			extension == ".gz" {
			gzReader, err := gzip.NewReader(reader)
			if err != nil {
				return fmt.Errorf("failed to create gzip reader: %w", err)
			}
			defer gzReader.Close()
			tarReader = tar.NewReader(gzReader)
		} else {
			tarReader = tar.NewReader(reader)
		}

		return extractTarArchive(tarReader, destPath)
	}

	return fmt.Errorf("unsupported archive format: %s", contentType)
}

// extractZipFile extracts a single file from a zip archive
func extractZipFile(file *zip.File, destPath string) error {
	// Skip directories
	if file.FileInfo().IsDir() {
		filePath := filepath.Join(destPath, file.Name)
		if err := os.MkdirAll(filePath, file.Mode()); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filePath, err)
		}
		return nil
	}

	// Create destination file
	filePath := filepath.Join(destPath, file.Name)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", filePath, err)
	}

	// Open the file for reading
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open zip file entry: %w", err)
	}
	defer src.Close()

	// Create the destination file
	dest, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer dest.Close()

	// Copy contents
	if _, err := io.Copy(dest, src); err != nil {
		return fmt.Errorf("failed to extract file %s: %w", filePath, err)
	}

	return nil
}

// extractTarArchive extracts files from a tar archive
func extractTarArchive(tarReader *tar.Reader, destPath string) error {
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar reading error: %w", err)
		}

		path := filepath.Join(destPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0o755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", path, err)
			}
		case tar.TypeReg:
			// Ensure directory exists
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", path, err)
			}

			// Create the file
			file, err := os.OpenFile(
				path,
				os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
				os.FileMode(header.Mode),
			)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", path, err)
			}

			// Copy content
			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return fmt.Errorf("failed to extract file %s: %w", path, err)
			}
			file.Close()
		}
	}
	return nil
}
