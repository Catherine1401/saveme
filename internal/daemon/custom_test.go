package daemon

import (
	"testing"
	"time"

	"saveme/internal/core"
)

func TestCustomDurationsConfirmRestartAndExpire(t *testing.T) {
	const owner, target, zone, delay = 1, 0, "UTC", 5 * time.Second
	const minimum, mixed, last, maximum = "1m", "90m", "1439m", "1440m"
	for _, text := range []string{minimum, mixed, last, maximum} {
		t.Run(text, func(t *testing.T) {
			f := newFixture(t)
			preview := f.engine.Handle(core.Request{Cmd: core.CreateRange, Entry: at(target), Duration: text}, owner)
			if !preview.Pending || f.engine.state.Entries[target].Lock != nil || customStoredLock(f.repo.state) || f.enforcer.active {
				t.Fatal(preview, f.repo.state)
			}
			span, err := core.Duration(text)
			if err != nil {
				t.Fatal(err)
			}
			f.clock.now = f.clock.now.Add(delay)
			end := f.clock.now.Add(span)
			confirmed := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner)
			if !confirmed.OK || !confirmed.Active || f.repo.state.Entries[target].Lock.EndAt != end.Unix() {
				t.Fatal(confirmed, f.repo.state)
			}
			restarted, err := New(f.repo.state, f.clock, f.repo, f.enforcer, asciiNormalizer{}, zone)
			if err != nil {
				t.Fatal(err)
			}
			f.clock.now = end.Add(-time.Second)
			if err := restarted.Tick(); err != nil || !f.enforcer.active {
				t.Fatal(err)
			}
			f.clock.now = end
			f.clock.trusted = false
			if err := restarted.Tick(); err != nil || !f.enforcer.active || !customStoredLock(f.repo.state) {
				t.Fatal(err, f.repo.state)
			}
			f.clock.trusted = true
			if err := restarted.Tick(); err != nil || f.enforcer.active || customStoredLock(f.repo.state) {
				t.Fatal(err, f.repo.state)
			}
		})
	}
}

func TestCustomRefusalCancellationAndStorageFailure(t *testing.T) {
	const owner, target, duration, expiry = 1, 0, "90m", 60 * time.Second
	const zero, over, bad = "0m", "1441m", "1.5m"
	f := newFixture(t)
	request := core.Request{Cmd: core.CreateRange, Entry: at(target), Duration: duration}
	for _, text := range []string{zero, over, bad} {
		invalid := request
		invalid.Duration = text
		if response := f.engine.Handle(invalid, owner); response.Error != core.ErrInvalid || response.Pending || customStoredLock(f.repo.state) {
			t.Fatal(response)
		}
	}
	f.clock.trusted = false
	if response := f.engine.Handle(request, owner); response.Error != core.ErrClockUnsync {
		t.Fatal(response)
	}
	f.clock.trusted = true
	preview := f.engine.Handle(request, owner)
	f.engine.Handle(core.Request{Cmd: core.Cancel}, owner)
	if response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner); response.Error != core.ErrBadCode {
		t.Fatal(response)
	}
	preview = f.engine.Handle(request, owner)
	f.clock.now = f.clock.now.Add(expiry)
	if response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner); response.Error != core.ErrCodeExpired || customStoredLock(f.repo.state) {
		t.Fatal(response)
	}
	preview = f.engine.Handle(request, owner)
	f.repo.fail = true
	if response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner); response.Error != core.ErrStorage || customStoredLock(f.engine.state) || f.enforcer.active {
		t.Fatal(response)
	}
	f.repo.fail = false
	if response := f.engine.Handle(request, owner); response.Error != core.ErrStorage {
		t.Fatal(response)
	}
	f = newFixture(t)
	preview = f.engine.Handle(request, owner)
	if response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner); !response.OK {
		t.Fatal(response)
	}
	if response := f.engine.Handle(request, owner); response.Error != core.ErrLocked {
		t.Fatal(response)
	}
}

func TestCustomTargetsShareDeadline(t *testing.T) {
	const owner, first, second, domain, duration = 1, 0, 1, "other.example", "90m"
	f := newFixture(t)
	if response := f.engine.Handle(core.Request{Cmd: core.AddDomain, Domain: domain}, owner); !response.OK {
		t.Fatal(response)
	}
	preview := f.engine.Handle(core.Request{Cmd: core.CreateRange, Entry: at(first), Targets: []int{second}, Duration: duration}, owner)
	if !preview.Pending {
		t.Fatal(preview)
	}
	response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner)
	const minutes = 90
	end := f.clock.now.Add(minutes * time.Minute).Unix()
	if !response.OK || f.repo.state.Entries[first].Lock.EndAt != end || f.repo.state.Entries[second].Lock.EndAt != end {
		t.Fatal(response, f.repo.state)
	}
}

func customStoredLock(state core.State) bool {
	for _, entry := range state.Entries {
		if entry.Lock != nil {
			return true
		}
	}
	return false
}
