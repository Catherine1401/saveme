package main

import (
	"errors"
	"os"
	"path/filepath"

	"saveme/internal/config"
	"saveme/internal/core"
	"saveme/internal/store"
)

func checkUpdate(cfg config.Config) error {
	const primary = "state.json"
	if _, err := os.Stat(filepath.Join(cfg.StateDir, primary)); errors.Is(err, os.ErrNotExist) {
		const backup = "state.bak.json"
		if _, err := os.Stat(filepath.Join(cfg.StateDir, backup)); errors.Is(err, os.ErrNotExist) {
			const missing = "saved state is missing; refusing update"
			return errors.New(missing)
		}
	}
	state, err := store.New(cfg.StateDir, nil).Load()
	if err != nil {
		return err
	}
	for _, entry := range state.Entries {
		if entry.Lock != nil {
			const locked = "remove every schedule and wait for every timed lock to expire before updating"
			return errors.Join(core.ErrLocked, errors.New(locked))
		}
	}
	return nil
}
