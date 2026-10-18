package main

import (
	"reflect"
	"saveme/internal/core"
	"time"
)

func preservedUpdate(before, after core.State, status core.Response) error {
	const zero = 0
	if !status.OK && status.Error != core.ErrClockUnsync {
		return core.ErrEnforcement
	}
	if after.Version != before.Version || after.MaxSeen < before.MaxSeen || status.Now < after.MaxSeen || len(before.Entries) != len(after.Entries) || len(status.Entries) != len(after.Entries) {
		return core.ErrStorage
	}
	trusted, active := status.OK, false
	now := time.Unix(status.Now, zero)
	for index, prior := range before.Entries {
		entry, observed := after.Entries[index], status.Entries[index]
		if prior.Domain != entry.Domain || observed.Domain != entry.Domain {
			return core.ErrStorage
		}
		expired := trusted && prior.Lock != nil && prior.Lock.Kind == core.Range && status.Now >= prior.Lock.EndAt
		if !reflect.DeepEqual(prior.Lock.Clone(), entry.Lock.Clone()) && !(expired && entry.Lock == nil) {
			return core.ErrLocked
		}
		if err := preservedEntry(entry, observed, now, trusted); err != nil {
			return err
		}
		active = active || observed.Active
	}
	if status.Active != active {
		return core.ErrEnforcement
	}
	return nil
}

func preservedEntry(entry core.Entry, observed core.EntryStatus, now time.Time, trusted bool) error {
	expected := core.EntryStatus{Domain: entry.Domain}
	if entry.Lock != nil {
		expected.Kind, expected.Zone = entry.Lock.Kind, entry.Lock.Zone
		expected.Active = !trusted || entry.Lock.Active(now)
		if expected.Active && trusted {
			expected.StartAt, expected.EndAt = entry.Lock.WindowStart(now), entry.Lock.Transition(now)
		} else if !expected.Active {
			expected.NextAt, expected.Windows = entry.Lock.Transition(now), entry.Lock.Windows
		}
	}
	expected.Windows = append([]core.Window{}, expected.Windows...)
	observed.Windows = append([]core.Window{}, observed.Windows...)
	if !reflect.DeepEqual(expected, observed) {
		return core.ErrLocked
	}
	return nil
}

func activeDomains(state core.State, status core.Response) []string {
	var domains []string
	for _, entry := range state.Entries {
		const zero = 0
		if entry.Lock != nil && (!status.OK || entry.Lock.Active(time.Unix(status.Now, zero))) {
			domains = append(domains, entry.Domain)
		}
	}
	return domains
}
