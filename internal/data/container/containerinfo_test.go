package container_test

import (
	"errors"
	"testing"

	"github.com/agoda-com/macOS-vz-kubelet/internal/data/container"
	"github.com/stretchr/testify/assert"
)

func TestContainerInfo_WithID(t *testing.T) {
	// Arrange
	initialID := "initial-id"
	newID := "new-id"
	containerInfo := container.ContainerInfo{ID: initialID}

	// Act
	updatedContainerInfo := containerInfo.WithID(newID)

	// Assert
	assert.Equal(t, newID, updatedContainerInfo.ID, "The ID should be updated to the new value")
	assert.Equal(t, initialID, containerInfo.ID, "The original ContainerInfo should remain unchanged")
}

func TestContainerInfo_WithError(t *testing.T) {
	// Arrange
	initialError := errors.New("initial error")
	newError := errors.New("new error")
	containerInfo := container.ContainerInfo{Error: initialError}

	// Act
	updatedContainerInfo := containerInfo.WithError(newError)

	// Assert
	assert.Equal(t, newError, updatedContainerInfo.Error, "The Error should be updated to the new value")
	assert.Equal(t, initialError, containerInfo.Error, "The original ContainerInfo should remain unchanged")
}

func TestContainerInfo_WithIDAndError(t *testing.T) {
	// Arrange
	initialID := "initial-id"
	newID := "new-id"
	initialError := errors.New("initial error")
	newError := errors.New("new error")
	containerInfo := container.ContainerInfo{ID: initialID, Error: initialError}

	// Act
	updatedContainerInfo := containerInfo.WithID(newID).WithError(newError)

	// Assert
	assert.Equal(t, newID, updatedContainerInfo.ID, "The ID should be updated to the new value")
	assert.Equal(t, newError, updatedContainerInfo.Error, "The Error should be updated to the new value")
	assert.Equal(t, initialID, containerInfo.ID, "The original ContainerInfo should remain unchanged")
	assert.Equal(t, initialError, containerInfo.Error, "The original ContainerInfo should remain unchanged")
}
