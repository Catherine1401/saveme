package ui

import (
	"testing"

	"saveme/internal/core"
)

func TestQuitReturnsOneLevelOutsideInput(t *testing.T) {
	const quit, message, empty = "q", "Previous error", ""
	parents := []struct{ from, parent screen }{{domainMenu, mainScreen}, {domainLocked, mainScreen}, {domainDelete, mainScreen}, {windowList, domainMenu}, {rangeList, domainMenu}, {lastWindow, windowList}}
	for _, pair := range parents {
		current, parent := pair.from, pair.parent
		model := readyModel()
		model.screen, model.error = current, message
		updated, action := model.Update(runeKey(quit))
		if action != nil || updated.screen != parent || updated.error != empty {
			t.Fatal(current, updated.screen, updated.error)
		}
	}
}

func TestQuitRemainsTextOrIgnoredInInput(t *testing.T) {
	const quit, upperQuit, empty = "q", "Q", ""
	for _, current := range []screen{domainEdit, windowEdit, confirmation} {
		model := readyModel()
		model.screen = current
		updated, action := model.Update(runeKey(quit))
		if action != nil || updated.screen != current {
			t.Fatal(current, updated.screen)
		}
		if current == domainEdit && updated.domainText != quit {
			t.Fatal(updated.domainText)
		}
	}
	model := readyModel()
	model.screen = confirmation
	updated, action := model.Update(runeKey(upperQuit))
	if action == nil || updated.screen != confirmation || updated.codeInput != empty {
		t.Fatal(updated, action)
	}
	updated, _ = updated.Update(action())
	if updated.codeInput != upperQuit {
		t.Fatal(updated.codeInput)
	}
}

func TestQuitExitsMainWithoutMutatingState(t *testing.T) {
	const quit, none, first = "q", "", 0
	for _, active := range []bool{false, true} {
		model := readyModel()
		model.status.Entries[first].Active = active
		updated, action := model.Update(runeKey(quit))
		if action == nil || updated.screen != mainScreen {
			t.Fatal(updated, action)
		}
		if _, ok := action().(QuitMsg); !ok {
			t.Fatal(action())
		}
		if model.client.(*recordingClient).last.Cmd != core.Command(none) {
			t.Fatal(model.client)
		}
	}
}
