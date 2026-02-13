package node

import (
	"io"

	vzio "github.com/agoda-com/macOS-vz-kubelet/internal/io"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
)

// ExecIO implements the api.AttachIO interface and holds the I/O streams and resize channel.
type ExecIO struct {
	tty      bool
	stdin    io.Reader
	stdout   io.WriteCloser
	stderr   io.WriteCloser
	chResize chan api.TermSize
}

// NewExecIO creates a new ExecIO with the given I/O streams and resize channel.
func NewExecIO(tty bool, stdin io.Reader, stdout, stderr io.WriteCloser, chResize chan api.TermSize) *ExecIO {
	return &ExecIO{
		tty:      tty,
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
		chResize: chResize,
	}
}

// TTY returns whether the terminal is a TTY.
func (e *ExecIO) TTY() bool {
	return e.tty
}

// Stdin returns the stdin reader.
func (e *ExecIO) Stdin() io.Reader {
	return e.stdin
}

// Stdout returns the stdout writer.
func (e *ExecIO) Stdout() io.WriteCloser {
	return e.stdout
}

// Stderr returns the stderr writer.
func (e *ExecIO) Stderr() io.WriteCloser {
	return e.stderr
}

// Resize returns the resize channel.
func (e *ExecIO) Resize() <-chan api.TermSize {
	return e.chResize
}

// DiscardingExecIO creates a new ExecIO with stdout set to discard all data.
// nolint: ireturn
func DiscardingExecIO() *ExecIO {
	return &ExecIO{
		tty:    false,
		stdout: &vzio.DiscardWriteCloser{},
	}
}
