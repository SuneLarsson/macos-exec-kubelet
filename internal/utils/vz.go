package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// GetContainerID returns a container ID based on the container name
func GetContainerID(protocol, containerName string) string {
	containerResourceID := fmt.Sprintf("containers/%s", containerName)

	h := sha256.New()
	if _, err := h.Write([]byte(strings.ToUpper(containerResourceID))); err != nil {
		return ""
	}
	hashBytes := h.Sum(nil)
	return fmt.Sprintf("%s://%s", protocol, hex.EncodeToString(hashBytes))
}
