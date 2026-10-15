package daemon

import (
	"errors"
	"net"
	"os"
	"syscall"

	"saveme/internal/core"
)

type Server struct {
	Engine   *Engine
	UID      uint32
	listener *net.UnixListener
	lock     *os.File
}

func (s *Server) Listen(path string) error {
	const network, suffix, mode = "unix", ".lock", 0600
	const zero, unchangedGroup = 0, -1
	lock, err := os.OpenFile(path+suffix, os.O_CREATE|os.O_RDWR, mode)
	if err != nil {
		return err
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		return err
	}
	s.lock = lock
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSocket == zero {
			s.Close()
			return core.ErrInvalid
		}
		if err := os.Remove(path); err != nil {
			s.Close()
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		s.Close()
		return err
	}
	listener, err := net.ListenUnix(network, &net.UnixAddr{Name: path, Net: network})
	if err != nil {
		s.Close()
		return err
	}
	s.listener = listener
	if err := os.Chmod(path, mode); err != nil {
		s.Close()
		return err
	}
	if err := os.Chown(path, int(s.UID), unchangedGroup); err != nil {
		s.Close()
		return err
	}
	return nil
}

func (s *Server) Close() {
	if s.listener != nil {
		s.listener.Close()
	}
	if s.lock != nil {
		s.lock.Close()
	}
}

func peerUID(conn *net.UnixConn) (uint32, error) {
	const zero = 0
	raw, err := conn.SyscallConn()
	if err != nil {
		return zero, err
	}
	var credentials *syscall.Ucred
	var controlErr error
	if err := raw.Control(func(fd uintptr) {
		credentials, controlErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil {
		return zero, err
	}
	if controlErr != nil {
		return zero, controlErr
	}
	return credentials.Uid, nil
}

// Nhận kết nối kế tiếp; trả nil khi peer không phải tài khoản được phép.
func (s *Server) accept() (net.Conn, error) {
	conn, err := s.listener.AcceptUnix()
	if err != nil {
		return nil, err
	}
	uid, err := peerUID(conn)
	if err != nil || uid != s.UID {
		conn.Close()
		return nil, nil
	}
	return conn, nil
}
