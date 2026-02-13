package node

import (
	"context"
	"io"
	"os"

	"github.com/virtual-kubelet/virtual-kubelet/log"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
)

const (
	DefaultTerminalType = "xterm-256color"

	DefaultTermWidth  = 60
	DefaultTermHeight = 120

	EnvVarTermType = "TERM"
)

// GetTerminalType retrieves the terminal type from the environment variables.
func GetTerminalType() string {
	term := os.Getenv(EnvVarTermType)
	if term == "" {
		term = DefaultTerminalType
	}
	return term
}

// GetConsoleSize returns the console size for terminal attachment.
func GetConsoleSize(ctx context.Context, attach api.AttachIO) *[2]uint {
	if attach == nil || !attach.TTY() {
		return nil
	}

	termSize := api.TermSize{Width: DefaultTermWidth, Height: DefaultTermHeight}
	select {
	case termSize = <-attach.Resize():
	case <-ctx.Done():
		log.G(ctx).Warnf("Failed to get terminal size, using default size: %dx%d", termSize.Height, termSize.Width)
	}

	if termSize.Width != 0 && termSize.Height != 0 {
		return &[2]uint{uint(termSize.Height), uint(termSize.Width)}
	}
	return nil
}

// HandleTerminalResizing listens for terminal resize events and adjusts accordingly using the provided resizeFunc.
func HandleTerminalResizing(ctx context.Context, attach api.AttachIO, resizeFunc func(api.TermSize) error) {
	select {
	case <-ctx.Done():
		return
	case size := <-attach.Resize():
		if err := resizeFunc(size); err != nil && err != io.EOF {
			log.G(ctx).Errorf("Failed to resize terminal: %s", err)
		}
	}
}
