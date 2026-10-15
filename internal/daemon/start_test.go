package daemon

import (
	"testing"

	"saveme/internal/core"
)

func TestActiveStatusReportsLockStart(t *testing.T) {
	const owner, first, zero = 1, 0, 0
	f := newFixture(t)
	createRange(t, f)
	response := f.engine.Handle(core.Request{Cmd: core.Status}, owner)
	entry := response.Entries[first]
	if !entry.Active || entry.StartAt != f.clock.now.Unix() || entry.EndAt <= entry.StartAt {
		t.Fatal(entry)
	}
	f.clock.trusted = false
	if entry := f.engine.Handle(core.Request{Cmd: core.Status}, owner).Entries[first]; entry.StartAt != zero || entry.EndAt != zero {
		t.Fatal(entry)
	}
}
