package core

import "testing"

func TestStateValidatesEachEntryLock(t *testing.T) {
	const zone, first, second, start, end, legacyVersion = "UTC", "a.com", "b.com", 60, 120, 1
	const invalidLock, duplicate, oldVersion = "invalid entry lock accepted", "duplicate domain accepted", "old version accepted"
	daily := &Lock{Kind: Daily, Zone: zone, Windows: []Window{{Start: start, End: end}}}
	timed := &Lock{Kind: Range, Zone: zone, StartAt: start, EndAt: end}
	state := EmptyState()
	state.Entries = []Entry{{Domain: first, Lock: daily}, {Domain: second, Lock: timed}}
	if err := state.Validate(); err != nil {
		t.Fatal(err)
	}
	state.Entries[1].Lock = &Lock{Kind: Range, Zone: zone, StartAt: end, EndAt: start}
	if state.Validate() != ErrInvalid {
		t.Fatal(invalidLock)
	}
	state.Entries[1] = Entry{Domain: first}
	if state.Validate() != ErrInvalid {
		t.Fatal(duplicate)
	}
	legacy := EmptyState()
	legacy.Version = legacyVersion
	if legacy.Validate() != ErrInvalid {
		t.Fatal(oldVersion)
	}
}

func TestStateCloneCopiesLocks(t *testing.T) {
	const zone, domain, start, end, changed, shared = "UTC", "a.com", 60, 120, 0, "clone shares windows"
	state := EmptyState()
	state.Entries = []Entry{{Domain: domain, Lock: &Lock{Kind: Daily, Zone: zone, Windows: []Window{{Start: start, End: end}}}}}
	copy := state.Clone()
	copy.Entries[0].Lock.Windows[0].Start = changed
	if state.Entries[0].Lock.Windows[0].Start != start {
		t.Fatal(shared)
	}
}
