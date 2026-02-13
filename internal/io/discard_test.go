package io_test

import (
	"testing"

	"github.com/agoda-com/macOS-vz-kubelet/internal/io"
	"github.com/stretchr/testify/assert"
)

func TestDiscardWriteCloser_Write(t *testing.T) {
	dwc := &io.DiscardWriteCloser{}
	n, err := dwc.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
}

func TestDiscardWriteCloser_Close(t *testing.T) {
	dwc := &io.DiscardWriteCloser{}
	assert.NoError(t, dwc.Close())
}
