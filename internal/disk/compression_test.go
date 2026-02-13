package disk_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/agoda-com/macOS-vz-kubelet/internal/disk"

	"github.com/opencontainers/go-digest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompressFileWithPath(t *testing.T) {
	ctx := context.Background()

	inputFileName := createTempFileWithContent(t, "This is a test content for compression")
	fileInfo, err := os.Stat(inputFileName)
	require.NoError(t, err)

	// Create a temporary file to write compressed data
	outputFileName := filepath.Join(t.TempDir(), "compressed.gz")
	outputFile, err := os.Create(outputFileName)
	require.NoError(t, err)

	// Call the compress function
	result, err := disk.CompressFileWithPath(ctx, inputFileName, outputFile)
	assert.NoError(t, err)

	// Test output details
	assert.Greater(t, result.CompressedSize, int64(0), "Expected output file size to be greater than zero")
	assert.Equal(t, fileInfo.Size(), result.UncompressedSize, "Expected uncompressed size to be equal to the input file size")

	assert.NoError(t, result.GzDigest.Validate())
	assert.NoError(t, result.UncompressedDigest.Validate())

	// validate input file and result digest
	inputFile, err := os.Open(inputFileName)
	require.NoError(t, err)

	verifier := result.UncompressedDigest.Verifier()
	_, err = io.Copy(verifier, inputFile)
	require.NoError(t, err)

	assert.True(t, verifier.Verified(), "Expected the compressed file to be verified")

	// validate compressed file and result digest
	_, err = outputFile.Seek(0, 0)
	require.NoError(t, err)

	gzVerifier := result.GzDigest.Verifier()
	_, err = io.Copy(gzVerifier, outputFile)
	require.NoError(t, err)

	assert.True(t, gzVerifier.Verified(), "Expected the compressed file to be verified")
}

func TestDecompressFileWithPath(t *testing.T) {
	ctx := context.Background()
	testInput := "This is a test content for decompression"

	// Create compressed input file (first need to compress something)
	inputFileName := createTempFileWithContent(t, testInput)

	compressedInputFile, err := os.CreateTemp(t.TempDir(), "*.gz")
	require.NoError(t, err)

	compressedResult, err := disk.CompressFileWithPath(ctx, inputFileName, compressedInputFile)
	assert.NoError(t, err)
	require.NoError(t, compressedInputFile.Close())

	// Create an output file
	outputFileName := filepath.Join(os.TempDir(), "decompressed.txt")

	// Call the decompression function
	digest, err := disk.DecompressFileWithPath(ctx, compressedInputFile.Name(), outputFileName, 0)
	assert.NoError(t, err)

	// Validate the computed digest
	assert.NoError(t, digest.Validate())
	assert.Equal(t, compressedResult.UncompressedDigest, digest, "Expected the digest to be equal to the computed digest")

	decompressedContent, err := os.ReadFile(outputFileName)
	require.NoError(t, err)
	assert.Equal(t, testInput, string(decompressedContent), "Expected decompressed content to be equal to test input")

	// validate cached digest
	digestFilePath := outputFileName + disk.DigestFileSuffix
	assert.FileExists(t, digestFilePath, "Expected digest file to be created")

	// Test if the digest file contains the correct digest
	digestFile, err := os.ReadFile(digestFilePath)
	require.NoError(t, err)
	assert.Equal(t, string(compressedResult.UncompressedDigest), string(digestFile))
}

func TestDecompressSkippingZeroChunks(t *testing.T) {
	ctx := context.Background()

	// Step 1: Create a test input file with embedded zero chunks
	inputFilePath := filepath.Join(t.TempDir(), "test.gz")
	outputFilePath := filepath.Join(t.TempDir(), "output.txt")

	// Create gzip data with non-zero data, zero chunks, and more data
	content := []byte{1, 2, 3}
	content = append(content, make([]byte, 1024)...) // Adding a 1KB zero chunk
	content = append(content, 4, 5, 6)

	buf := bytes.Buffer{}
	gw := gzip.NewWriter(&buf)
	_, err := gw.Write(content)
	require.NoError(t, err)
	require.NoError(t, gw.Close())

	err = os.WriteFile(inputFilePath, buf.Bytes(), 0644)
	require.NoError(t, err)

	uncompressedSize := int64(len(content))

	// Step 2: Decompress the file
	d, err := disk.DecompressFileWithPath(ctx, inputFilePath, outputFilePath, uncompressedSize)
	require.NoError(t, err)

	// Step 3: Read the output file
	outputData, err := os.ReadFile(outputFilePath)
	require.NoError(t, err)

	// Check the uncompressed data size with the expected size, including zero chunks
	assert.Equal(t, uncompressedSize, int64(len(outputData)), "Uncompressed file size should include zero chunks")

	// Step 4: Check digest
	expectedDigest := digest.FromBytes(content)
	assert.Equal(t, expectedDigest, d, "Digest should reflect the entire original data including zero chunks")

	// Step 5: Explicitly verify content in the areas where zero chunks are expected
	// Reading the middle 1KB supposed to be zeros
	if len(outputData) > 1024+3 { // check exists to avoid panics
		middleContent := outputData[3 : 3+1024]
		for _, b := range middleContent {
			assert.Equal(t, byte(0), b, "Expected zero bytes in the middle of the output file")
		}
	}
}

// Helper function to create a temporary file with some content
func createTempFileWithContent(t *testing.T, content string) string {
	t.Helper()

	tmpfile, err := os.CreateTemp(t.TempDir(), "example.*.txt")
	require.NoError(t, err)

	_, err = tmpfile.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	return tmpfile.Name()
}
