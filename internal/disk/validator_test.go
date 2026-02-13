package disk_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agoda-com/macOS-vz-kubelet/internal/disk"

	"github.com/opencontainers/go-digest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFileWithDigest(t *testing.T) {
	ctx := context.Background()

	fileContent := []byte("test data")
	testDigest := digest.FromBytes(fileContent)

	tests := []struct {
		name             string
		setup            func(t *testing.T, filePath, digestPath string) digest.Digest
		expectError      bool
		verifyDigestFile func(t *testing.T, filePath, digestPath string)
	}{
		{
			name: "No file exists",
			setup: func(t *testing.T, filePath, digestPath string) digest.Digest {
				t.Helper()
				return testDigest
			},
			expectError: true,
		},
		{
			name: "No digest file exists, digest computed manually",
			setup: func(t *testing.T, filePath, digestPath string) digest.Digest {
				t.Helper()
				return prepareFileAndDigest(t, filePath, digestPath, fileContent, false)
			},
			expectError: false,
			verifyDigestFile: func(t *testing.T, filePath, digestPath string) {
				t.Helper()
				assert.FileExists(t, digestPath, "Digest file should be created")
				storedDigest, err := os.ReadFile(digestPath)
				require.NoError(t, err)
				assert.Equal(t, testDigest.String(), string(storedDigest))
			},
		},
		{
			name: "Digest file outdated",
			setup: func(t *testing.T, filePath, digestPath string) digest.Digest {
				t.Helper()
				prepareFileAndDigest(t, filePath, digestPath, fileContent, true)
				timestamp := time.Now().Add(-2 * time.Hour)
				err := os.Chtimes(digestPath, timestamp, timestamp)
				require.NoError(t, err)
				return testDigest
			},
			expectError: false,
			verifyDigestFile: func(t *testing.T, filePath, digestPath string) {
				t.Helper()
				assert.FileExists(t, digestPath, "Digest file should be updated")
				storedDigest, err := os.ReadFile(digestPath)
				require.NoError(t, err)
				assert.Equal(t, testDigest.String(), string(storedDigest))
			},
		},
		{
			name: "Successful digest validation",
			setup: func(t *testing.T, filePath, digestPath string) digest.Digest {
				t.Helper()
				return prepareFileAndDigest(t, filePath, digestPath, fileContent, true)
			},
			expectError: false,
		},
		{
			name: "Digest mismatch",
			setup: func(t *testing.T, filePath, digestPath string) digest.Digest {
				t.Helper()
				prepareFileAndDigest(t, filePath, digestPath, fileContent, false)
				require.NoError(t, os.WriteFile(digestPath, []byte("invalid_digest"), 0644))
				return testDigest
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filePath := filepath.Join(t.TempDir(), "file.txt")
			digestPath := filePath + disk.DigestFileSuffix
			expectedDigest := tc.setup(t, filePath, digestPath)

			err := disk.ValidateFileWithDigest(ctx, filePath, expectedDigest)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tc.verifyDigestFile != nil {
				tc.verifyDigestFile(t, filePath, digestPath)
			}
		})
	}
}

// prepareFileAndDigest sets up a test file and optionally a digest file.
func prepareFileAndDigest(t *testing.T, filePath, digestPath string, content []byte, prepareDigest bool) digest.Digest {
	t.Helper()
	d := digest.FromBytes(content)
	require.NoError(t, os.WriteFile(filePath, content, 0644))
	if prepareDigest {
		require.NoError(t, os.WriteFile(digestPath, []byte(d.String()), 0644))
	}
	return d
}
