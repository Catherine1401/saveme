package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"saveme/internal/config"
	"saveme/internal/core"
	"saveme/internal/store"
)

func TestUpdateGuardPreservesState(t *testing.T) {
	const unlocked, timed, daily = "unlocked", "timed", "daily"
	const zone = "UTC"
	const start, end, minute = 100, 200, 60
	cases := []struct {
		name string
		lock *core.Lock
	}{
		{name: unlocked},
		{name: timed, lock: &core.Lock{Kind: core.Range, Zone: zone, StartAt: start, EndAt: end}},
		{name: daily, lock: &core.Lock{Kind: core.Daily, Zone: zone, Windows: []core.Window{{Start: start, End: start + minute}}}},
	}
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			const domain, other, primary = "example.com", "other.com", "state.json"
			const changed, expected = "guard changed stored state", "unexpected update permission"
			cfg := config.Config{StateDir: t.TempDir()}
			state := core.EmptyState()
			state.Entries = []core.Entry{{Domain: domain}, {Domain: other, Lock: scenario.lock}}
			if err := store.New(cfg.StateDir, nil).Save(state); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(cfg.StateDir, primary)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			err = checkUpdate(cfg)
			if (scenario.lock == nil && err != nil) || (scenario.lock != nil && !errors.Is(err, core.ErrLocked)) {
				t.Fatal(expected, err)
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal(changed, err)
			}
		})
	}
}

func TestUpdateGuardRejectsMissingAndCorruptState(t *testing.T) {
	const primary, backup, broken, mode = "state.json", "state.bak.json", "broken", 0600
	const acceptedMissing, acceptedCorrupt = "missing state accepted", "corrupt state accepted"
	cfg := config.Config{StateDir: t.TempDir()}
	if err := checkUpdate(cfg); err == nil {
		t.Fatal(acceptedMissing)
	}
	for _, name := range []string{primary, backup} {
		if err := os.WriteFile(filepath.Join(cfg.StateDir, name), []byte(broken), mode); err != nil {
			t.Fatal(err)
		}
	}
	if err := checkUpdate(cfg); err == nil {
		t.Fatal(acceptedCorrupt)
	}
}
