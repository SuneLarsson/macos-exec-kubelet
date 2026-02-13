package netutil_test

import (
	"net"
	"testing"

	"github.com/agoda-com/macOS-vz-kubelet/internal/netutil"

	"github.com/stretchr/testify/assert"
)

// TestGenerateRandMAC tests the GenerateRandMAC function
func TestGenerateRandMAC(t *testing.T) {
	for range 10 { // Generate multiple MACs to test randomness and format
		mac, err := netutil.GenerateRandMAC()
		if err != nil {
			t.Fatalf("GenerateRandMAC() error: %v", err)
		}

		hwAddr, err := net.ParseMAC(mac)
		assert.NoError(t, err)
		assert.Equal(t, hwAddr[0]&0x01, uint8(0))

		if hwAddr[0]&0x02 == 0 || hwAddr[0]&0x01 != 0 {
			t.Errorf("GenerateRandMAC() generated MAC does not have locally administered and unicast format: %v", mac)
		}
	}
}

// TestNormalizeMACAddress tests the NormalizeMACAddress function
func TestNormalizeMACAddress(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"00:1A:2B:3C:4D:5E", "0:1A:2B:3C:4D:5E"},
		{"01:23:45:67:89:AB", "1:23:45:67:89:AB"},
		{"0A:BC:DE:F0:12:34", "A:BC:DE:F0:12:34"},
		{"00:00:00:00:00:00", "0:0:0:0:0:0"},
		{"0:1a:2b:3c:4d:5e", "0:1a:2b:3c:4d:5e"}, // Already normalized
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := netutil.NormalizeMACAddress(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
