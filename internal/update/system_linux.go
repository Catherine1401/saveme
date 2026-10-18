package update

import (
	"context"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
	"time"
)

func AcquireLock(path string) (func(), error) {
	const mode = 0600
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, mode)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, err
	}
	return func() { file.Close() }, nil
}

func (s System) Lock() (func(), error) {
	const path = "/run/saveme-update.lock"
	return AcquireLock(path)
}

func (s System) control(action string) error {
	const tool, name, timeout = "systemctl", "saved.service", 60 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return exec.CommandContext(ctx, tool, action, name).Run()
}

func (s System) Stop() error  { const action = "stop"; return s.control(action) }
func (s System) Start() error { const action = "start"; return s.control(action) }

func (s System) readStatus() ([]byte, error) {
	const base, tool, ownerFlag, delimiter, status, timeout = 10, "runuser", "-u", "--", "-status", 8 * time.Second
	owner, err := user.LookupId(strconv.FormatUint(uint64(*s.Config.AllowedUID), base))
	if err != nil {
		return nil, err
	}
	version, err := s.Paths.ReadVersion()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return exec.CommandContext(ctx, tool, ownerFlag, owner.Username, delimiter, s.Paths.Payload(version), status).Output()
}

func (s System) Protect() error   { return nil }
func (s System) Unprotect() error { return nil }
