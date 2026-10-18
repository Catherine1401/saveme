package main

import (
	"os"
	"path/filepath"

	"saveme/internal/config"
	"saveme/internal/core"
	"saveme/internal/daemon"
	"saveme/internal/enforce"
	"saveme/internal/store"
	"saveme/internal/update"
)

func stoppedUpdate(cfg config.Config, action func() error) error {
	client, err := daemon.Dial(cfg.Socket)
	if err == nil {
		client.Close()
		return core.ErrLocked
	}
	return action()
}

func verifyUpdate(cfg config.Config, snapshotPath, statusPath string) error {
	var before core.State
	if err := readUpdateJSON(snapshotPath, &before); err != nil {
		return err
	}
	if err := before.Validate(); err != nil {
		return err
	}
	var status core.Response
	if err := readUpdateJSON(statusPath, &status); err != nil {
		return err
	}
	if err := checkPreservedUpdate(cfg); err != nil {
		return err
	}
	after, err := store.New(cfg.StateDir, nil).Load()
	if err != nil {
		return err
	}
	if err := preservedUpdate(before, after, status); err != nil {
		return err
	}
	enforcer, err := enforce.ConfiguredFirewall(cfg.Upstreams, cfg.DoH, enforce.PowerShell)
	if err != nil {
		return err
	}
	if err := enforcer.Verify(status.Active); err != nil {
		return err
	}
	if status.Active {
		const v4, v6 = "127.0.0.1:53", "[::1]:53"
		return verifyBlockedDNS(activeDomains(after, status), []string{v4, v6})
	}
	return nil
}

func updatePending() bool {
	const folder, journal, lock = "transaction", "journal.json", "update.lock"
	paths := update.Installed()
	if _, err := os.Stat(filepath.Join(paths.Root, folder, journal)); !os.IsNotExist(err) {
		return true
	}
	lockPath := filepath.Join(paths.Root, lock)
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		return false
	}
	unlock, err := update.AcquireLock(lockPath)
	if err != nil {
		return true
	}
	unlock()
	return false
}
