package ui

import (
	"strings"
	"testing"

	"saveme/internal/core"
)

func TestDisabledTimedLockVisibleBeforeAndAfterFocus(t *testing.T) {
	const disabled, reason, focus, narrow, first, none = "(disabled", "delete the daily schedule first", 1, 24, 0, ""
	for _, plain := range []bool{false, true} {
		model := readyModel().Appearance(plain, true)
		model.screen = domainMenu
		model.status.Entries[first].Kind = core.Daily
		for _, selected := range []int{first, focus} {
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
		model.status.Entries[first].Kind = none
		if strings.Contains(model.View(), disabled) {
			t.Fatal(model.View())
		}
	}
}

func TestDisabledTimedLockCannotActivateByShortcutOrEnter(t *testing.T) {
	const shortcut, enter, selected, first, none = "2", "enter", 1, 0, ""
	model := readyModel()
	model.screen = domainMenu
	model.status.Entries[first].Kind = core.Daily
	model.selected = selected
	for _, key := range []KeyMsg{runeKey(shortcut), control(enter)} {
		updated, action := model.Update(key)
		if action != nil || updated.screen != domainMenu || updated.error == none {
			t.Fatal(updated, action)
		}
	}
	model.status.Entries[first].Kind = none
	updated, _ := model.Update(control(enter))
	if updated.screen != rangeList {
		t.Fatal(updated.screen)
	}
}

func TestEmptyListGuidesFirstDomainAndFooterStaysShort(t *testing.T) {
	const welcome, help, locked, edit, domain = "No domains yet.", "? help", "(locked)", "e edit", "example.com"
	model := readyModel().Appearance(true, true)
	model.status.Entries = nil
	if view := model.View(); !strings.Contains(view, welcome) || !strings.Contains(view, help) {
		t.Fatal(view)
	}
	model.status.Entries = []core.EntryStatus{{Domain: domain, Active: true}}
	if view := model.View(); strings.Contains(view, locked) || strings.Contains(view, edit) {
		t.Fatal(view)
	}
}
