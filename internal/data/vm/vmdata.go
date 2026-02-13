package vm

import (
	"sync"
	"sync/atomic"

	"k8s.io/apimachinery/pkg/types"
)

// VirtualMachineData stores the information about macOS virtual machines
type VirtualMachineData struct {
	data    sync.Map // map[types.NamespacedName]VirtualMachineInfo (podNamespace/podName -> VirtualMachineInfo)
	counter int32    // number of virtual machines stored
}

// GetVirtualMachineInfo retrieves the VirtualMachineInfo for a specific pod.
// It returns the VirtualMachineInfo and a boolean indicating whether the virtual machine information was found.
func (d *VirtualMachineData) GetVirtualMachineInfo(podNamespace, podName string) (VirtualMachineInfo, bool) {
	key := types.NamespacedName{Namespace: podNamespace, Name: podName}
	val, ok := d.data.Load(key)
	if !ok {
		return VirtualMachineInfo{}, false
	}
	return *val.(*VirtualMachineInfo), true
}

// UpdateVirtualMachineInfo updates the VirtualMachineInfo for a specific pod.
// It returns the VirtualMachineInfo and a boolean indicating whether the virtual machine information was found.
func (d *VirtualMachineData) UpdateVirtualMachineInfo(podNamespace, podName string, updateFunc func(VirtualMachineInfo) VirtualMachineInfo) (VirtualMachineInfo, bool) {
	key := types.NamespacedName{Namespace: podNamespace, Name: podName}
	val, ok := d.data.Load(key)
	if !ok {
		return VirtualMachineInfo{}, false
	}
	newVal := updateFunc(*val.(*VirtualMachineInfo))
	d.data.Store(key, &newVal)
	return newVal, true
}

// GetOrCreateVirtualMachineInfo retrieves the VirtualMachineInfo for a specific pod,
// or creates and stores the provided VirtualMachineInfo if it doesn't already exist.
// It returns the VirtualMachineInfo and a boolean indicating whether the virtual machine information was already present.
func (d *VirtualMachineData) GetOrCreateVirtualMachineInfo(podNamespace, podName string, info VirtualMachineInfo) (VirtualMachineInfo, bool) {
	key := types.NamespacedName{Namespace: podNamespace, Name: podName}
	val, loaded := d.data.LoadOrStore(key, &info)
	if !loaded {
		d.incrementCounter()
	}
	return *val.(*VirtualMachineInfo), loaded
}

// RemoveVirtualMachineInfo removes the VirtualMachineInfo for a specific pod.
func (d *VirtualMachineData) RemoveVirtualMachineInfo(podNamespace, podName string) {
	key := types.NamespacedName{Namespace: podNamespace, Name: podName}
	_, loaded := d.data.LoadAndDelete(key)
	if loaded {
		d.decrementCounter()
	}
}

// ListVirtualMachines returns a map of all virtual machines stored.
func (d *VirtualMachineData) ListVirtualMachines() map[types.NamespacedName]VirtualMachineInfo {
	vmMap := make(map[types.NamespacedName]VirtualMachineInfo)
	d.data.Range(func(key, value interface{}) bool {
		vmMap[key.(types.NamespacedName)] = *value.(*VirtualMachineInfo)
		return true
	})
	return vmMap
}

// Count returns the number of virtual machines stored.
// It is safe to call concurrently.
func (d *VirtualMachineData) Count() int32 {
	return atomic.LoadInt32(&d.counter)
}

// incrementCounter increments the number of virtual machines stored.
func (d *VirtualMachineData) incrementCounter() {
	atomic.AddInt32(&d.counter, 1)
}

// decrementCounter decrements the number of virtual machines stored.
func (d *VirtualMachineData) decrementCounter() {
	atomic.AddInt32(&d.counter, -1)
}
