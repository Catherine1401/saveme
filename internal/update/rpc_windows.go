package update

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
	"saveme/internal/config"
)

func listen(path string, cfg config.Config) (net.Listener, func(), error) {
	const descriptor = "D:P(A;;GA;;;SY)(A;;GRGW;;;%s)"
	listener, err := winio.ListenPipe(path, &winio.PipeConfig{SecurityDescriptor: fmt.Sprintf(descriptor, cfg.AllowedSID)})
	return listener, func() {}, err
}

func authorized(net.Conn, config.Config) bool { return true }
func dial(path string) (net.Conn, error) {
	const wait = time.Second
	timeout := wait
	return winio.DialPipe(path, &timeout)
}

func Session(path string, exclusive bool) (func(), error) {
	const zero, one = 0, 1
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	flags := uint32(windows.LOCKFILE_FAIL_IMMEDIATELY)
	if exclusive {
		flags |= windows.LOCKFILE_EXCLUSIVE_LOCK
	}
	overlap := &windows.Overlapped{}
	if err := windows.LockFileEx(windows.Handle(file.Fd()), flags, zero, one, zero, overlap); err != nil {
		file.Close()
		return nil, err
	}
	return func() { file.Close() }, nil
}
