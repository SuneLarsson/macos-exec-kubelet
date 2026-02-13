package oci

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/agoda-com/macOS-vz-kubelet/internal/disk"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// descriptorFromStorageFile creates an ocispec.Descriptor for a given storage item by compressing it and storing it in a temp file.
// Returns the descriptor which includes media type, digest, size, and annotations.
func (s *Store) descriptorFromStorageFile(ctx context.Context, mediaType MediaType, path string) (desc ocispec.Descriptor, err error) {
	tempFile, err := s.tempFile()
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	defer func() {
		err = errors.Join(err, tempFile.Close())
	}()

	res, err := disk.CompressFileWithPath(ctx, path, tempFile)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	// Save the path since file was prepared successfully
	s.digestToPath.Store(res.GzDigest, res.OutputFilePath)

	return ocispec.Descriptor{
		MediaType: string(mediaType),
		Digest:    res.GzDigest,       // digest of the compressed content
		Size:      res.CompressedSize, // size of the compressed content
		Annotations: map[string]string{
			ocispec.AnnotationTitle:      mediaType.Title(),
			AnnotationUncompressedSize:   fmt.Sprintf("%d", res.UncompressedSize), // size of the uncompressed content
			AnnotationUncompressedDigest: res.UncompressedDigest.String(),         // digest of the uncompressed content
		},
	}, err
}

// descriptorFromConfig generates a descriptor for the config itself, including the serialization of the config into a file.
// Returns the descriptor representing the config configuration.
func (s *Store) descriptorFromConfig(_ context.Context, cfg Config) (ocispec.Descriptor, error) {
	// serialize config
	cfgData, err := json.Marshal(&cfg)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	// save as json file
	configFile, err := s.tempFile()
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	defer func() {
		err = errors.Join(err, configFile.Close())
	}()

	digester := digest.Canonical.Digester()
	w := io.MultiWriter(configFile, digester.Hash())
	if _, err := w.Write(cfgData); err != nil {
		return ocispec.Descriptor{}, err
	}

	// flush all
	if err := configFile.Sync(); err != nil {
		return ocispec.Descriptor{}, err
	}

	// get file info
	fi, err := configFile.Stat()
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	configDigest := digester.Digest()
	s.digestToPath.Store(configDigest, configFile.Name())

	return ocispec.Descriptor{
		MediaType: string(cfg.MediaType),
		Digest:    configDigest,
		Size:      fi.Size(),
		Annotations: map[string]string{
			ocispec.AnnotationTitle: cfg.MediaType.Title(),
		},
	}, nil
}
