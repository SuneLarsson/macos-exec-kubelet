package container

import (
	"sync"

	"k8s.io/apimachinery/pkg/types"
)

// ContainerData is a thread-safe map to store container information, indexed by NamespacedName and container name.
type ContainerData struct {
	data sync.Map // map[types.NamespacedName]map[string]ContainerInfo (podNamespace/podName -> containerName -> ContainerInfo)
}

// GetContainerInfo retrieves the ContainerInfo for a specific container within a specific pod.
// It returns the ContainerInfo and a boolean indicating whether the container information was found.
func (d *ContainerData) GetContainerInfo(podNamespace, podName, containerName string) (ContainerInfo, bool) {
	key := types.NamespacedName{Namespace: podNamespace, Name: podName}
	val, ok := d.data.Load(key)
	if !ok {
		return ContainerInfo{}, false
	}
	infoval, ok := val.(*sync.Map).Load(containerName)
	if !ok {
		return ContainerInfo{}, false
	}
	return *infoval.(*ContainerInfo), true
}

// GetOrCreateContainerInfo retrieves the ContainerInfo for a specific container within a specific pod,
// or creates and stores the provided ContainerInfo if it doesn't already exist.
// It returns the ContainerInfo and a boolean indicating whether the container information was already present.
func (d *ContainerData) GetOrCreateContainerInfo(podNamespace, podName, containerName string, info ContainerInfo) (ContainerInfo, bool) {
	key := types.NamespacedName{Namespace: podNamespace, Name: podName}
	val, _ := d.data.LoadOrStore(key, &sync.Map{})
	infoval, loaded := val.(*sync.Map).LoadOrStore(containerName, &info)
	return *infoval.(*ContainerInfo), loaded
}

// SetContainerInfo sets the ContainerInfo for a specific container within a specific pod.
func (d *ContainerData) SetContainerInfo(podNamespace, podName, containerName string, info ContainerInfo) {
	key := types.NamespacedName{Namespace: podNamespace, Name: podName}
	val, _ := d.data.LoadOrStore(key, &sync.Map{})
	val.(*sync.Map).Store(containerName, &info)
}

// RemoveAllContainerInfo removes all container information for a specific pod.
// It returns a map of container names to ContainerInfo and a boolean indicating whether the pod information was found.
func (d *ContainerData) RemoveAllContainerInfo(podNamespace, podName string) (map[string]ContainerInfo, bool) {
	key := types.NamespacedName{Namespace: podNamespace, Name: podName}
	val, ok := d.data.LoadAndDelete(key)
	if !ok {
		return nil, false
	}
	containerInfoMap := make(map[string]ContainerInfo)
	val.(*sync.Map).Range(func(key, value interface{}) bool {
		containerInfoMap[key.(string)] = *value.(*ContainerInfo)
		return true
	})
	return containerInfoMap, true
}

// GetAllContainerInfo retrieves all container information for a specific pod.
// It returns a map of container names to ContainerInfo and a boolean indicating whether the pod information was found.
func (d *ContainerData) GetAllContainerInfo(podNamespace, podName string) (map[string]ContainerInfo, bool) {
	key := types.NamespacedName{Namespace: podNamespace, Name: podName}
	val, ok := d.data.Load(key)
	if !ok {
		return nil, false
	}
	containerInfoMap := make(map[string]ContainerInfo)
	val.(*sync.Map).Range(func(key, value interface{}) bool {
		containerInfoMap[key.(string)] = *value.(*ContainerInfo)
		return true
	})
	return containerInfoMap, true
}

// GetAllData retrieves all container information for all pods.
// It returns a map of NamespacedName to a map of container names to ContainerInfo.
func (d *ContainerData) GetAllData() map[types.NamespacedName]map[string]ContainerInfo {
	containerData := make(map[types.NamespacedName]map[string]ContainerInfo)
	d.data.Range(func(key, value interface{}) bool {
		containerInfoMap := make(map[string]ContainerInfo)
		value.(*sync.Map).Range(func(key, value interface{}) bool {
			containerInfoMap[key.(string)] = *value.(*ContainerInfo)
			return true
		})
		containerData[key.(types.NamespacedName)] = containerInfoMap
		return true
	})
	return containerData
}
