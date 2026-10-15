package daemon

import (
	"net"
	"time"

	"github.com/Microsoft/go-winio"
)

// Kết nối named pipe của daemon.
func dial(path string, timeout time.Duration) (net.Conn, error) {
	return winio.DialPipe(path, &timeout)
}
