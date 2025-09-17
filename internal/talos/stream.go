package talos

import (
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"net/http"
)

// StreamAndCacheWriter writes to both HTTP response and cache simultaneously.
type StreamAndCacheWriter struct {
	responseWriter http.ResponseWriter
	cacheWriter    io.Writer
	hash           hash.Hash
	totalBytes     int64
}

// NewStreamAndCacheWriter creates a new dual writer.
func NewStreamAndCacheWriter(w http.ResponseWriter, cacheWriter io.Writer) *StreamAndCacheWriter {
	return &StreamAndCacheWriter{
		responseWriter: w,
		cacheWriter:    cacheWriter,
		hash:           sha256.New(),
	}
}

// Write implements io.Writer interface.
func (s *StreamAndCacheWriter) Write(p []byte) (n int, err error) {
	// Write to hash for checksum
	s.hash.Write(p)

	// Write to response
	n1, err1 := s.responseWriter.Write(p)

	// Write to cache
	n2, err2 := s.cacheWriter.Write(p)

	s.totalBytes += int64(n1)

	// Return the minimum bytes written and any error
	if err1 != nil {
		return n1, err1
	}
	if err2 != nil {
		return n2, err2
	}

	return minimum(n1, n2), nil
}

// Checksum returns the SHA256 checksum of all written data.
func (s *StreamAndCacheWriter) Checksum() string {
	return fmt.Sprintf("%x", s.hash.Sum(nil))
}

// TotalBytes returns the total number of bytes written.
func (s *StreamAndCacheWriter) TotalBytes() int64 {
	return s.totalBytes
}

// minimum returns the minimum of two integers.
func minimum(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// StreamCompressedImage downloads and decompresses an image while streaming to response and cache.
func StreamCompressedImage(
	w http.ResponseWriter,
	imageURL string,
	cacheWriter io.Writer,
) (*StreamAndCacheWriter, error) {
	// Download compressed image
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	// Create gzip reader for decompression
	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Create dual writer
	dualWriter := NewStreamAndCacheWriter(w, cacheWriter)

	// Stream decompressed data to both response and cache
	_, err = io.Copy(dualWriter, gzipReader)
	if err != nil {
		return nil, fmt.Errorf("failed to stream image: %w", err)
	}

	return dualWriter, nil
}
