package ui

import (
	"testing"

	"saveme/internal/core"
)

func TestVimListNavigationEnabledByDefault(t *testing.T) {
	const down, up, first, last, top, second, bottom = "j", "k", "g", "G", 0, 1, 2
	model := readyModel()
	model, _ = model.Update(runeKey(down))
	if model.selected != second {
		t.Fatal(model.selected)
	}
	model, _ = model.Update(runeKey(up))
	if model.selected != top {
		t.Fatal(model.selected)
	}
	model, _ = model.Update(runeKey(up))
	if model.selected != bottom {
		t.Fatal(model.selected)
	}
	model, _ = model.Update(runeKey(first))
	if model.selected != bottom {
		t.Fatal(model.selected)
	}
	model, _ = model.Update(runeKey(first))
	if model.selected != top {
		t.Fatal(model.selected)
	}
	model, _ = model.Update(runeKey(last))
	if model.selected != bottom {
		t.Fatal(model.selected)
	}
}

func TestVimTimeFieldsAndTextInput(t *testing.T) {
	const left, right, up, down, field, value, initial, letter, zero = "h", "l", "k", "j", 1, 1, 22, "j", 0
	model := readyModel()
	model.beginWindow()
	model, _ = model.Update(runeKey(right))
	model, _ = model.Update(runeKey(up))
	if model.field != field || model.fields[field] != value {
		t.Fatal(model.fields, model.field)
	}
	model, _ = model.Update(runeKey(down))
	model, _ = model.Update(runeKey(left))
	if model.field != zero || model.fields[field] != zero || model.fields[zero] != initial {
		t.Fatal(model.fields, model.field)
	}
	model.screen = domainEdit
	model, _ = model.Update(runeKey(letter))
	if model.domainText != letter {
		t.Fatal(model.domainText)
	}
	model.screen = confirmation
	model, action := model.Update(runeKey(up))
	if action != nil || model.error != "Use uppercase code letters and digits 2–9." {
		t.Fatal(model.error)
	}
}

func TestVimDisabledEmptyAndLockedGuards(t *testing.T) {
	const down, enter, last, selected, empty = "j", "enter", "G", 1, ""
	model := readyModel()
	model.status.State.Lock = &core.Lock{Kind: core.Daily}
	model, _ = model.Update(runeKey(down))
	model, action := model.Update(control(enter))
	if action != nil || model.screen != mainScreen || model.error == empty || model.selected != selected {
		t.Fatal(model, action)
	}
	model.screen, model.status.State.Domains = domainList, nil
	model, action = model.Update(runeKey(last))
	if action != nil || model.selected != 0 {
		t.Fatal(model.selected)
	}
	model.status.Active = true
	before := model.selected
	model, action = model.Update(runeKey(down))
	if action != nil || model.selected != before {
		t.Fatal(model, action)
	}
}

func TestVimHorizontalNavigationOpensAndReturns(t *testing.T) {
	const left, right, next, domain = "h", "l", "j", "example.com"
	model := readyModel()
	model, _ = model.Update(runeKey(left))
	if model.screen != mainScreen {
		t.Fatal(model.screen)
	}
	model, _ = model.Update(runeKey(next))
	model, _ = model.Update(runeKey(right))
	if model.screen != rangeList {
		t.Fatal(model.screen)
	}
	model, _ = model.Update(runeKey(left))
	if model.screen != mainScreen {
		t.Fatal(model.screen)
	}
	model.screen = domainList
	model, _ = model.Update(runeKey(right))
	if model.screen != domainEdit || model.domainText != domain {
		t.Fatal(model.screen, model.domainText)
	}
	model, _ = model.Update(runeKey(left))
	if model.screen != domainEdit || model.domainText != domain+left {
		t.Fatal(model.screen, model.domainText)
	}
	model.screen = lastWindow
	model, _ = model.Update(runeKey(left))
	if model.screen != windowList {
		t.Fatal(model.screen)
	}
}
