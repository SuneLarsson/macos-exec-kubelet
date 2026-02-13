package netutil

import (
	"net"
	"strings"

	psnet "github.com/shirou/gopsutil/v4/net"

	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
)

// GetActiveInterface returns the IP address of the active network interface.
func GetActiveInterface(ifs psnet.InterfaceStatList) (string, error) {
	// First pass: Prefer Ethernet interfaces
	for _, i := range ifs {
		if isEthernet(i.Name) {
			if ip, ok := findValidIP(i); ok {
				return ip, nil
			}
		}
	}

	// Second pass: Check other interfaces if no suitable Ethernet interface is found
	for _, i := range ifs {
		if !isEthernet(i.Name) {
			if ip, ok := findValidIP(i); ok {
				return ip, nil
			}
		}
	}

	return "", errdefs.NotFound("no valid IP address found")
}

// isEthernet returns true if the interface name starts with "en"
func isEthernet(name string) bool {
	return strings.HasPrefix(name, "en")
}

// findValidIP returns the first valid IPv4 address found in the interface's addresses
func findValidIP(i psnet.InterfaceStat) (string, bool) {
	for _, a := range i.Addrs {
		// Parse the IP address from the address string
		ip, _, err := net.ParseCIDR(a.Addr)
		if err != nil {
			continue
		}
		// Check if the IP is a valid IPv4 address and not a loopback address
		if ip.To4() != nil && !ip.IsLoopback() {
			return ip.String(), true
		}
	}
	return "", false
}
