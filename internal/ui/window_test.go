package ui

import (
	"testing"

	"saveme/internal/core"
)

func TestEditDailyWindowLoadsSelectionAndSubmitsReplacement(t *testing.T) {
	const minutes, start, end, edit, enter, selected = 60, 14, 15, "e", "enter", 1
	const staleError, empty = "Previous action was refused.", ""
	model := readyModel()
	model.screen, model.selected, model.error = windowList, selected, staleError
	model.status.State.Lock = &core.Lock{Kind: core.Daily, Windows: []core.Window{{Start: 10 * minutes, End: 11 * minutes}, {Start: start * minutes, End: end * minutes}}}
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
	if request.Cmd != core.EditWindow || request.Index == nil || *request.Index != selected || request.Start != "16:00" || request.End != "17:00" {
		t.Fatal(request)
	}
}

func TestEditDailyWindowAllowsOwnRangeButRejectsOtherOverlap(t *testing.T) {
	const minutes, edit, enter = 60, "e", "enter"
	model := readyModel()
	model.screen = windowList
	model.status.State.Lock = &core.Lock{Kind: core.Daily, Windows: []core.Window{{Start: 14 * minutes, End: 15 * minutes}, {Start: 16 * minutes, End: 17 * minutes}}}
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
