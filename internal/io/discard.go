package io

// DiscardWriteCloser is an io.WriteCloser that discards all data written to it.
type DiscardWriteCloser struct{}

// Write discards the data and returns the length of the data.
func (d *DiscardWriteCloser) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// Close is a no-op and returns nil.
func (d *DiscardWriteCloser) Close() error {
	return nil
}
