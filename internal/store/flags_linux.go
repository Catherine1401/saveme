package store

import (
	"os"
	"syscall"
	"unsafe"
)

type Immutable struct{}

func (Immutable) Set(path string, enabled bool) error {
	const getFlags, setFlags, immutable, zero = 0x80086601, 0x40086602, 0x10, 0
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	var flags uint32
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), getFlags, uintptr(unsafe.Pointer(&flags))); errno != zero {
		return errno
	}
	if enabled {
		flags |= immutable
	} else {
		flags &^= immutable
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), setFlags, uintptr(unsafe.Pointer(&flags)))
	if errno != zero {
		return errno
	}
	return nil
}
