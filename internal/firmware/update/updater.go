// Package update provides firmware update handling functionality.
package update

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bmcpi/pibmc/internal/firmware/types"
	"github.com/bmcpi/pibmc/internal/firmware/util"
)

// FirmwareUpdater handles firmware updates from various sources.
type FirmwareUpdater struct {
	RootPath string
	Version  string
	Sources  []*types.FirmwareSource
}

// NewFirmwareUpdater creates a new firmware updater.
func NewFirmwareUpdater(rootPath, version string) *FirmwareUpdater {
	return &FirmwareUpdater{
		RootPath: rootPath,
		Version:  version,
		Sources:  []*types.FirmwareSource{},
	}
}

// AddSource adds a firmware source to the updater.
func (f *FirmwareUpdater) AddSource(path, url string) {
	f.Sources = append(f.Sources, &types.FirmwareSource{
		Path: path,
		URL:  url,
	})
}

// DownloadAndExtract downloads firmware files and extracts them if needed.
func (f *FirmwareUpdater) DownloadAndExtract() error {
	for _, source := range f.Sources {
		if source.URL == "" {
			continue
		}

		// Create temporary download file
		tmpFile, err := os.CreateTemp("", "firmware-download-*")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		tmpPath := tmpFile.Name()
		tmpFile.Close()
		defer os.Remove(tmpPath)

		// Download the file
		if err := downloadFile(source.URL, tmpPath); err != nil {
			return fmt.Errorf("download failed: %w", err)
		}

		// Determine destination path
		destPath := source.Path
		if destPath == "" {
			destPath = filepath.Join(f.RootPath, filepath.Base(source.URL))
		}

		// If it's an archive, extract it
		if source.IsArchive() {
			extractDir := filepath.Join(f.RootPath, strings.TrimSuffix(filepath.Base(source.URL), filepath.Ext(source.URL)))
			if err := os.MkdirAll(extractDir, 0755); err != nil {
				return fmt.Errorf("failed to create extract directory: %w", err)
			}

			if err := util.ExtractArchive(tmpPath, extractDir); err != nil {
				return fmt.Errorf("extraction failed: %w", err)
			}
		} else {
			// Just copy the file to destination
			destDir := filepath.Dir(destPath)
			if err := os.MkdirAll(destDir, 0755); err != nil {
				return fmt.Errorf("failed to create destination directory: %w", err)
			}

			if err := util.CopyFile(tmpPath, destPath); err != nil {
				return fmt.Errorf("failed to copy file: %w", err)
			}
		}
	}

	return nil
}

// downloadFile downloads a file from a URL.
func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// ApplyFirmwareUpdate applies downloaded firmware to the target system.
func (f *FirmwareUpdater) ApplyFirmwareUpdate() error {
	// Implementation depends on the specific firmware update mechanism
	// This is a placeholder for system-specific update logic
	return fmt.Errorf("firmware update not implemented for this system")
}

// ValidateFilenames checks if all required firmware files are present.
func (f *FirmwareUpdater) ValidateFilenames(requiredFiles []string) []string {
	var missing []string
	for _, required := range requiredFiles {
		found := false
		for _, source := range f.Sources {
			fileName := path.Base(source.Path)
			if fileName == required {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, required)
		}
	}
	return missing
}

// GetFirmwareFiles returns the list of firmware file paths.
func (f *FirmwareUpdater) GetFirmwareFiles() []string {
	files := make([]string, 0, len(f.Sources))
	for _, source := range f.Sources {
		if !slices.Contains(files, source.Path) {
			files = append(files, source.Path)
		}
	}
	return files
}
