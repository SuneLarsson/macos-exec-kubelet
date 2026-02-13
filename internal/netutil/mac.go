package netutil

import (
	"crypto/rand"
	"fmt"
	"net"
	"strings"
)

// GenerateRandMAC generates a random MAC address
func GenerateRandMAC() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("unable to retrieve 6 rnd bytes: %w", err)
	}

	// Set locally administered addresses bit and reset multicast bit
	buf[0] = (buf[0] | 0x02) & 0xfe

	return net.HardwareAddr(buf).String(), nil
}

// NormalizeMACAddress normalizes MAC addresses by ensuring all hex digits are in lowercase
// and leading zeros in each octet are removed.
func NormalizeMACAddress(mac string) string {
	parts := strings.Split(mac, ":")
	for i, part := range parts {
		if len(part) == 2 && part[0] == '0' {
			parts[i] = part[1:]
		}
	}
	return strings.Join(parts, ":")
}
