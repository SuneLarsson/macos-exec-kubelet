package ssh

import (
	"context"
	"time"

	"github.com/virtual-kubelet/virtual-kubelet/log"
	"golang.org/x/crypto/ssh"
)

// SendKeepalive sends keepalive messages to the SSH server.
func SendKeepalive(ctx context.Context, conn ssh.Conn) {
	logger := log.G(ctx)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, _, err := conn.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				logger.Errorf("Failed to send keep-alive: %s", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
