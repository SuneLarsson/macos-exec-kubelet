package resource

import "time"

// ContainerStatus represents the status of a container.
type ContainerStatus int

const (
	// Container is waiting to be created.
	ContainerStatusWaiting ContainerStatus = iota

	// Container has been created
	ContainerStatusCreated

	// Container is currently running
	ContainerStatusRunning

	// Container has been paused
	ContainerStatusPaused

	// Container is in the process of restarting.
	ContainerStatusRestarting

	// Container was killed due to an out-of-memory condition.
	ContainerStatusOOMKilled

	// Container has terminated.
	ContainerStatusDead

	// Container status is unknown.
	ContainerStatusUnknown
)

// ContainerState holds information about the current and past state of a container.
type ContainerState struct {
	Status     ContainerStatus
	StartedAt  time.Time
	FinishedAt time.Time
	ExitCode   int
	Error      string
}

// Container represents a single container with its ID, name, and state.
type Container struct {
	ID    string
	Name  string
	State ContainerState
}
