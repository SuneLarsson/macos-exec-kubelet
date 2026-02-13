package oci

import "encoding/json"

// Config represents an OCI bundle.
type Config struct {
	MediaType         MediaType   `json:"mediatype,omitempty"`
	OS                string      `json:"os"`
	HardwareModelData string      `json:"hardwareModelData"`
	MachineIdData     string      `json:"machineIdData"`
	Storage           []MediaType `json:"-"`
}

// NewMacOSConfig creates a new Bundle tailored for macOS OCI images.
func NewMacOSConfig(hardwareModelData, machineIdData string) Config {
	return Config{
		MediaType:         MediaTypeConfigV1,
		OS:                "darwin",
		HardwareModelData: hardwareModelData,
		MachineIdData:     machineIdData,
		Storage: []MediaType{
			MediaTypeAuxImage,
			MediaTypeDiskImage,
		},
	}
}

// StorageItem represents the structure for storage when marshaled.
type storageItem struct {
	MediaType MediaType `json:"mediatype"`
	File      string    `json:"file"`
}

// MarshalJSON customizes the JSON encoding for Config.
func (c *Config) MarshalJSON() ([]byte, error) {
	type Alias Config
	aux := &struct {
		Storage []storageItem `json:"storage"`
		*Alias
	}{
		Alias:   (*Alias)(c),
		Storage: make([]storageItem, 0, len(c.Storage)),
	}

	// Manually handle the storage field
	for _, mediaType := range c.Storage {
		item := storageItem{
			MediaType: mediaType,
			File:      mediaType.Title(),
		}
		aux.Storage = append(aux.Storage, item)
	}

	// Marshal the custom structure
	return json.Marshal(aux)
}

// UnmarshalJSON customizes the JSON decoding for Config.
func (c *Config) UnmarshalJSON(data []byte) error {
	type Alias Config // Create an alias to avoid recursion
	aux := &struct {
		Storage []storageItem `json:"storage"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Convert StorageItems back to MediaType
	for _, item := range aux.Storage {
		c.Storage = append(c.Storage, item.MediaType)
	}

	return nil
}
