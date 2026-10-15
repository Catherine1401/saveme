package ui

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"saveme/internal/core"
)

func TestAnimationDoesNotCallDaemonOrChangeCountdown(t *testing.T) {
	const now, end, frames, empty = 1000, 1060, 12, ""
	model := readyModel()
	model.screen, model.status = domainLocked, core.Response{OK: true, Now: now, Entries: []core.EntryStatus{{Domain: "example.com", Active: true, EndAt: end}}}
	before := model.status
	client := model.client.(*recordingClient)
	for range frames {
		var action Cmd
		model, action = model.Update(animationFrame(time.Now()))
		if action == nil || !reflect.DeepEqual(model.status, before) || client.last.Cmd != core.Command(empty) {
			t.Fatal(model.status, client.last)
		}
	}
}

func TestRepeatedLockedStatusDoesNotClearScreen(t *testing.T) {
	const now, end = 1000, 1060
	model := readyModel()
	model.screen = domainMenu
	response := core.Response{OK: true, Now: now, Entries: []core.EntryStatus{{Domain: "example.com", Active: true, EndAt: end}}}
	model, action := model.Update(reply{request: core.Request{Cmd: core.Status}, response: response})
	if action == nil {
		t.Fatal(response)
	}
	model, action = model.Update(reply{request: core.Request{Cmd: core.Status}, response: response})
	if action != nil || model.screen != domainLocked {
		t.Fatal(model.status)
	}
}

func TestPlainStillAndNarrowConfirmation(t *testing.T) {
	const code, width, escape = "ABCD", 20, "\x1b["
	model := readyModel().Appearance(true, true)
	if model.animate() != nil || strings.Contains(model.View(), escape) {
		t.Fatal(model.View())
	}
	model.screen = confirmation
	model.pending.Code = code
	model, _ = model.Update(Resize{Width: width})
	if model.confirmCode() != code {
		t.Fatal(model.confirmCode())
	}
}

func TestStyledRowsPreserveLineBreaks(t *testing.T) {
	const text, newline, one = "Timed lock", "\n", 1
	model := readyModel()
	for _, plain := range []bool{false, true} {
		model = model.Appearance(plain, false)
		row := model.row(model.selected, text)
		if strings.Count(row, newline) != one || !strings.HasSuffix(row, newline) {
			t.Fatal(row)
		}
	}
}

func TestNarrowMenuWrapsWithoutLosingChoices(t *testing.T) {
	const width = 20
	model := readyModel()
	model.screen = rangeList
	model, _ = model.Update(Resize{Width: width})
	view := model.View()
	if lipgloss.Width(view) > width {
		t.Fatal(view)
	}
	for _, choice := range durationChoices() {
		if !strings.Contains(view, choice.Label) {
			t.Fatal(choice, view)
		}
	}
}
