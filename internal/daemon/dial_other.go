//go:build !windows

package daemon

import (
	"net"
	"time"
)

// Kết nối Unix socket của daemon.
func dial(path string, timeout time.Duration) (net.Conn, error) {
	const network = "unix"
	return net.DialTimeout(network, path, timeout)
}
