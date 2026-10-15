package store

import (
	"os"
	"path/filepath"
	"testing"

	"saveme/internal/core"
)

func TestRecoveryAndCorruption(t *testing.T) {
	const (
		number0 = 0
	)

	const domain, primary, backup, broken, mode = "example.com", "state.json", "state.bak.json", "broken", 0600
	dir := t.TempDir()
	store := New(dir, nil)
	state, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	state.Entries = []core.Entry{{Domain: domain}}
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, primary), []byte(broken), mode); err != nil {
		t.Fatal(err)
	}
	recovered, err := New(dir, nil).Load()
	if err != nil || recovered.Entries[number0].Domain != domain {
		t.Fatal(recovered, err)
	}
	if err := os.WriteFile(filepath.Join(dir, backup), []byte(broken), mode); err != nil {
		t.Fatal(err)
	}
	if _, err := New(dir, nil).Load(); err == nil {
		const (
			textBothCorruptMustFail = "both corrupt must fail"
		)

		t.Fatal(textBothCorruptMustFail)
	}
}

func TestInterruptedWriteKeepsValidCopy(t *testing.T) {
	const domain, temporary, incomplete, mode = "example.com", ".state-interrupted", "{", 0600
	dir := t.TempDir()
	store := New(dir, nil)
	state := core.EmptyState()
	state.Entries = []core.Entry{{Domain: domain}}
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, temporary), []byte(incomplete), mode); err != nil {
		t.Fatal(err)
	}
	if loaded, err := New(dir, nil).Load(); err != nil || len(loaded.Entries) != len(state.Entries) {
		t.Fatal(loaded, err)
	}
}

func TestNewestValidGenerationWins(t *testing.T) {
	const (
		number0 = 0
	)

	const first, second, backup, primary, mode = "example.com", "youtube.com", "state.bak.json", "state.json", 0600
	dir := t.TempDir()
	store := New(dir, nil)
	state := core.EmptyState()
	state.Entries = []core.Entry{{Domain: first}}
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	old, err := os.ReadFile(filepath.Join(dir, backup))
	if err != nil {
		t.Fatal(err)
	}
	state.Entries = []core.Entry{{Domain: second}}
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, backup), old, mode); err != nil {
		t.Fatal(err)
	}
	loaded, err := New(dir, nil).Load()
	if err != nil || loaded.Entries[number0].Domain != second {
		t.Fatal(loaded, err)
	}
	newer, err := os.ReadFile(filepath.Join(dir, primary))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, backup), newer, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, primary), old, mode); err != nil {
		t.Fatal(err)
	}
	loaded, err = New(dir, nil).Load()
	if err != nil || loaded.Entries[number0].Domain != second {
		t.Fatal(loaded, err)
	}
}

func TestPartialSaveReportsFailureAndRecoversCommittedCopy(t *testing.T) {
	const backup, directoryMode, domain = "state.bak.json", 0700, "example.com"

	const zero = 0
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, backup), directoryMode); err != nil {
		t.Fatal(err)
	}
	repo := New(dir, nil)
	state := core.EmptyState()
	state.Entries = []core.Entry{{Domain: domain}}
	if err := repo.Save(state); err == nil {
		t.Fatal(core.ErrStorage)
	}
	loaded, err := New(dir, nil).Load()

	if err != nil || loaded.Entries[zero].Domain != domain {
		t.Fatal(loaded, err)
	}
}
