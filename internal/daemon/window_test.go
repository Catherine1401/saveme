package daemon

import (
	"testing"
	"time"

	"saveme/internal/core"
)

func scheduledFixture(t *testing.T) fixture {
	const owner, start, end, target = 1, "14:00", "15:00", 0
	t.Helper()
	f := newFixture(t)
	preview := f.engine.Handle(core.Request{Cmd: core.CreateDaily, Entry: at(target), Windows: []core.InputWindow{{Start: start, End: end}}}, owner)
	response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner)
	if !response.OK {
		t.Fatal(response)
	}
	return f
}

func TestEditWindowConfirmsPersistsAndRestarts(t *testing.T) {
	const owner, index, start, end, minute, hour, zone = 1, 0, "16:00", "17:00", 60, 16, "UTC"
	f := scheduledFixture(t)
	selected := index
	preview := f.engine.Handle(core.Request{Cmd: core.EditWindow, Entry: at(index), Index: &selected, Start: start, End: end}, owner)
	if !preview.Pending || f.engine.state.Entries[index].Lock.Windows[index].Start == hour*minute {
		t.Fatal(preview)
	}
	response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner)
	if !response.OK || f.repo.state.Entries[index].Lock.Windows[index].Start != hour*minute || len(f.repo.state.Entries[index].Lock.Windows) != 1 {
		t.Fatal(response, f.repo.state)
	}
	restarted, err := New(f.repo.state, f.clock, f.repo, f.enforcer, asciiNormalizer{}, zone)
	if err != nil || restarted.state.Entries[index].Lock.Windows[index] != f.engine.state.Entries[index].Lock.Windows[index] {
		t.Fatal(err)
	}
}

func TestEditWindowRejectsInvalidOverlapAndActiveConfirmation(t *testing.T) {
	const owner, index, badIndex, start, end = 1, 0, 9, "14:30", "16:00"
	const hour, minute, lifetime = 14, 60, 60 * time.Second
	f := scheduledFixture(t)
	f.engine.state.Entries[index].Lock.Windows = append(f.engine.state.Entries[index].Lock.Windows, core.Window{Start: 16 * minute, End: 17 * minute})
	selected, invalid := index, badIndex
	requests := []core.Request{
		{Cmd: core.EditWindow, Entry: at(index), Start: start, End: end},
		{Cmd: core.EditWindow, Entry: at(index), Index: &invalid, Start: start, End: end},
		{Cmd: core.EditWindow, Entry: at(index), Index: &selected, Start: "15:30", End: "16:30"},
	}
	for _, request := range requests {
		if response := f.engine.Handle(request, owner); response.OK || response.Pending {
			t.Fatal(response)
		}
	}
	f.clock.now = f.clock.now.Add(2*time.Hour - time.Second)
	preview := f.engine.Handle(core.Request{Cmd: core.EditWindow, Entry: at(index), Index: &selected, Start: start, End: end}, owner)
	if !preview.Pending {
		t.Fatal(preview)
	}
	f.clock.now = f.clock.now.Add(time.Second)
	response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner)
	if response.Error != core.ErrLocked || f.engine.state.Entries[index].Lock.Windows[index].Start != hour*minute {
		t.Fatal(response)
	}
	f.clock.now = f.clock.now.Add(lifetime)
	if response := f.engine.Handle(core.Request{Cmd: core.EditWindow, Entry: at(index), Index: &selected, Start: start, End: end}, owner); response.Error != core.ErrLocked {
		t.Fatal(response)
	}
}

func TestEditWindowCancelExpiryAndStorageFailurePreserveSchedule(t *testing.T) {
	const owner, index, start, end, expiry = 1, 0, "16:00", "17:00", 60 * time.Second
	f := scheduledFixture(t)
	selected := index
	request := core.Request{Cmd: core.EditWindow, Entry: at(index), Index: &selected, Start: start, End: end}
	original := f.engine.state.Entries[index].Lock.Windows[index]
	preview := f.engine.Handle(request, owner)
	f.engine.Handle(core.Request{Cmd: core.Cancel}, owner)
	if response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner); response.Error != core.ErrBadCode {
		t.Fatal(response)
	}
	preview = f.engine.Handle(request, owner)
	f.clock.now = f.clock.now.Add(expiry)
	if response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner); response.Error != core.ErrCodeExpired {
		t.Fatal(response)
	}
	f.clock.trusted = false
	if response := f.engine.Handle(request, owner); response.Error != core.ErrLocked {
		t.Fatal(response)
	}
	f.clock.trusted = true
	preview = f.engine.Handle(request, owner)
	f.repo.fail = true
	if response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner); response.Error != core.ErrStorage {
		t.Fatal(response)
	}
	if f.engine.state.Entries[index].Lock.Windows[index] != original || f.repo.state.Entries[index].Lock.Windows[index] != original {
		t.Fatal(f.engine.state, f.repo.state)
	}
}
