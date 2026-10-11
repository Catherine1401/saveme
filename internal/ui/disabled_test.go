package ui

import (
	"strings"
	"testing"

	"saveme/internal/core"
)

func TestDisabledTimedLockVisibleBeforeAndAfterFocus(t *testing.T) {
	const disabled, reason, focus, narrow = "[disabled]", "delete the daily schedule first", 1, 24
	for _, plain := range []bool{false, true} {
		model := readyModel().Appearance(plain, true)
		model.status.State.Lock = &core.Lock{Kind: core.Daily}
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
		model.status.State.Lock = nil
		if strings.Contains(model.View(), disabled) {
			t.Fatal(model.View())
		}
	}
}

func TestDisabledTimedLockCannotActivateByShortcutOrEnter(t *testing.T) {
	const shortcut, enter, selected = "2", "enter", 1
	model := readyModel()
	model.status.State.Lock = &core.Lock{Kind: core.Daily}
	model.selected = selected
	for _, key := range []KeyMsg{runeKey(shortcut), control(enter)} {
		updated, action := model.Update(key)
		if action != nil || updated.screen != mainScreen || updated.error == "" {
			t.Fatal(updated, action)
		}
	}
	model.status.State.Lock = nil
	updated, _ := model.Update(control(enter))
	if updated.screen != rangeList {
		t.Fatal(updated.screen)
	}
}

func TestUnavailableDomainActionsExplainTheirConditions(t *testing.T) {
	const noDomains, lastDomain, disabled = "no domains", "last domain is required", "[disabled:"
	model := readyModel().Appearance(true, true)
	model.screen = domainList
	model.status.State.Domains = nil
	if strings.Count(model.View(), noDomains) != 2 {
		t.Fatal(model.View())
	}
	model.status.State.Domains = []string{"example.com"}
	model.status.State.Lock = &core.Lock{Kind: core.Daily}
	if !strings.Contains(model.View(), lastDomain) {
		t.Fatal(model.View())
	}
	model.status.State.Lock = nil
	if strings.Contains(model.View(), disabled) {
		t.Fatal(model.View())
	}
}
