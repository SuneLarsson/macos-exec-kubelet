package utils

import (
	"regexp"
	"strings"
)

// SanitizeAppleCPUModelForK8sLabel simplifies the Apple CPU model name to a valid k8s label value
func SanitizeAppleCPUModelForK8sLabel(name string) string {
	// Remove the "Apple" prefix
	model := strings.ReplaceAll(name, "Apple ", "")

	// Define a regex pattern to replace invalid characters (anything not alphanumeric, '_', '-', or '.')
	// and collapse multiple underscores into one.
	invalidCharRegex := regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)

	// Replace invalid characters with underscores and collapse consecutive underscores
	model = invalidCharRegex.ReplaceAllString(model, "_")

	// Ensure the model doesn't start or end with non-alphanumeric characters
	model = strings.Trim(model, "_-.")

	return model
}
