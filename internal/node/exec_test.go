package node_test

import (
	"testing"

	vzio "github.com/agoda-com/macOS-vz-kubelet/internal/io"
	"github.com/agoda-com/macOS-vz-kubelet/internal/node"
	"github.com/stretchr/testify/assert"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
)

type mockReader struct{}

func (m *mockReader) Read(p []byte) (n int, err error) {
	return len(p), nil
}

type mockWriteCloser struct{}

func (m *mockWriteCloser) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockWriteCloser) Close() error {
	return nil
}

func TestNewExecIO(t *testing.T) {
	stdin := &mockReader{}
	stdout := &mockWriteCloser{}
	stderr := &mockWriteCloser{}
	chResize := make(chan api.TermSize)

	execIO := node.NewExecIO(true, stdin, stdout, stderr, chResize)

	assert.True(t, execIO.TTY())
	assert.Equal(t, stdin, execIO.Stdin())
	assert.Equal(t, stdout, execIO.Stdout())
	assert.Equal(t, stderr, execIO.Stderr())
	assert.Equal(t, (<-chan api.TermSize)(chResize), execIO.Resize())
}

func TestExecIO_TTY(t *testing.T) {
	execIO := node.NewExecIO(true, nil, nil, nil, nil)
	assert.True(t, execIO.TTY())

	execIO = node.NewExecIO(false, nil, nil, nil, nil)
	assert.False(t, execIO.TTY())
}

func TestExecIO_Stdin(t *testing.T) {
	stdin := &mockReader{}
	execIO := node.NewExecIO(false, stdin, nil, nil, nil)
	assert.Equal(t, stdin, execIO.Stdin())
}

func TestExecIO_Stdout(t *testing.T) {
	stdout := &mockWriteCloser{}
	execIO := node.NewExecIO(false, nil, stdout, nil, nil)
	assert.Equal(t, stdout, execIO.Stdout())
}

func TestExecIO_Stderr(t *testing.T) {
	stderr := &mockWriteCloser{}
	execIO := node.NewExecIO(false, nil, nil, stderr, nil)
	assert.Equal(t, stderr, execIO.Stderr())
}

func TestExecIO_Resize(t *testing.T) {
	chResize := make(chan api.TermSize)
	execIO := node.NewExecIO(false, nil, nil, nil, chResize)
	assert.Equal(t, (<-chan api.TermSize)(chResize), execIO.Resize())
}

func TestDiscardingExecIO(t *testing.T) {
	execIO := node.DiscardingExecIO()
	assert.False(t, execIO.TTY())
	assert.Nil(t, execIO.Stdin())
	assert.IsType(t, &vzio.DiscardWriteCloser{}, execIO.Stdout())
	assert.Nil(t, execIO.Stderr())
	assert.Nil(t, execIO.Resize())
}
