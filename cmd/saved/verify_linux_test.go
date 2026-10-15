package main

import (
	"testing"

	"saveme/internal/core"
)

func updateState() core.State {
	const domain, zone, start, end, floor, target = "example.com", "UTC", 100, 200, 110, 0
	state := core.EmptyState()
	state.Entries, state.MaxSeen = []core.Entry{{Domain: domain}}, floor
	state.Entries[target].Lock = &core.Lock{Kind: core.Range, Zone: zone, StartAt: start, EndAt: end}
	return state
}

func TestPreservedUpdateRejectsWeakenedCommitment(t *testing.T) {
	const now, end, earlier, other, target = 120, 200, 109, "other.test", 0
	const removed, shortened, domains, rewind, inactive, exposed = "removed", "shortened", "domains", "rewind", "inactive", "exposed"
	before := updateState()
	status := core.Response{OK: true, Now: now, Active: true, Entries: []core.EntryStatus{{Domain: before.Entries[target].Domain, Active: true, Kind: core.Range, Zone: before.Entries[target].Lock.Zone, StartAt: before.Entries[target].Lock.StartAt, EndAt: end}}}
	if err := preservedUpdate(before, before.Clone(), status); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		change func(*core.State, *core.Response)
	}{
		{removed, func(s *core.State, _ *core.Response) { s.Entries[target].Lock = nil }},
		{shortened, func(s *core.State, _ *core.Response) { s.Entries[target].Lock.EndAt = now }},
		{domains, func(s *core.State, _ *core.Response) { s.Entries[target].Domain = other }},
		{rewind, func(s *core.State, _ *core.Response) { s.MaxSeen = earlier }},
		{inactive, func(_ *core.State, r *core.Response) { r.Active = false }},
		{exposed, func(s *core.State, r *core.Response) { r.Entries = nil }},
	}
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			after, response := before.Clone(), status
			scenario.change(&after, &response)
			if err := preservedUpdate(before, after, response); err == nil {
				t.Fatal(scenario.name)
			}
		})
	}
}

func TestPreservedUpdateExpiresOnlyWithTrustedClock(t *testing.T) {
	const expired, waiting, end, minute, zone, target = 210, 120, 200, 60, "UTC", 0
	before := updateState()
	after := before.Clone()
	after.Entries[target].Lock = nil
	response := core.Response{OK: true, Now: expired, Entries: []core.EntryStatus{{Domain: after.Entries[target].Domain}}}
	if err := preservedUpdate(before, after, response); err != nil {
		t.Fatal(err)
	}
	response.OK, response.Error = false, core.ErrClockUnsync
	if err := preservedUpdate(before, after, response); err == nil {
		const message = "untrusted clock released commitment"
		t.Fatal(message)
	}
	response.Active = true
	response.Entries = []core.EntryStatus{{Domain: before.Entries[target].Domain, Active: true, Kind: core.Range, Zone: before.Entries[target].Lock.Zone}}
	if err := preservedUpdate(before, before.Clone(), response); err != nil {
		t.Fatal(err)
	}
	before.Entries[target].Lock = &core.Lock{Kind: core.Daily, Zone: zone, Windows: []core.Window{{Start: minute, End: end}}}
	response = core.Response{OK: true, Now: waiting, Entries: []core.EntryStatus{{Domain: before.Entries[target].Domain, Kind: core.Daily, Zone: zone, NextAt: int64(minute * minute), Windows: before.Entries[target].Lock.Windows}}}
	if err := preservedUpdate(before, before.Clone(), response); err != nil {
		t.Fatal(err)
	}
}

func TestPreservedUpdateRejectsDifferentInactiveDaemonState(t *testing.T) {
	const now, other, target = 120, "other.test", 0
	const accepted = "different daemon commitment accepted"
	before := updateState()
	before.Entries[target].Lock = nil
	observed := before.Clone()
	observed.Entries[target].Domain = other
	response := core.Response{OK: true, Now: now, Entries: []core.EntryStatus{{Domain: observed.Entries[target].Domain}}}
	if err := preservedUpdate(before, before.Clone(), response); err == nil {
		t.Fatal(accepted)
	}
}

func TestPreservedUpdateChecksEveryDomainIndependently(t *testing.T) {
	const other, now, end, first, second = "other.test", 210, 300, 0, 1
	before := updateState()
	before.Entries = append(before.Entries, core.Entry{Domain: other, Lock: before.Entries[first].Lock.Clone()})
	before.Entries[second].Lock.EndAt = end
	after := before.Clone()
	after.Entries[first].Lock = nil
	response := core.Response{OK: true, Now: now, Active: true, Entries: []core.EntryStatus{
		{Domain: after.Entries[first].Domain},
		{Domain: other, Active: true, Kind: core.Range, Zone: after.Entries[second].Lock.Zone, StartAt: after.Entries[second].Lock.StartAt, EndAt: end},
	}}
	if err := preservedUpdate(before, after, response); err != nil {
		t.Fatal(err)
	}
	after.Entries[second].Lock = nil
	if err := preservedUpdate(before, after, response); err == nil {
		const message = "unexpired second lock was removed"
		t.Fatal(message)
	}
}
