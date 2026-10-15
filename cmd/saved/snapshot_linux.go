package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"syscall"

	"saveme/internal/config"
	"saveme/internal/core"
	"saveme/internal/store"
)

func readUpdateJSON(path string, result any) error {
	const limit = 1024 * 1024
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) > limit {
		return core.ErrStorage
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(result); err != nil {
		return err
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return core.ErrInvalid
	}
	return nil
}

func stoppedUpdate(cfg config.Config, action func() error) error {
	const suffix, mode = ".lock", 0600
	file, err := os.OpenFile(cfg.Socket+suffix, os.O_RDWR, mode)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return err
	}
	return action()
}

func snapshotUpdate(cfg config.Config, path string) error {
	return stoppedUpdate(cfg, func() error {
		const mode = 0600
		if err := checkPreservedUpdate(cfg); err != nil {
			return err
		}
		state, err := store.New(cfg.StateDir, nil).Load()
		if err != nil {
			return err
		}
		data, err := json.Marshal(state)
		if err != nil {
			return err
		}
		return os.WriteFile(path, data, mode)
	})
}

func restoreUpdate(cfg config.Config, path string) error {
	var snapshot core.State
	if err := readUpdateJSON(path, &snapshot); err != nil {
		return err
	}
	if err := snapshot.Validate(); err != nil {
		return err
	}
	return stoppedUpdate(cfg, func() error {
		repo := store.New(cfg.StateDir, store.Immutable{})
		if current, err := repo.Load(); err == nil && current.MaxSeen > snapshot.MaxSeen {
			snapshot.MaxSeen = current.MaxSeen
		}
		return repo.Save(snapshot)
	})
}
