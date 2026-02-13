package disk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/opencontainers/go-digest"
	"github.com/virtual-kubelet/virtual-kubelet/log"
)

// ValidateFileWithDigest validates the file at the given path against the expected digest.
// If the digest file exists and is up-to-date, the digest is read from the file and compared against the expected digest.
func ValidateFileWithDigest(ctx context.Context, filePath string, expectedDigest digest.Digest) error {
	// Check the file existence
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("error checking file: %w", err)
	}

	// Check digest file existence
	digestFilePath := digestFilePath(filePath)
	digestFileInfo, err := os.Stat(digestFilePath)
	isNotExistErr := os.IsNotExist(err)
	if err != nil && !isNotExistErr {
		return fmt.Errorf("error checking digest file: %w", err)
	}

	// If the digest file does not exist or the digest file is older than the file, compute and verify the digest manually
	if isNotExistErr || !digestFileInfo.ModTime().After(fileInfo.ModTime()) {
		log.G(ctx).Warnf("Digest file for %s does not exist or is outdated, computing digest manually", filePath)
		return ComputeAndVerifyFileDigest(filePath, expectedDigest)
	}

	storedDigest, err := os.ReadFile(digestFilePath)
	if err != nil {
		return fmt.Errorf("error reading digest file: %w", err)
	}

	if expectedDigest.String() != string(storedDigest) {
		return fmt.Errorf("digest does not match: got %s, expected %s", string(storedDigest), expectedDigest)
	}

	return nil
}

// ComputeAndVerifyFileDigest computes the digest of the file at the given path and verifies it against the expected digest.
func ComputeAndVerifyFileDigest(filePath string, expectedDigest digest.Digest) error {
	rd, err := os.Open(filePath)
	if err != nil {
		return err
	}

	verifier := expectedDigest.Verifier()
	if _, err := io.Copy(verifier, rd); err != nil {
		return err
	}

	if !verifier.Verified() {
		return errors.New("digest verification failed")
	}

	// Write the digest to the digest file for future validation
	return writeDigestFile(filePath, expectedDigest)
}
