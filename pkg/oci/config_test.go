package oci_test

import (
	"encoding/json"
	"testing"

	"github.com/agoda-com/macOS-vz-kubelet/pkg/oci"

	"github.com/stretchr/testify/assert"
)

func TestConfig_MarshalJSON(t *testing.T) {
	// Sample input data for Config
	hardwareModelData := "YnBsaXN0MDDTAQIDBAUGXxAZRGF0YVJlcHJlc2VudGF0aW9uVmVyc2lvbl8QD1BsYXRmb3JtVmVyc2lvbl8QEk1pbmltdW1TdXBwb3J0ZWRPUxQAAAAAAAAAAAAAAAAAAAABEAKjBwgIEA0QAAgPKz1SY2VpawAAAAAAAAEBAAAAAAAAAAkAAAAAAAAAAAAAAAAAAABt"
	machineIdData := "YnBsaXN0MDDRAQJURUNJRBMWYaHTD7Jo1QgLEAAAAAAAAAEBAAAAAAAAAAMAAAAAAAAAAAAAAAAAAAAZ"

	config := oci.NewMacOSConfig(hardwareModelData, machineIdData)

	// Perform JSON marshaling
	jsonData, err := json.Marshal(&config)

	// Check for any marshaling errors
	assert.NoError(t, err)

	// Expected JSON structure
	expectedJSON := `{
		"mediatype": "application/vnd.agoda.macosvz.config.v1+json",
		"os": "darwin",
		"hardwareModelData": "` + hardwareModelData + `",
		"machineIdData": "` + machineIdData + `",
		"storage": [
			{
				"mediatype": "application/vnd.agoda.macosvz.aux.image.v1",
				"file": "aux.img"
			},
			{
				"mediatype": "application/vnd.agoda.macosvz.disk.image.v1",
				"file": "disk.img"
			}
		]
	}`

	// Assert that the marshaled JSON matches the expected output
	assert.JSONEq(t, expectedJSON, string(jsonData))
}

func TestConfig_UnmarshalJSON(t *testing.T) {
	// Input JSON that mimics the expected format
	inputJSON := `{
		"mediatype": "application/vnd.agoda.macosvz.config.v1+json",
		"os": "darwin",
		"hardwareModelData": "YnBsaXN0MDDTAQIDBAUGXxAZRGF0YVJlcHJlc2VudGF0aW9uVmVyc2lvbl8QD1BsYXRmb3JtVmVyc2lvbl8QEk1pbmltdW1TdXBwb3J0ZWRPUxQAAAAAAAAAAAAAAAAAAAABEAKjBwgIEA0QAAgPKz1SY2VpawAAAAAAAAEBAAAAAAAAAAkAAAAAAAAAAAAAAAAAAABt",
		"machineIdData": "YnBsaXN0MDDRAQJURUNJRBMWYaHTD7Jo1QgLEAAAAAAAAAEBAAAAAAAAAAMAAAAAAAAAAAAAAAAAAAAZ",
		"storage": [
			{
				"mediatype": "application/vnd.agoda.macosvz.aux.image.v1",
				"file": "aux.img"
			},
			{
				"mediatype": "application/vnd.agoda.macosvz.disk.image.v1",
				"file": "disk.img"
			}
		]
	}`

	// Create a Config object and unmarshal the JSON into it
	var config oci.Config
	err := json.Unmarshal([]byte(inputJSON), &config)

	// Assert that unmarshaling does not produce an error
	assert.NoError(t, err)

	// Assert individual fields
	assert.Equal(t, "application/vnd.agoda.macosvz.config.v1+json", string(config.MediaType))
	assert.Equal(t, "darwin", config.OS)
	assert.Equal(t, "YnBsaXN0MDDTAQIDBAUGXxAZRGF0YVJlcHJlc2VudGF0aW9uVmVyc2lvbl8QD1BsYXRmb3JtVmVyc2lvbl8QEk1pbmltdW1TdXBwb3J0ZWRPUxQAAAAAAAAAAAAAAAAAAAABEAKjBwgIEA0QAAgPKz1SY2VpawAAAAAAAAEBAAAAAAAAAAkAAAAAAAAAAAAAAAAAAABt", config.HardwareModelData)
	assert.Equal(t, "YnBsaXN0MDDRAQJURUNJRBMWYaHTD7Jo1QgLEAAAAAAAAAEBAAAAAAAAAAMAAAAAAAAAAAAAAAAAAAAZ", config.MachineIdData)

	// Assert storage media types
	expectedStorage := []oci.MediaType{
		oci.MediaTypeAuxImage,
		oci.MediaTypeDiskImage,
	}
	assert.Equal(t, expectedStorage, config.Storage)
}
