package node_test

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	vzio "github.com/agoda-com/macOS-vz-kubelet/internal/io"
	"github.com/agoda-com/macOS-vz-kubelet/internal/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
)

func TestGetTerminalType(t *testing.T) {
	t.Run("TERM is set", func(t *testing.T) {
		// Set the TERM environment variable
		term := "vt100"
		require.NoError(t, os.Setenv(node.EnvVarTermType, term))

		termType := node.GetTerminalType()
		assert.Equal(t, term, termType)
	})

	t.Run("TERM is not set", func(t *testing.T) {
		// Unset the TERM environment variable
		require.NoError(t, os.Unsetenv(node.EnvVarTermType))

		termType := node.GetTerminalType()
		assert.Equal(t, node.DefaultTerminalType, termType)
	})
}

func TestGetConsoleSize(t *testing.T) {
	t.Run("attach is nil", func(t *testing.T) {
		ctx := context.Background()
		size := node.GetConsoleSize(ctx, nil)
		assert.Nil(t, size)
	})

	t.Run("attach.TTY() is false", func(t *testing.T) {
		ctx := context.Background()
		attach := node.DiscardingExecIO()

		size := node.GetConsoleSize(ctx, attach)
		assert.Nil(t, size)
	})

	t.Run("attach.Resize() returns valid size", func(t *testing.T) {
		ctx := context.Background()
		resizeCh := make(chan api.TermSize, 1)
		resizeCh <- api.TermSize{Width: 80, Height: 24}

		attach := node.NewExecIO(true, nil, &vzio.DiscardWriteCloser{}, &vzio.DiscardWriteCloser{}, resizeCh)

		size := node.GetConsoleSize(ctx, attach)
		assert.NotNil(t, size)
		assert.Equal(t, uint(24), size[0])
		assert.Equal(t, uint(80), size[1])
	})

	t.Run("attach.Resize() times out", func(t *testing.T) {
		ctxDuration := 1 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), ctxDuration)
		defer cancel()

		resizeCh := make(chan api.TermSize)

		attach := node.NewExecIO(true, nil, &vzio.DiscardWriteCloser{}, &vzio.DiscardWriteCloser{}, resizeCh)

		start := time.Now()
		size := node.GetConsoleSize(ctx, attach)
		elapsed := time.Since(start)

		assert.Equal(t, uint(node.DefaultTermHeight), size[0])
		assert.Equal(t, uint(node.DefaultTermWidth), size[1])
		assert.GreaterOrEqual(t, elapsed, ctxDuration)
	})

	t.Run("attach.Resize() returns zero size", func(t *testing.T) {
		ctx := context.Background()
		resizeCh := make(chan api.TermSize, 1)
		resizeCh <- api.TermSize{Width: 0, Height: 0}

		attach := node.NewExecIO(true, nil, &vzio.DiscardWriteCloser{}, &vzio.DiscardWriteCloser{}, resizeCh)

		size := node.GetConsoleSize(ctx, attach)
		assert.Nil(t, size)
	})
}

func TestHandleTerminalResizing(t *testing.T) {
	t.Run("context is canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		resizeCh := make(chan api.TermSize)
		attach := node.NewExecIO(true, nil, &vzio.DiscardWriteCloser{}, &vzio.DiscardWriteCloser{}, resizeCh)

		resizeFunc := func(size api.TermSize) error {
			return nil
		}

		node.HandleTerminalResizing(ctx, attach, resizeFunc)

		// No assertions needed, just ensuring no panic or error
	})

	t.Run("resizeFunc succeeds", func(t *testing.T) {
		ctx := context.Background()
		resizeCh := make(chan api.TermSize, 1)
		resizeCh <- api.TermSize{Width: 80, Height: 24}

		attach := node.NewExecIO(true, nil, &vzio.DiscardWriteCloser{}, &vzio.DiscardWriteCloser{}, resizeCh)

		resizeFunc := func(size api.TermSize) error {
			assert.Equal(t, uint16(80), size.Width)
			assert.Equal(t, uint16(24), size.Height)
			return nil
		}

		node.HandleTerminalResizing(ctx, attach, resizeFunc)
	})

	t.Run("resizeFunc returns error", func(t *testing.T) {
		ctx := context.Background()
		resizeCh := make(chan api.TermSize, 1)
		resizeCh <- api.TermSize{Width: 80, Height: 24}

		attach := node.NewExecIO(true, nil, &vzio.DiscardWriteCloser{}, &vzio.DiscardWriteCloser{}, resizeCh)

		resizeFunc := func(size api.TermSize) error {
			return io.ErrUnexpectedEOF
		}

		node.HandleTerminalResizing(ctx, attach, resizeFunc)
	})

	t.Run("resizeFunc returns io.EOF", func(t *testing.T) {
		ctx := context.Background()
		resizeCh := make(chan api.TermSize, 1)
		resizeCh <- api.TermSize{Width: 80, Height: 24}

		attach := node.NewExecIO(true, nil, &vzio.DiscardWriteCloser{}, &vzio.DiscardWriteCloser{}, resizeCh)

		resizeFunc := func(size api.TermSize) error {
			return io.EOF
		}

		node.HandleTerminalResizing(ctx, attach, resizeFunc)
	})
}
