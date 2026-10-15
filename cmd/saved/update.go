package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"saveme/internal/config"
	"saveme/internal/core"
	"saveme/internal/store"
)

func checkStoredUpdate(cfg config.Config) error {
	const primary = "state.json"
	if _, err := os.Stat(filepath.Join(cfg.StateDir, primary)); errors.Is(err, os.ErrNotExist) {
		const backup = "state.bak.json"
		if _, err := os.Stat(filepath.Join(cfg.StateDir, backup)); errors.Is(err, os.ErrNotExist) {
			const missing = "saved state is missing; refusing update"
			return errors.New(missing)
		}
	}
	_, err := store.New(cfg.StateDir, nil).Load()
	if err != nil {
		return err
	}
	return nil
}

func checkUpdate(cfg config.Config) error {
	if err := checkStoredUpdate(cfg); err != nil {
		return err
	}
	state, err := store.New(cfg.StateDir, nil).Load()
	if err != nil {
		return err
	}
	for _, entry := range state.Entries {
		if entry.Lock != nil {
			const locked = "remove every schedule and wait for every timed lock to expire before updating or uninstalling"
			return errors.Join(core.ErrLocked, errors.New(locked))
		}
	}
	return nil
}

func checkPreservedUpdate(cfg config.Config) error {
	if err := checkStoredUpdate(cfg); err != nil {
		return err
	}
	const primary, backup, legacyVersion = "state.json", "state.bak.json", 1
	for _, name := range []string{primary, backup} {
		data, err := os.ReadFile(filepath.Join(cfg.StateDir, name))
		if err != nil {
			continue
		}
		var record struct {
			State struct {
				Version int `json:"version"`
			} `json:"state"`
		}
		if json.Unmarshal(data, &record) == nil && record.State.Version == legacyVersion {
			const migration = "legacy state requires migration before a preserved update; refusing before service stop"
			return errors.New(migration)
		}
	}
	return nil
}
