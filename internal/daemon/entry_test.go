package daemon

import (
	"slices"
	"testing"
	"time"

	"saveme/internal/core"
)

func TestDomainsLockIndependently(t *testing.T) {
	const owner, first, second, other, duration = 1, 0, 1, "youtube.com", "1h"
	const start, end, extra, count = "18:00", "19:00", "reddit.com", 3
	f := newFixture(t)
	if response := f.engine.Handle(core.Request{Cmd: core.AddDomain, Domain: other}, owner); !response.OK {
		t.Fatal(response)
	}
	preview := f.engine.Handle(core.Request{Cmd: core.CreateRange, Entry: at(second), Duration: duration}, owner)
	if preview.Preview == nil || preview.Preview.Domain != other {
		t.Fatal(preview)
	}
	response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner)
	if !response.Entries[second].Active || response.Entries[first].Active || !slices.Equal(f.enforcer.blocked, []string{other}) {
		t.Fatal(response, f.enforcer.blocked)
	}
	if response := f.engine.Handle(core.Request{Cmd: core.AddDomain, Domain: extra}, owner); !response.OK || len(response.Entries) != count {
		t.Fatal(response)
	}
	preview = f.engine.Handle(core.Request{Cmd: core.CreateDaily, Entry: at(first), Windows: []core.InputWindow{{Start: start, End: end}}}, owner)
	if response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner); !response.OK || response.Entries[first].Active || response.Entries[first].NextAt == first {
		t.Fatal(response)
	}
	f.clock.now = f.clock.now.Add(time.Hour)
	if err := f.engine.Tick(); err != nil || f.enforcer.active || f.repo.state.Entries[second].Lock != nil || f.repo.state.Entries[first].Lock == nil {
		t.Fatal(err, f.repo.state)
	}
}

func TestUnlockedDomainEditKeepsScheduleAndDeleteNeedsConfirmation(t *testing.T) {
	const owner, target, start, end, renamed, empty = 1, 0, "18:00", "19:00", "renamed.com", 0
	f := newFixture(t)
	preview := f.engine.Handle(core.Request{Cmd: core.CreateDaily, Entry: at(target), Windows: []core.InputWindow{{Start: start, End: end}}}, owner)
	f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner)
	if response := f.engine.Handle(core.Request{Cmd: core.EditDomain, Index: at(target), Domain: renamed}, owner); !response.OK {
		t.Fatal(response)
	}
	if entry := f.repo.state.Entries[target]; entry.Domain != renamed || entry.Lock == nil {
		t.Fatal(entry)
	}
	if response := f.engine.Handle(core.Request{Cmd: core.DeleteDomain, Index: at(target)}, owner); response.Error != core.ErrInvalid {
		t.Fatal(response)
	}
	if response := f.engine.Handle(core.Request{Cmd: core.DeleteDomain, Index: at(target), Confirmed: true}, owner); !response.OK || len(f.repo.state.Entries) != empty {
		t.Fatal(response, f.repo.state)
	}
}

func TestPendingTargetsOneDomain(t *testing.T) {
	const owner, first, second, other, duration = 1, 0, 1, "youtube.com", "1h"
	f := newFixture(t)
	f.engine.Handle(core.Request{Cmd: core.AddDomain, Domain: other}, owner)
	createRange(t, f)
	if response := f.engine.Handle(core.Request{Cmd: core.CreateRange, Entry: at(first), Duration: duration}, owner); response.Error != core.ErrLocked {
		t.Fatal(response)
	}
	preview := f.engine.Handle(core.Request{Cmd: core.CreateRange, Entry: at(second), Duration: duration}, owner)
	if !preview.Pending {
		t.Fatal(preview)
	}
	if response := f.engine.Handle(core.Request{Cmd: core.Cancel}, owner); !response.OK {
		t.Fatal(response)
	}
	invalid := []core.Request{{Cmd: core.CreateRange, Duration: duration}, {Cmd: core.CreateRange, Entry: at(len(f.engine.state.Entries)), Duration: duration}}
	for _, request := range invalid {
		if response := f.engine.Handle(request, owner); response.Error != core.ErrInvalid {
			t.Fatal(response)
		}
	}
}
