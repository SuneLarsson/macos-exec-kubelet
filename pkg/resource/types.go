package resource

import "time"

const (
	// MacOSRuntime is the runtime name for macOS virtual machines.
	MacOSRuntime = "vz"

	// ContainerRuntime is the runtime name for containerized workloads.
	ContainerRuntime = "docker"
)

type ExecAction struct {
	// Command is the command line to execute inside the container.
	// Exit status of 0 is treated as live/healthy and non-zero is unhealthy.
	Command []string

	// TimeoutDuration is the maximum duration to wait for the command to complete.
	TimeoutDuration time.Duration
}
