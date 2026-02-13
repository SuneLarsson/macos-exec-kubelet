package vm

import (
	"context"

	"github.com/agoda-com/macOS-vz-kubelet/pkg/resource"
)

// VirtualMachineInfo stores the information about macOS virtual machine
type VirtualMachineInfo struct {
	Ref                string
	Resource           resource.MacOSVirtualMachine
	DownloadCancelFunc context.CancelFunc
}
