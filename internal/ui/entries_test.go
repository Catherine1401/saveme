package ui

import (
	"strings"
	"testing"

	"saveme/internal/core"
)

func TestMainListShowsEachDomainLockState(t *testing.T) {
	const now, end, next = 1000, 1060, 1120
	const locked, upcoming, free = "a.com  ● locked     00:01:00 left", "b.com  ◷ scheduled  starts in 00:02:00", "c.com  ○ unlocked   —"
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
	const edit, remove, empty, first = "e", "d", "", 0
	model := readyModel()
	model.status.Entries[first].Active = true
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
	model.selected = model.position(second)
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

func TestDeleteUnscheduledDomainAsksFirst(t *testing.T) {
	const remove, enter, question = "d", "enter", "Delete example.com?"
	model := readyModel()
	model, action := model.Update(runeKey(remove))
	if action != nil || model.screen != domainDelete || !strings.Contains(model.View(), question) {
		t.Fatal(model.screen)
	}
	model, action = model.Update(control(enter))
	if action == nil {
		t.Fatal(model)
	}
	action()
	request := model.client.(*recordingClient).last
	if request.Cmd != core.DeleteDomain || !request.Confirmed {
		t.Fatal(request)
	}
}

func TestOpenDomainChoosesMenuOrCountdown(t *testing.T) {
	const enter, first = "enter", 0
	model := readyModel()
	updated, _ := model.Update(control(enter))
	if updated.screen != domainMenu {
		t.Fatal(updated.screen)
	}
	model.status.Entries[first].Active = true
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

func TestListSortsLockedThenScheduledAndFilters(t *testing.T) {
	const slash, enter, escape, query = "/", "enter", "esc", "news"
	const free, daily, locked, news = "a.com", "b.com", "c.com", "news.site"
	const typedQ, backspace, empty, freeIndex, dailyIndex, lockedIndex, newsIndex, total = "q", "backspace", "", 0, 1, 2, 3, 4
	const firstRow, secondRow, thirdRow, fourthRow, oneMatch = 0, 1, 2, 3, 1
	model := readyModel()
	model.status.Entries = []core.EntryStatus{{Domain: free}, {Domain: daily, Kind: core.Daily}, {Domain: locked, Active: true}, {Domain: news}}
	if got := model.order(); got[firstRow] != lockedIndex || got[secondRow] != dailyIndex || got[thirdRow] != freeIndex || got[fourthRow] != newsIndex {
		t.Fatal(got)
	}
	model, _ = model.Update(runeKey(slash))
	for _, char := range query {
		model, _ = model.Update(runeKey(string(char)))
	}
	model, _ = model.Update(runeKey(typedQ))
	if model.screen != mainScreen || model.filter != query+typedQ {
		t.Fatal(model.filter)
	}
	model, _ = model.Update(control(backspace))
	model, _ = model.Update(control(enter))
	if got := model.order(); len(got) != oneMatch || got[firstRow] != newsIndex || model.searching {
		t.Fatal(got)
	}
	model, _ = model.Update(control(enter))
	if entry, _ := model.current(); entry.Domain != news || model.screen != domainMenu {
		t.Fatal(entry, model.screen)
	}
	model.screen = mainScreen
	model, _ = model.Update(runeKey(slash))
	model, _ = model.Update(control(escape))
	if model.filter != empty || len(model.order()) != total {
		t.Fatal(model.filter)
	}
}

func TestEscapeClearsKeptFilter(t *testing.T) {
	const slash, enter, escape, query, empty = "/", "enter", "esc", "news", ""
	const free, news, total = "a.com", "news.site", 2
	model := readyModel()
	model.status.Entries = []core.EntryStatus{{Domain: free}, {Domain: news}}
	model, _ = model.Update(runeKey(slash))
	for _, char := range query {
		model, _ = model.Update(runeKey(string(char)))
	}
	model, _ = model.Update(control(enter))
	model, _ = model.Update(control(escape))
	if model.filter != empty || model.searching || len(model.order()) != total {
		t.Fatal(model.filter, model.searching)
	}
}
