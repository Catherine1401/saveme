package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
			err = checkPreservedUpdate(cfg)
			if err != nil {
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
	if err := checkPreservedUpdate(cfg); err == nil {
		t.Fatal(acceptedMissing)
	}
	for _, name := range []string{primary, backup} {
		if err := os.WriteFile(filepath.Join(cfg.StateDir, name), []byte(broken), mode); err != nil {
			t.Fatal(err)
		}
	}
	if err := checkPreservedUpdate(cfg); err == nil {
		t.Fatal(acceptedCorrupt)
	}
}

func TestRemovalGuardStillRejectsStoredLocks(t *testing.T) {
	const unlocked, timed, daily = "unlocked", "timed", "daily"
	const domain, zone, start, end = "example.com", "UTC", 100, 200
	cases := []struct {
		name string
		lock *core.Lock
	}{
		{name: unlocked},
		{name: timed, lock: &core.Lock{Kind: core.Range, Zone: zone, StartAt: start, EndAt: end}},
		{name: daily, lock: &core.Lock{Kind: core.Daily, Zone: zone, Windows: []core.Window{{Start: start, End: end}}}},
	}
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			cfg := config.Config{StateDir: t.TempDir()}
			state := core.EmptyState()
			state.Entries = []core.Entry{{Domain: domain, Lock: scenario.lock}}
			if err := store.New(cfg.StateDir, nil).Save(state); err != nil {
				t.Fatal(err)
			}
			if err := checkUpdate(cfg); (err != nil) != (scenario.lock != nil) {
				t.Fatal(err)
			}
		})
	}
}

func TestPreservedUpdateRefusesLegacyBeforeMigration(t *testing.T) {
	const version, generation, mode, primary = 1, 1, 0600, "state.json"
	const domain = "example.com"
	type legacyState struct {
		Version int        `json:"version"`
		Domains []string   `json:"domains"`
		Lock    *core.Lock `json:"lock,omitempty"`
		MaxSeen int64      `json:"max_seen"`
	}
	type payload struct {
		Generation uint64      `json:"generation"`
		State      legacyState `json:"state"`
	}
	type record struct {
		Generation uint64      `json:"generation"`
		State      legacyState `json:"state"`
		Checksum   string      `json:"checksum"`
	}
	state := legacyState{Version: version, Domains: []string{domain}}
	data, err := json.Marshal(payload{Generation: generation, State: state})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	data, err = json.Marshal(record{Generation: generation, State: state, Checksum: hex.EncodeToString(sum[:])})
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{StateDir: t.TempDir()}
	path := filepath.Join(cfg.StateDir, primary)
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
	if err := checkUpdate(cfg); err != nil {
		t.Fatal(err)
	}
	if err := checkPreservedUpdate(cfg); err == nil {
		const accepted = "legacy state accepted for snapshot rollback"
		t.Fatal(accepted)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, after) {
		t.Fatal(err)
	}
}
