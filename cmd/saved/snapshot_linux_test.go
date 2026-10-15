package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"

	"saveme/internal/config"
	"saveme/internal/core"
	"saveme/internal/store"
)

func TestSnapshotRequiresStoppedDaemonAndPreservesState(t *testing.T) {
	const socket, snapshot, suffix, mode = "socket", "snapshot.json", ".lock", 0600
	const refused = "snapshot accepted while daemon holds lock"
	root := t.TempDir()
	cfg := config.Config{Socket: filepath.Join(root, socket), StateDir: t.TempDir()}
	state := updateState()
	if err := store.New(cfg.StateDir, nil).Save(state); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(cfg.Socket+suffix, os.O_CREATE|os.O_RDWR, mode)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, snapshot)
	if err := snapshotUpdate(cfg, path); err == nil {
		t.Fatal(refused)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	if err := snapshotUpdate(cfg, path); err != nil {
		t.Fatal(err)
	}
	var loaded core.State
	if err := readUpdateJSON(path, &loaded); err != nil || !reflect.DeepEqual(state, loaded) {
		t.Fatal(loaded, err)
	}
}

func TestUpdateJSONRejectsUnknownTrailingAndOversizeData(t *testing.T) {
	const name, mode, limit = "snapshot.json", 0600, 1024 * 1024
	const unknown, trailing, oversized = "unknown", "trailing", "oversized"
	valid, err := json.Marshal(updateState())
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		data []byte
	}{
		{unknown, []byte(`{"unknown":true}`)},
		{trailing, append(valid, valid...)},
		{oversized, make([]byte, limit+len(valid))},
	}
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), name)
			if err := os.WriteFile(path, scenario.data, mode); err != nil {
				t.Fatal(err)
			}
			var state core.State
			if err := readUpdateJSON(path, &state); err == nil {
				t.Fatal(scenario.name)
			}
		})
	}
}

func TestUpdateHeldTracksExclusiveLock(t *testing.T) {
	const name, mode = "update.lock", 0600
	const missing, unlocked, held = "missing lock is busy", "unlocked file is busy", "exclusive lock was ignored"
	path := filepath.Join(t.TempDir(), name)
	if updateHeld(path) {
		t.Fatal(missing)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, mode)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if updateHeld(path) {
		t.Fatal(unlocked)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	if !updateHeld(path) {
		t.Fatal(held)
	}
}
