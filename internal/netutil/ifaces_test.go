package netutil_test

import (
	"testing"

	"github.com/agoda-com/macOS-vz-kubelet/internal/netutil"

	psnet "github.com/shirou/gopsutil/v4/net"
	"github.com/stretchr/testify/assert"
	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
)

func TestGetActiveInterface(t *testing.T) {
	// Define the test cases
	tests := []struct {
		name          string
		interfaces    psnet.InterfaceStatList
		expectedIP    string
		expectedError error
	}{
		{
			name: "Ethernet with valid IP",
			interfaces: psnet.InterfaceStatList{
				{
					Name: "en0",
					Addrs: []psnet.InterfaceAddr{
						{Addr: "192.168.1.10/24"},
					},
				},
			},
			expectedIP:    "192.168.1.10",
			expectedError: nil,
		},
		{
			name: "Non-Ethernet with valid IP",
			interfaces: psnet.InterfaceStatList{
				{
					Name: "wlan0",
					Addrs: []psnet.InterfaceAddr{
						{Addr: "10.0.0.5/24"},
					},
				},
			},
			expectedIP:    "10.0.0.5",
			expectedError: nil,
		},
		{
			name: "No valid IP",
			interfaces: psnet.InterfaceStatList{
				{
					Name: "en0",
					Addrs: []psnet.InterfaceAddr{
						{Addr: "fe80::1/64"}, // IPv6 address
					},
				},
				{
					Name: "lo0",
					Addrs: []psnet.InterfaceAddr{
						{Addr: "127.0.0.1/8"}, // Loopback address
					},
				},
			},
			expectedIP:    "",
			expectedError: errdefs.NotFound("no valid IP address found"),
		},
		{
			name: "Loopback and IPv6",
			interfaces: psnet.InterfaceStatList{
				{
					Name: "lo0",
					Addrs: []psnet.InterfaceAddr{
						{Addr: "127.0.0.1/8"}, // Loopback address
					},
				},
				{
					Name: "en0",
					Addrs: []psnet.InterfaceAddr{
						{Addr: "fe80::1/64"}, // IPv6 address
					},
				},
			},
			expectedIP:    "",
			expectedError: errdefs.NotFound("no valid IP address found"),
		},
	}

	// Loop over the test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function
			ip, err := netutil.GetActiveInterface(tc.interfaces)

			// Check the expected IP address and error
			assert.Equal(t, tc.expectedIP, ip)
			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
