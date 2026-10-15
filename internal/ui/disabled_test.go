package ui

import (
	"strings"
	"testing"

	"saveme/internal/core"
)

func TestDisabledTimedLockVisibleBeforeAndAfterFocus(t *testing.T) {
	const disabled, reason, focus, narrow = "(disabled", "delete the daily schedule first", 1, 24
	for _, plain := range []bool{false, true} {
		model := readyModel().Appearance(plain, true)
		model.screen = domainMenu
		model.status.Entries[0].Kind = core.Daily
		for _, selected := range []int{0, focus} {
			model.selected = selected
			view := model.View()
			if !strings.Contains(view, disabled) || !strings.Contains(view, reason) {
				t.Fatal(view)
			}
		}
		model, _ = model.Update(Resize{Width: narrow})
		if !strings.Contains(model.View(), disabled) {
			t.Fatal(model.View())
		}
		model.status.Entries[0].Kind = ""
		if strings.Contains(model.View(), disabled) {
			t.Fatal(model.View())
		}
	}
}

func TestDisabledTimedLockCannotActivateByShortcutOrEnter(t *testing.T) {
	const shortcut, enter, selected = "2", "enter", 1
	model := readyModel()
	model.screen = domainMenu
	model.status.Entries[0].Kind = core.Daily
	model.selected = selected
	for _, key := range []KeyMsg{runeKey(shortcut), control(enter)} {
		updated, action := model.Update(key)
		if action != nil || updated.screen != domainMenu || updated.error == "" {
			t.Fatal(updated, action)
		}
	}
	model.status.Entries[0].Kind = ""
	updated, _ := model.Update(control(enter))
	if updated.screen != rangeList {
		t.Fatal(updated.screen)
	}
}

func TestUnavailableDomainActionsExplainTheirConditions(t *testing.T) {
	const noDomains, locked, disabled = "no domains", "(locked)", "(locked)"
	model := readyModel().Appearance(true, true)
	model.status.Entries = nil
	if strings.Count(model.View(), noDomains) != 2 {
		t.Fatal(model.View())
	}
	model.status.Entries = []core.EntryStatus{{Domain: "example.com", Active: true}}
	if strings.Count(model.View(), locked) != 2 {
		t.Fatal(model.View())
	}
	model.status.Entries[0].Active = false
	if strings.Contains(model.View(), disabled) {
		t.Fatal(model.View())
	}
}
