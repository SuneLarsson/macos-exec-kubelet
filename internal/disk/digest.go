package disk

import (
	"errors"
	"os"

	"github.com/opencontainers/go-digest"
)

const (
	// DigestFileSuffix is the suffix for the digest file.
	DigestFileSuffix = ".digest"
)

// writeDigestFile writes the digest to a file.
func writeDigestFile(filePath string, d digest.Digest) error {
	digestFilePath := digestFilePath(filePath)
	digestFile, err := os.Create(digestFilePath)
	if err != nil {
		return err
	}

	if _, err := digestFile.WriteString(d.String()); err != nil {
		return errors.Join(err, digestFile.Close())
	}

	return digestFile.Close()
}

// digestFilePath returns the path to the digest file.
func digestFilePath(filePath string) string {
	return filePath + DigestFileSuffix
}
