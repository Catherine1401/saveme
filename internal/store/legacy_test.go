package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"saveme/internal/core"
)

// Ghi record v1 với checksum tính theo cấu trúc v1 như binary cũ.
func writeLegacy(t *testing.T, dir string, legacy legacyState, sum string) {
	const generation, mode = 1, 0600
	const primary = "state.json"
	if sum == "" {
		computed, err := checksum(legacy, generation)
		if err != nil {
			t.Fatal(err)
		}
		sum = computed
	}
	data, err := json.Marshal(struct {
		Generation uint64      `json:"generation"`
		State      legacyState `json:"state"`
		Checksum   string      `json:"checksum"`
	}{Generation: generation, State: legacy, Checksum: sum})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, primary), data, mode); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyStateMigratesSharedLockToEachDomain(t *testing.T) {
	const version, zone, first, second, start, end, maxSeen, empty = 1, "UTC", "a.com", "b.com", 60, 120, 99, ""
	lock := &core.Lock{Kind: core.Daily, Zone: zone, Windows: []core.Window{{Start: start, End: end}}}
	dir := t.TempDir()
	writeLegacy(t, dir, legacyState{Version: version, Domains: []string{first, second}, Lock: lock, MaxSeen: maxSeen}, empty)
	loaded, err := New(dir, nil).Load()
	if err != nil {
		t.Fatal(err)
	}
	const count = 2
	if loaded.Version != core.StateVersion || loaded.MaxSeen != maxSeen || len(loaded.Entries) != count {
		t.Fatal(loaded)
	}
	for index, domain := range []string{first, second} {
		entry := loaded.Entries[index]
		if entry.Domain != domain || entry.Lock == nil || entry.Lock.Windows[0] != lock.Windows[0] {
			t.Fatal(entry)
		}
	}
	if loaded.Entries[0].Lock == loaded.Entries[1].Lock {
		const shared = "migrated entries share one lock"
		t.Fatal(shared)
	}
	if err := New(dir, nil).Save(loaded); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyStateWithoutLockAndBadChecksum(t *testing.T) {
	const version, domain, empty, wrong = 1, "a.com", "", "00"
	dir := t.TempDir()
	writeLegacy(t, dir, legacyState{Version: version, Domains: []string{domain}}, empty)
	loaded, err := New(dir, nil).Load()
	if err != nil || len(loaded.Entries) != version || loaded.Entries[0].Lock != nil {
		t.Fatal(loaded, err)
	}
	broken := t.TempDir()
	writeLegacy(t, broken, legacyState{Version: version, Domains: []string{domain}}, wrong)
	if _, err := New(broken, nil).Load(); err == nil {
		const accepted = "bad legacy checksum accepted"
		t.Fatal(accepted)
	}
}
