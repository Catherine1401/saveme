package store

import (
	"errors"
	"os"
	"testing"

	"saveme/internal/core"
)

func TestImmutableSaveAndReload(t *testing.T) {
	const flag, enabled, reason, first, second, value, mode = "SAVEME_IMMUTABLE_TEST", "1", "requires CAP_LINUX_IMMUTABLE on a test filesystem", "example.com", "other.test", "changed", 0600
	if os.Getenv(flag) != enabled {
		t.Skip(reason)
	}
	protection := Immutable{}
	repo := New(t.TempDir(), protection)
	t.Cleanup(func() {
		for _, path := range repo.paths() {
			if err := protection.Set(path, false); err != nil && !errors.Is(err, os.ErrNotExist) {
				t.Error(err)
			}
		}
	})
	state := core.EmptyState()
	for _, domain := range []string{first, second} {
		state.Domains = []string{domain}
		if err := repo.Save(state); err != nil {
			t.Fatal(err)
		}
		for _, path := range repo.paths() {
			if err := os.WriteFile(path, []byte(value), mode); err == nil {
				t.Fatal(reason)
			}
			if err := os.Remove(path); err == nil {
				t.Fatal(reason)
			}
		}
		reloaded, err := New(repo.dir, protection).Load()
		if err != nil {
			t.Fatal(err)
		}
		if !core.DomainBlocked(domain, reloaded.Domains) {
			t.Fatal(reloaded)
		}
	}
}
