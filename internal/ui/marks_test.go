package ui

import (
	"slices"
	"strings"
	"testing"

	"saveme/internal/core"
)

func TestMarkedDomainsJoinTheTimedLock(t *testing.T) {
	const space, enter, timed, escape = " ", "enter", "2", "esc"
	const free, daily, other, marked, also = "a.com", "b.com", "c.com", "2 marked", "Also locks 1 marked"
	const first, second, third, none, empty = 0, 1, 2, 0, ""
	model := readyModel()
	model.status.Entries = []core.EntryStatus{{Domain: free}, {Domain: daily, Kind: core.Daily}, {Domain: other}}
	model.selected = model.position(second)
	model, _ = model.Update(runeKey(space))
	if model.error == empty || len(model.marked) != none {
		t.Fatal(model.error, model.marked)
	}
	model.selected = model.position(first)
	model, _ = model.Update(runeKey(space))
	model.selected = model.position(third)
	model, _ = model.Update(runeKey(space))
	if !strings.Contains(model.View(), marked) {
		t.Fatal(model.View())
	}
	model, _ = model.Update(control(enter))
	if model.entry != third || !strings.Contains(model.View(), also) {
		t.Fatal(model.entry, model.View())
	}
	model, _ = model.Update(runeKey(timed))
	model, action := model.Update(control(enter))
	if action == nil {
		t.Fatal(model)
	}
	action()
	request := model.client.(*recordingClient).last
	if request.Cmd != core.CreateRange || *request.Entry != third || !slices.Equal(request.Targets, []int{first}) {
		t.Fatal(request)
	}
	model.screen, model.busy = mainScreen, false
	model, _ = model.Update(control(escape))
	if len(model.marked) != none {
		t.Fatal(model.marked)
	}
}

func TestConfirmationListsEveryLockedDomain(t *testing.T) {
	const listed, code, first, second, end = "Domains: a.com, c.com", "K7MQ", "a.com", "c.com", 3600
	model := readyModel()
	model.screen = confirmation
	model.pending = core.Response{Code: code, Preview: &core.Entry{Domain: first, Lock: &core.Lock{Kind: core.Range, EndAt: end}}, Domains: []string{first, second}}
	if view := model.View(); !strings.Contains(view, listed) {
		t.Fatal(view)
	}
}
