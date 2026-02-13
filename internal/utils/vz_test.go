package utils_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/agoda-com/macOS-vz-kubelet/internal/utils"
)

// TestGetContainerID tests the GetContainerID function
func TestGetContainerID(t *testing.T) {
	tests := []struct {
		protocol       string
		containerName  string
		expectedPrefix string // We can't know the exact hash, but we can check the prefix
	}{
		{"http", "myContainer", "http://"},
		{"https", "anotherContainer", "https://"},
		{"ftp", "ftpContainer", "ftp://"},
		{"http", "container_with_special_chars!@#", "http://"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.protocol, tt.containerName), func(t *testing.T) {
			got := utils.GetContainerID(tt.protocol, tt.containerName)
			if !strings.HasPrefix(got, tt.expectedPrefix) {
				t.Errorf("GetContainerID(%q, %q) = %q; want prefix %q", tt.protocol, tt.containerName, got, tt.expectedPrefix)
			}
			// Further validate the length of the resulting hash
			expectedLength := len(tt.expectedPrefix) + 64 // protocol:// + sha256 hash (64 hex characters)
			if len(got) != expectedLength {
				t.Errorf("GetContainerID(%q, %q) = %q; want length %d", tt.protocol, tt.containerName, got, expectedLength)
			}
		})
	}
}
