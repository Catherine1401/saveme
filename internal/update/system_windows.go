package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
	"saveme/internal/core"
	"saveme/internal/daemon"
	"saveme/internal/enforce"
	"saveme/internal/store"
)

func AcquireLock(path string) (func(), error) {
	const flags = windows.LOCKFILE_EXCLUSIVE_LOCK | windows.LOCKFILE_FAIL_IMMEDIATELY
	const mode, zero, one = 0600, 0, 1
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, mode)
	if err != nil {
		return nil, err
	}
	overlap := &windows.Overlapped{}

	if err := windows.LockFileEx(windows.Handle(file.Fd()), flags, zero, one, zero, overlap); err != nil {
		file.Close()
		return nil, err
	}
	return func() { file.Close() }, nil
}

func (s System) Lock() (func(), error) {
	const name = "update.lock"
	return AcquireLock(filepath.Join(s.Paths.Root, name))
}

func (s System) control(stop bool) error {
	const name = "saved"
	const attempts = 60
	manager, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer manager.Disconnect()
	service, err := manager.OpenService(name)
	if err != nil {
		return err
	}
	defer service.Close()
	target := svc.Running
	current, err := service.Query()
	if err != nil {
		return err
	}
	if stop {
		target = svc.Stopped
	}
	if current.State == target {
		return nil
	}
	if stop {
		if current.State != svc.StopPending {
			_, err = service.Control(svc.Stop)
		}
	} else if current.State != svc.StartPending {
		err = service.Start()
	}
	if err != nil {
		return err
	}
	for range attempts {
		const interval = time.Second
		current, err = service.Query()
		if err != nil {
			return err
		}
		if current.State == target {
			return nil
		}
		time.Sleep(interval)
	}
	return os.ErrDeadlineExceeded
}

func (s System) Stop() error  { return s.control(true) }
func (s System) Start() error { return s.control(false) }
func (s System) readStatus() ([]byte, error) {
	client, err := daemon.Dial(s.Config.Socket)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	response, err := client.Call(core.Request{Cmd: core.Status})
	if err != nil {
		return nil, err
	}
	return json.Marshal(response)
}

func (s System) Protect() error {
	state, err := store.New(s.Config.StateDir, nil).Load()
	if err != nil {
		return err
	}
	for _, entry := range state.Entries {
		if entry.Lock != nil {
			firewall, err := enforce.ConfiguredFirewall(s.Config.Upstreams, s.Config.DoH, enforce.PowerShell)
			if err != nil {
				return err
			}
			return firewall.Guard(true)
		}
	}
	return nil
}

func (s System) Unprotect() error {
	firewall, err := enforce.ConfiguredFirewall(s.Config.Upstreams, s.Config.DoH, enforce.PowerShell)
	if err != nil {
		return err
	}
	return firewall.Guard(false)
}
