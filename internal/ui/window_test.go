package ui

import (
	"testing"

	"saveme/internal/core"
)

func TestEditDailyWindowLoadsSelectionAndSubmitsReplacement(t *testing.T) {
	const minutes, start, end, edit, enter, selected = 60, 14, 15, "e", "enter", 1
	const wantStart, wantEnd = "16:00", "17:00"
	const staleError, empty, domain, first, otherStart, otherEnd = "Previous action was refused.", "", "example.com", 0, 10, 11
	model := readyModel()
	model.screen, model.selected, model.error = windowList, selected, staleError
	model.status.Entries[first] = core.EntryStatus{Domain: domain, Kind: core.Daily, Windows: []core.Window{{Start: otherStart * minutes, End: otherEnd * minutes}, {Start: start * minutes, End: end * minutes}}}
	model, _ = model.Update(runeKey(edit))
	if model.error != empty || !model.editingWindow || model.screen != windowEdit || model.fields[0] != start || model.fields[2] != end {
		t.Fatal(model)
	}
	model.fields[0], model.fields[2] = 16, 17
	model, action := model.Update(control(enter))
	if action == nil {
		t.Fatal(model.error)
	}
	action()
	request := model.client.(*recordingClient).last
	if request.Cmd != core.EditWindow || request.Entry == nil || *request.Entry != first || request.Index == nil || *request.Index != selected || request.Start != wantStart || request.End != wantEnd {
		t.Fatal(request)
	}
}

func TestEditDailyWindowAllowsOwnRangeButRejectsOtherOverlap(t *testing.T) {
	const minutes, edit, enter, domain, first, firstStart, firstEnd, secondStart, secondEnd = 60, "e", "enter", "example.com", 0, 14, 15, 16, 17
	model := readyModel()
	model.screen = windowList
	model.status.Entries[first] = core.EntryStatus{Domain: domain, Kind: core.Daily, Windows: []core.Window{{Start: firstStart * minutes, End: firstEnd * minutes}, {Start: secondStart * minutes, End: secondEnd * minutes}}}
	model, _ = model.Update(runeKey(edit))
	if _, action := model.Update(control(enter)); action == nil {
		t.Fatal(model.error)
	}
	model.fields[2] = 17
	updated, action := model.Update(control(enter))
	if action != nil || updated.error == "" {
		t.Fatal(updated, action)
	}
}
