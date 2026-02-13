package container_test

import (
	"testing"

	"github.com/agoda-com/macOS-vz-kubelet/internal/data/container"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
)

func TestGetContainerInfo(t *testing.T) {
	data := container.ContainerData{}
	info := container.ContainerInfo{ID: "123", Error: nil}
	data.SetContainerInfo("default", "pod1", "container1", info)

	result, found := data.GetContainerInfo("default", "pod1", "container1")
	assert.True(t, found)
	assert.Equal(t, info, result)

	result, found = data.GetContainerInfo("default", "pod1", "container2")
	assert.False(t, found)
	assert.Equal(t, container.ContainerInfo{}, result)

	result, found = data.GetContainerInfo("default", "pod2", "container1")
	assert.False(t, found)
	assert.Equal(t, container.ContainerInfo{}, result)
}

func TestGetOrCreateContainerInfo(t *testing.T) {
	data := container.ContainerData{}
	info := container.ContainerInfo{ID: "123", Error: nil}

	result, loaded := data.GetOrCreateContainerInfo("default", "pod1", "container1", info)
	assert.False(t, loaded)
	assert.Equal(t, info, result)

	// Test retrieving the same container info
	result, loaded = data.GetOrCreateContainerInfo("default", "pod1", "container1", info)
	assert.True(t, loaded)
	assert.Equal(t, info, result)
}

func TestSetContainerInfo(t *testing.T) {
	data := container.ContainerData{}
	info := container.ContainerInfo{ID: "123", Error: nil}

	data.SetContainerInfo("default", "pod1", "container1", info)

	result, found := data.GetContainerInfo("default", "pod1", "container1")
	assert.True(t, found)
	assert.Equal(t, info, result)
}

func TestRemoveAllContainerInfo(t *testing.T) {
	data := container.ContainerData{}
	info1 := container.ContainerInfo{ID: "123", Error: nil}
	info2 := container.ContainerInfo{ID: "456", Error: nil}

	data.SetContainerInfo("default", "pod1", "container1", info1)
	data.SetContainerInfo("default", "pod1", "container2", info2)

	containerInfoMap, found := data.RemoveAllContainerInfo("default", "pod1")
	assert.True(t, found)
	assert.Len(t, containerInfoMap, 2)
	assert.Equal(t, info1, containerInfoMap["container1"])
	assert.Equal(t, info2, containerInfoMap["container2"])

	// Ensure data is removed
	containerInfoMap, found = data.RemoveAllContainerInfo("default", "pod1")
	assert.False(t, found)
	assert.Nil(t, containerInfoMap)
}

func TestGetAllContainerInfo(t *testing.T) {
	data := container.ContainerData{}
	info1 := container.ContainerInfo{ID: "123", Error: nil}
	info2 := container.ContainerInfo{ID: "456", Error: nil}

	data.SetContainerInfo("default", "pod1", "container1", info1)
	data.SetContainerInfo("default", "pod1", "container2", info2)

	containerInfoMap, found := data.GetAllContainerInfo("default", "pod1")
	assert.True(t, found)
	assert.Len(t, containerInfoMap, 2)
	assert.Equal(t, info1, containerInfoMap["container1"])
	assert.Equal(t, info2, containerInfoMap["container2"])

	containerInfoMap, found = data.GetAllContainerInfo("default", "pod2")
	assert.False(t, found)
	assert.Nil(t, containerInfoMap)
}

func TestGetAllData(t *testing.T) {
	data := container.ContainerData{}
	info1 := container.ContainerInfo{ID: "123", Error: nil}
	info2 := container.ContainerInfo{ID: "456", Error: nil}
	info3 := container.ContainerInfo{ID: "789", Error: nil}

	data.SetContainerInfo("default", "pod1", "container1", info1)
	data.SetContainerInfo("default", "pod1", "container2", info2)
	data.SetContainerInfo("default", "pod2", "container3", info3)

	allData := data.GetAllData()
	assert.Len(t, allData, 2)
	assert.Len(t, allData[types.NamespacedName{Namespace: "default", Name: "pod1"}], 2)
	assert.Len(t, allData[types.NamespacedName{Namespace: "default", Name: "pod2"}], 1)

	assert.Equal(t, info1, allData[types.NamespacedName{Namespace: "default", Name: "pod1"}]["container1"])
	assert.Equal(t, info2, allData[types.NamespacedName{Namespace: "default", Name: "pod1"}]["container2"])
	assert.Equal(t, info3, allData[types.NamespacedName{Namespace: "default", Name: "pod2"}]["container3"])
}
