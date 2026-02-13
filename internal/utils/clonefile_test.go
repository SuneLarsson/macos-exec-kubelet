package utils_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agoda-com/macOS-vz-kubelet/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileCloner(t *testing.T) {
	cloner := utils.NewFileCloner()
	assert.NotNil(t, cloner)
	assert.Equal(t, os.TempDir(), cloner.TempDir)
	assert.Equal(t, "macosvz_file_", cloner.FilenamePrefix)
	assert.NotNil(t, cloner.SysClonefileFunc)
}

func TestClonefile_Success(t *testing.T) {
	cloner := getTestFileCloner(t)

	srcFile, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	pattern := "test-pattern"
	expectedClonedPath := filepath.Join(cloner.TempDir, cloner.FilenamePrefix+filepath.Base(srcFile.Name())+"."+pattern)

	clonedPath, err := cloner.Clonefile(srcFile.Name(), pattern)
	require.NoError(t, err)
	assert.Equal(t, expectedClonedPath, clonedPath)
}

func TestClonefile_Failure(t *testing.T) {
	cloner := getTestFileCloner(t)

	srcPath := "/invalid/source/file"
	pattern := "test-pattern"

	clonedPath, err := cloner.Clonefile(srcPath, pattern)
	assert.Error(t, err)
	assert.Equal(t, "", clonedPath)
	assert.Contains(t, err.Error(), "failed to clone storage file")
}

func TestClonefile_FileExists(t *testing.T) {
	cloner := getTestFileCloner(t)

	srcFile, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	pattern := "test-pattern"
	expectedClonedPath := filepath.Join(cloner.TempDir, cloner.FilenamePrefix+filepath.Base(srcFile.Name())+"."+pattern)

	// Create a file at the expected cloned path to simulate the file already existing
	err = os.WriteFile(expectedClonedPath, []byte("dummy data"), 0644)
	require.NoError(t, err)

	clonedPath, err := cloner.Clonefile(srcFile.Name(), pattern)
	require.NoError(t, err)
	assert.Equal(t, expectedClonedPath, clonedPath)
}

func getTestFileCloner(t *testing.T) *utils.FileCloner {
	t.Helper()

	cloner := utils.NewFileCloner()
	cloner.TempDir = t.TempDir()
	return cloner
}
