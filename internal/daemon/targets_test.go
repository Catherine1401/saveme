package daemon

import (
	"slices"
	"testing"

	"saveme/internal/core"
)

func TestTargetsLockSeveralDomainsInOneConfirmation(t *testing.T) {
	const owner, first, second, third, duration = 1, 0, 1, 2, "1h"
	const youtube, reddit = "youtube.com", "reddit.com"
	const separate = "locks must be separate copies"
	f := newFixture(t)
	f.engine.Handle(core.Request{Cmd: core.AddDomain, Domain: youtube}, owner)
	f.engine.Handle(core.Request{Cmd: core.AddDomain, Domain: reddit}, owner)
	preview := f.engine.Handle(core.Request{Cmd: core.CreateRange, Entry: at(first), Targets: []int{third}, Duration: duration}, owner)
	if !preview.Pending || !slices.Equal(preview.Domains, []string{f.repo.state.Entries[first].Domain, reddit}) {
		t.Fatal(preview)
	}
	response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner)
	if !response.Entries[first].Active || response.Entries[second].Active || !response.Entries[third].Active {
		t.Fatal(response)
	}
	if f.repo.state.Entries[first].Lock == f.repo.state.Entries[third].Lock {
		t.Fatal(separate)
	}
}

func TestTargetsRejectInvalidLockedAndScheduledDomains(t *testing.T) {
	const owner, first, second, third, duration, start, end = 1, 0, 1, 2, "1h", "18:00", "19:00"
	const youtube, reddit = "youtube.com", "reddit.com"
	f := newFixture(t)
	f.engine.Handle(core.Request{Cmd: core.AddDomain, Domain: youtube}, owner)
	f.engine.Handle(core.Request{Cmd: core.AddDomain, Domain: reddit}, owner)
	preview := f.engine.Handle(core.Request{Cmd: core.CreateDaily, Entry: at(third), Windows: []core.InputWindow{{Start: start, End: end}}}, owner)
	f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner)
	cases := []struct {
		request core.Request
		want    core.ErrorCode
	}{
		{core.Request{Cmd: core.CreateRange, Entry: at(first), Targets: []int{first}, Duration: duration}, core.ErrInvalid},
		{core.Request{Cmd: core.CreateRange, Entry: at(first), Targets: []int{second, second}, Duration: duration}, core.ErrInvalid},
		{core.Request{Cmd: core.CreateRange, Entry: at(first), Targets: []int{len(f.repo.state.Entries)}, Duration: duration}, core.ErrInvalid},
		{core.Request{Cmd: core.CreateRange, Entry: at(first), Targets: []int{third}, Duration: duration}, core.ErrHasLock},
		{core.Request{Cmd: core.AddWindow, Entry: at(third), Targets: []int{second}, Start: end, End: start}, core.ErrInvalid},
	}
	for _, tc := range cases {
		if response := f.engine.Handle(tc.request, owner); response.Error != tc.want {
			t.Fatal(tc.request, response)
		}
	}
}

func TestTargetsRecheckDomainsAtConfirmation(t *testing.T) {
	const owner, first, second, duration, renamed = 1, 0, 1, "1h", "renamed.com"
	const youtube = "youtube.com"
	f := newFixture(t)
	f.engine.Handle(core.Request{Cmd: core.AddDomain, Domain: youtube}, owner)
	preview := f.engine.Handle(core.Request{Cmd: core.CreateRange, Entry: at(first), Targets: []int{second}, Duration: duration}, owner)
	f.engine.pending.Targets[first].Domain = renamed
	if response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner); response.Error != core.ErrInvalid || f.repo.state.Entries[first].Lock != nil {
		t.Fatal(response)
	}
}
