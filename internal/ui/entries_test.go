package ui

import (
	"strings"
	"testing"

	"saveme/internal/core"
)

func TestMainListShowsEachDomainLockState(t *testing.T) {
	const now, end, next = 1000, 1060, 1120
	const locked, upcoming, free = "a.com  ● locked · 00:01:00 left", "b.com  ◷ next lock in 00:02:00", "c.com  ○ unlocked"
	const first, second, third = "a.com", "b.com", "c.com"
	model := readyModel().Appearance(true, true)
	model.status.Now = now
	model.status.Entries = []core.EntryStatus{{Domain: first, Active: true, EndAt: end}, {Domain: second, Kind: core.Daily, NextAt: next}, {Domain: third}}
	view := model.View()
	for _, want := range []string{locked, upcoming, free} {
		if !strings.Contains(view, want) {
			t.Fatal(want, view)
		}
	}
}

func TestLockedDomainRejectsEditAndDelete(t *testing.T) {
	const edit, remove, empty = "e", "d", ""
	model := readyModel()
	model.status.Entries[0].Active = true
	for _, key := range []string{edit, remove} {
		updated, action := model.Update(runeKey(key))
		if action != nil || updated.screen != mainScreen || updated.error == empty {
			t.Fatal(key, updated.screen, updated.error)
		}
	}
}

func TestDeleteScheduledDomainRequiresConfirmation(t *testing.T) {
	const remove, enter, second, domain = "d", "enter", 1, "b.com"
	model := readyModel()
	model.status.Entries = append(model.status.Entries, core.EntryStatus{Domain: domain, Kind: core.Daily})
	model.selected = second
	model, action := model.Update(runeKey(remove))
	if action != nil || model.screen != domainDelete || model.entry != second || !strings.Contains(model.View(), domain) {
		t.Fatal(model.screen, model.entry)
	}
	model, action = model.Update(control(enter))
	if action == nil {
		t.Fatal(model)
	}
	action()
	request := model.client.(*recordingClient).last
	if request.Cmd != core.DeleteDomain || request.Index == nil || *request.Index != second || !request.Confirmed {
		t.Fatal(request)
	}
}

func TestDeleteUnscheduledDomainSubmitsDirectly(t *testing.T) {
	const remove = "d"
	model := readyModel()
	model, action := model.Update(runeKey(remove))
	if action == nil || model.screen != mainScreen {
		t.Fatal(model.screen)
	}
	action()
	request := model.client.(*recordingClient).last
	if request.Cmd != core.DeleteDomain || request.Confirmed {
		t.Fatal(request)
	}
}

func TestOpenDomainChoosesMenuOrCountdown(t *testing.T) {
	const enter = "enter"
	model := readyModel()
	updated, _ := model.Update(control(enter))
	if updated.screen != domainMenu {
		t.Fatal(updated.screen)
	}
	model.status.Entries[0].Active = true
	updated, _ = model.Update(control(enter))
	if updated.screen != domainLocked {
		t.Fatal(updated.screen)
	}
}

func TestFollowTracksOpenDomain(t *testing.T) {
	const domain = "example.com"
	status := func(entries []core.EntryStatus) reply {
		return reply{request: core.Request{Cmd: core.Status}, response: core.Response{OK: true, Entries: entries}}
	}
	model := readyModel()
	model.screen = domainLocked
	model, action := model.Update(status([]core.EntryStatus{{Domain: domain}}))
	if action == nil || model.screen != domainMenu {
		t.Fatal(model.screen)
	}
	model, action = model.Update(status(nil))
	if action == nil || model.screen != mainScreen {
		t.Fatal(model.screen)
	}
}

func TestAddDomainSucceedsWhileClockUnsynchronized(t *testing.T) {
	const domain = "example.com"
	entries := []core.EntryStatus{{Domain: domain}}
	unsync := core.Response{Error: core.ErrClockUnsync, Entries: entries}
	if !succeeded(reply{request: core.Request{Cmd: core.AddDomain}, response: unsync}) {
		t.Fatal(unsync)
	}
	if succeeded(reply{request: core.Request{Cmd: core.EditDomain}, response: unsync}) {
		t.Fatal(unsync)
	}
	if succeeded(reply{request: core.Request{Cmd: core.AddDomain}, response: core.Response{Error: core.ErrClockUnsync}}) {
		t.Fatal(unsync)
	}
}
