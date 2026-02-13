package oci

import (
	"k8s.io/apimachinery/pkg/util/sets"
)

// MediaType represents a media type.
type MediaType string

const (
	// MediaTypeDiskImage specifies the media type for a disk image.
	MediaTypeDiskImage MediaType = "application/vnd.agoda.macosvz.disk.image.v1"

	// MediaTypeAuxImage specifies the media type for an auxiliary (nvram) image.
	MediaTypeAuxImage MediaType = "application/vnd.agoda.macosvz.aux.image.v1"

	// MediaTypeConfigV1 specifies the media type for a configuration.
	// Internal use only.
	MediaTypeConfigV1 MediaType = "application/vnd.agoda.macosvz.config.v1+json"
)

// mediaTypeToTitle maps media types to their titles.
var mediaTypeToTitle = map[MediaType]string{
	MediaTypeConfigV1:  "config.json",
	MediaTypeDiskImage: "disk.img",
	MediaTypeAuxImage:  "aux.img",
}

// Title returns the title of the media type.
func (mt MediaType) Title() string {
	return mediaTypeToTitle[mt]
}

// supportedMediaTypes contains the supported media types.
var supportedMediaTypes = sets.NewString(
	string(MediaTypeConfigV1),
	string(MediaTypeDiskImage),
	string(MediaTypeAuxImage),
)

// IsMediaTypeSupported checks if the media type is supported.
func IsMediaTypeSupported(mediaType string) bool {
	return supportedMediaTypes.Has(mediaType)
}
