package update

import (
	"net"
	"os"
	"syscall"
	"time"

	"saveme/internal/config"
)

func listen(path string, cfg config.Config) (net.Listener, func(), error) {
	const suffix = ".lock"
	const mode = 0600
	const network = "unix"
	unlock, err := AcquireLock(path + suffix)
	if err != nil {
		return nil, nil, err
	}
	if info, err := os.Lstat(path); err == nil {
		const literal1 = 0
		if info.Mode()&os.ModeSocket == literal1 {
			unlock()
			return nil, nil, os.ErrInvalid
		}
		if err := os.Remove(path); err != nil {
			unlock()
			return nil, nil, err
		}
	} else if !os.IsNotExist(err) {
		unlock()
		return nil, nil, err
	}

	listener, err := net.Listen(network, path)
	if err != nil {
		unlock()
		return nil, nil, err
	}
	if err = os.Chmod(path, mode); err == nil {
		const unchangedGroup = -1
		err = os.Chown(path, int(*cfg.AllowedUID), unchangedGroup)
	}
	if err != nil {
		listener.Close()
		unlock()
		return nil, nil, err
	}
	return listener, unlock, nil
}

func authorized(connection net.Conn, cfg config.Config) bool {
	const zero = 0
	unix, ok := connection.(*net.UnixConn)
	if !ok {
		return false
	}
	raw, err := unix.SyscallConn()
	if err != nil {
		return false
	}
	var credential *syscall.Ucred
	var credentialErr error
	err = raw.Control(func(fd uintptr) {
		credential, credentialErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	})
	return err == nil && credentialErr == nil && credential != nil && credential.Uid != uint32(zero) && credential.Uid == *cfg.AllowedUID
}

func dial(path string) (net.Conn, error) {
	const network, timeout = "unix", time.Second
	return net.DialTimeout(network, path, timeout)
}

func Session(path string, exclusive bool) (func(), error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	flags := syscall.LOCK_SH | syscall.LOCK_NB
	if exclusive {
		flags = syscall.LOCK_EX | syscall.LOCK_NB
	}
	if err := syscall.Flock(int(file.Fd()), flags); err != nil {
		file.Close()
		return nil, err
	}
	return func() { file.Close() }, nil
}
