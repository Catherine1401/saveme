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

func quotaState(used int64) core.State {
	const domain, limit, day, floor = "example.com", 600, "2026-10-09", 110
	state := core.EmptyState()
	state.MaxSeen = floor
	state.Entries = []core.Entry{{Domain: domain, Quota: &core.Quota{LimitSec: limit, UsedSec: used, Day: day}}}
	return state
}

func quotaResponse(state core.State, used int64) core.Response {
	const now, reset = 120, 86400
	quota := state.Entries[0].Quota
	remaining := max(quota.LimitSec-used, 0)
	entry := core.EntryStatus{Domain: state.Entries[0].Domain, Active: remaining == 0, Quota: &core.QuotaStatus{LimitSec: quota.LimitSec, UsedSec: used, RemainingSec: remaining, ResetAt: reset}}
	return core.Response{OK: true, Now: now, Active: entry.Active, Entries: []core.EntryStatus{entry}}
}

func TestPreservedUpdateKeepsQuotaAndRejectsLoss(t *testing.T) {
	const used, target = 300, 0
	before := quotaState(used)
	status := quotaResponse(before, used)
	if err := preservedUpdate(before, before.Clone(), status); err != nil {
		t.Fatal(err)
	}
	grown := before.Clone()
	grown.Entries[target].Quota.UsedSec = used + 30
	if err := preservedUpdate(before, grown, quotaResponse(grown, used+30)); err != nil {
		t.Fatal("usage growth rejected:", err)
	}
	cases := map[string]func(*core.State){
		"quota removed":  func(s *core.State) { s.Entries[target].Quota = nil },
		"usage reduced":  func(s *core.State) { s.Entries[target].Quota.UsedSec = used - 1 },
		"limit changed":  func(s *core.State) { s.Entries[target].Quota.LimitSec *= 2 },
		"day went back":  func(s *core.State) { s.Entries[target].Quota.Day = "2026-10-08" },
		"quota invented": func(s *core.State) { s.Entries[target].Quota = &core.Quota{LimitSec: 60} },
	}
	for name, change := range cases {
		after := before.Clone()
		change(&after)
		if name == "quota invented" {
			before2 := before.Clone()
			before2.Entries[target].Quota = nil
			if preservedUpdate(before2, after, status) == nil {
				t.Fatal(name)
			}
			continue
		}
		if preservedUpdate(before, after, status) == nil {
			t.Fatal(name)
		}
	}
}

func TestPreservedUpdateQuotaDayChangeNeedsTrustedClock(t *testing.T) {
	const used, target = 300, 0
	before := quotaState(used)
	after := before.Clone()
	after.Entries[target].Quota.Day, after.Entries[target].Quota.UsedSec = "2026-10-10", 0
	status := quotaResponse(after, 0)
	if err := preservedUpdate(before, after, status); err != nil {
		t.Fatal(err)
	}
	status.OK, status.Error = false, core.ErrClockUnsync
	if preservedUpdate(before, after, status) == nil {
		t.Fatal("day reset accepted with untrusted clock")
	}
}

func TestPreservedUpdateExhaustedQuotaMustStayActive(t *testing.T) {
	const limit, target = 600, 0
	before := quotaState(limit)
	status := quotaResponse(before, limit)
	if err := preservedUpdate(before, before.Clone(), status); err != nil {
		t.Fatal(err)
	}
	status.Entries[target].Active, status.Active = false, false
	if preservedUpdate(before, before.Clone(), status) == nil {
		t.Fatal("exhausted quota reported inactive")
	}
	if domains := activeDomains(before, quotaResponse(before, limit)); len(domains) != 1 || domains[0] != before.Entries[target].Domain {
		t.Fatal(domains)
	}
	if domains := activeDomains(quotaState(1), quotaResponse(quotaState(1), 1)); len(domains) != 0 {
		t.Fatal(domains)
	}
}
