package container

// ContainerInfo holds information about a Docker container.
type ContainerInfo struct {
	ID    string // ID of the container after it was created
	Error error  // Error encountered during container pull or creation
}

// WithID sets the ID of the ContainerInfo and returns the updated ContainerInfo.
func (i ContainerInfo) WithID(id string) ContainerInfo {
	i.ID = id
	return i
}

// WithError sets the Error of the ContainerInfo and returns the updated ContainerInfo.
func (i ContainerInfo) WithError(err error) ContainerInfo {
	i.Error = err
	return i
}
