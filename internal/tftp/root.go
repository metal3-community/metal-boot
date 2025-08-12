package tftp

import (
	"io/fs"
	"os"
	"path/filepath"
)

// Root provides a rooted filesystem for TFTP operations.
type Root struct {
	root string
}

// NewRoot creates a new Root rooted at the given path.
func NewRoot(root string) (*Root, error) {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, err
	}
	return &Root{
		root: root,
	}, nil
}

// Open opens a file from the rooted filesystem.
func (r *Root) Open(name string) (fs.File, error) {
	return os.Open(filepath.Join(r.root, name))
}

// Create creates a file in the rooted filesystem.
func (r *Root) Create(name string) (*os.File, error) {
	return os.Create(filepath.Join(r.root, name))
}

// MkdirAll creates a directory path in the rooted filesystem.
func (r *Root) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(filepath.Join(r.root, path), perm)
}

// Stat returns the FileInfo for a file in the rooted filesystem.
func (r *Root) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(filepath.Join(r.root, name))
}

// Exists checks if a path exists in the rooted filesystem.
func (r *Root) Exists(path string) bool {
	_, err := r.Stat(path)
	return err == nil
}

// Close is a no-op for Root.
func (r *Root) Close() error {
	return nil // Nothing to close
}

// OpenFile opens a file from the rooted filesystem with the specified flag.
func (r *Root) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(filepath.Join(r.root, name), flag, perm)
}

// Lstat returns a FileInfo describing the named file.
func (r *Root) Lstat(name string) (fs.FileInfo, error) {
	return os.Lstat(filepath.Join(r.root, name))
}

// Mkdir creates a new directory with the specified name and permission bits.
func (r *Root) Mkdir(name string, perm os.FileMode) error {
	return os.Mkdir(filepath.Join(r.root, name), perm)
}
