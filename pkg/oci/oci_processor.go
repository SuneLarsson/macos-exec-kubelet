package oci

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/agoda-com/macOS-vz-kubelet/internal/disk"

	contentpkg "oras.land/oras-go/v2/content"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/virtual-kubelet/virtual-kubelet/log"
	"github.com/virtual-kubelet/virtual-kubelet/trace"
)

// processContentByType processes content based on whether it is compressed or regular.
func (s *Store) processContentByType(ctx context.Context, expected ocispec.Descriptor, content io.Reader, outputFilePath string) (err error) {
	ctx, span := trace.StartSpan(ctx, "OCI.processContentByType")
	ctx = span.WithFields(ctx, log.Fields{
		"expected.MediaType": expected.MediaType,
		"expected.Digest":    expected.Digest,
		"outputFilePath":     outputFilePath,
	})
	defer func() {
		span.SetStatus(err)
		span.End()
	}()

	if err = os.MkdirAll(s.workingDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to ensure the working directory exists: %w", err)
	}

	uncompressedSize, sizeExists := expected.Annotations[AnnotationUncompressedSize]
	uncompressedDigest, digestExists := expected.Annotations[AnnotationUncompressedDigest]
	if sizeExists && digestExists {
		return s.processCompressedContent(ctx, expected, content, outputFilePath, uncompressedSize, uncompressedDigest)
	}

	return s.processRegularContent(ctx, expected, content, outputFilePath)
}

// processCompressedContent handles content that is compressed, attempting to decompress it.
func (s *Store) processCompressedContent(ctx context.Context, expected ocispec.Descriptor, content io.Reader, outputFilePath, uncompressedSize, uncompressedDigest string) (err error) {
	ctx, span := trace.StartSpan(ctx, "OCI.processCompressedContent")
	ctx = span.WithFields(ctx, log.Fields{
		"uncompressedDigest": uncompressedDigest,
		"uncompressedSize":   uncompressedSize,
	})
	defer func() {
		span.SetStatus(err)
		span.End()
	}()

	size, err := strconv.ParseInt(uncompressedSize, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid uncompressed size: %w", err)
	}

	fp, err := s.tempFile()
	if err != nil {
		return err
	}
	ctx = span.WithField(ctx, "tmp", fp.Name())
	defer func() {
		err = errors.Join(err, fp.Close())
	}()

	path := fp.Name()

	// save the content to the temp file
	if err = s.saveFile(ctx, fp, expected, content); err != nil {
		return fmt.Errorf("failed to save content to temp file: %w", err)
	}

	// Since file was saved successfully, store the digest and path
	s.digestToPath.Store(expected.Digest, path)

	d, err := disk.DecompressFileWithPath(ctx, path, outputFilePath, size)
	if err != nil {
		return fmt.Errorf("failed to decompress file: %w", err)
	}

	if d != digest.Digest(uncompressedDigest) {
		return fmt.Errorf("digest mismatch: expected %s, got %s", uncompressedDigest, d)
	}
	s.mediaTypeToPath.Store(expected.MediaType, outputFilePath)

	return nil
}

// processRegularContent handles content that is not compressed, saving it directly.
func (s *Store) processRegularContent(ctx context.Context, expected ocispec.Descriptor, content io.Reader, outputFilePath string) (err error) {
	ctx, span := trace.StartSpan(ctx, "OCI.processRegularContent")
	defer func() {
		span.SetStatus(err)
		span.End()
	}()

	fp, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		err = errors.Join(err, fp.Close())
	}()

	if err = s.saveFile(ctx, fp, expected, content); err != nil {
		return fmt.Errorf("failed to save content: %w", err)
	}

	// Since file was saved successfully, store the digest and path
	s.digestToPath.Store(expected.Digest, outputFilePath)

	if err = disk.ComputeAndVerifyFileDigest(outputFilePath, expected.Digest); err != nil {
		return fmt.Errorf("failed to verify file digest: %w", err)
	}

	s.mediaTypeToPath.Store(expected.MediaType, outputFilePath)

	return nil
}

// saveFile saves content matching an ocispec.Descriptor to a given file, performing verification.
func (s *Store) saveFile(ctx context.Context, fp *os.File, expected ocispec.Descriptor, content io.Reader) (err error) {
	ctx, span := trace.StartSpan(ctx, "OCI.saveFile")
	defer func() {
		span.SetStatus(err)
		span.End()
	}()

	path := fp.Name()
	_ = span.WithField(ctx, "path", path)

	// verify while copying
	vr := contentpkg.NewVerifyReader(content, expected)

	// copy content to the file
	if _, err = io.Copy(fp, vr); err != nil {
		return fmt.Errorf("failed to copy content to %s: %w", path, err)
	}

	// verify the content
	if err = vr.Verify(); err != nil {
		return fmt.Errorf("failed to verify content in %s: %w", path, err)
	}

	// sync file
	if err = fp.Sync(); err != nil {
		return fmt.Errorf("failed to sync %s: %w", path, err)
	}

	return nil
}
