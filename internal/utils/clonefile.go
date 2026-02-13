package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// FileCloner is a utility to clone files.
type FileCloner struct {
	TempDir          string
	FilenamePrefix   string
	SysClonefileFunc func(src, dst string, flags int) error
}

// NewFileCloner creates a new FileCloner with default values.
func NewFileCloner() *FileCloner {
	return &FileCloner{
		TempDir:          os.TempDir(),
		FilenamePrefix:   "macosvz_file_",
		SysClonefileFunc: unix.Clonefile,
	}
}

// Clonefile clones the file at the given path and returns the path of the cloned file.
func (fc *FileCloner) Clonefile(path, pattern string) (string, error) {
	clonedPath := filepath.Join(fc.TempDir, fc.FilenamePrefix+filepath.Base(path)+"."+pattern)

	// remove the overlay storage file if it already exists
	_ = os.Remove(clonedPath)
	if err := fc.SysClonefileFunc(path, clonedPath, 0); err != nil {
		return "", fmt.Errorf("failed to clone storage file: %w", err)
	}

	return clonedPath, nil
}
