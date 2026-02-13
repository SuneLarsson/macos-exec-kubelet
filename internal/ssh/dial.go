// Note: Remove when proposal https://go-review.googlesource.com/c/crypto/+/550096 is merged
package ssh

import (
	"context"
	"net"

	"github.com/virtual-kubelet/virtual-kubelet/log"
	"golang.org/x/crypto/ssh"
)

// DialContext starts a client connection to the given SSH server. It is a
// convenience function that connects to the given network address,
// initiates the SSH handshake, and then sets up a Client.
//
// The provided Context must be non-nil. If the context expires before the
// connection is complete, an error is returned. Once successfully connected,
// any expiration of the context will not affect the connection.
//
// See [Dial] for additional information.
func DialContext(ctx context.Context, network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	d := net.Dialer{
		Timeout: config.Timeout,
	}
	conn, err := d.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	type result struct {
		client *ssh.Client
		err    error
	}
	ch := make(chan result)
	go func() {
		var client *ssh.Client
		c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
		if err == nil {
			client = ssh.NewClient(c, chans, reqs)
		}
		select {
		case ch <- result{client, err}:
		case <-ctx.Done():
			if client != nil {
				if err := client.Close(); err != nil {
					log.G(ctx).WithError(err).Error("Failed to close client")
				}
			}
		}
	}()
	select {
	case res := <-ch:
		return res.client, res.err
	case <-ctx.Done():
		return nil, context.Cause(ctx)
	}
}
