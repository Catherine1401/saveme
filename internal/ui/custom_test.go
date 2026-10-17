package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"saveme/internal/core"
)

func TestCustomMenuNavigationAndCancel(t *testing.T) {
	const last, enter, escape, custom, hour, zero, empty = "G", "enter", "esc", "Custom time", 1, 0, ""
	model := readyModel().Appearance(true, true)
	model.screen = rangeList
	model, _ = model.Update(runeKey(last))
	if model.selected != len(durationChoices()) || !strings.Contains(model.View(), custom) {
		t.Fatal(model.View())
	}
	model, action := model.Update(control(enter))
	if action != nil || model.screen != durationEdit || model.duration.hours != hour || model.duration.minutes != zero {
		t.Fatal(model, action)
	}
	model, _ = model.Update(runeKey(last))
	if model.duration.field != zero || model.duration.digits != empty {
		t.Fatal(model.duration)
	}
	model, _ = model.Update(control(escape))
	if model.screen != rangeList || model.selected != len(durationChoices()) {
		t.Fatal(model.screen, model.selected)
	}
}

func TestCustomInputAndVimFields(t *testing.T) {
	const zero, one, two, four, three = "0", "1", "2", "4", "3"
	const right, up, left, quit, down, enter, expected = "l", "k", "h", "q", "j", "enter", "90m"
	model := readyModel().beginDuration()
	for _, key := range []string{zero, one, three, zero, right, up, down, left, quit} {
		model, _ = model.Update(runeKey(key))
	}
	const hour, minutes = 1, 30
	if model.screen != durationEdit || model.duration.hours != hour || model.duration.minutes != minutes {
		t.Fatal(model.duration)
	}
	model, action := model.Update(control(enter))
	if action == nil {
		t.Fatal(model)
	}
	action()
	if request := model.client.(*recordingClient).last; request.Cmd != core.CreateRange || request.Duration != expected {
		t.Fatal(request)
	}
	model = readyModel().beginDuration()
	for _, key := range []string{two, four, zero, one} {
		model, _ = model.Update(runeKey(key))
	}
	if _, action := model.Update(control(enter)); action != nil {
		t.Fatal(model.duration)
	}
}

func TestCustomSubmitBoundsAndRendering(t *testing.T) {
	const enter, empty, title, minimum, maximum = "enter", "", "Custom time", "00:01", "24:00"
	const minute, mixed, last, max = "1m", "90m", "1439m", "1440m"
	type scenario struct {
		hours, minutes int
		want           string
	}
	const zero, one, hour, half, lastHour, lastMinute, maxHour = 0, 1, 1, 30, 23, 59, 24
	cases := []scenario{{zero, one, minute}, {hour, half, mixed}, {lastHour, lastMinute, last}, {maxHour, zero, max}, {zero, zero, empty}, {maxHour, one, empty}}
	for _, item := range cases {
		model := readyModel().Appearance(true, true).beginDuration()
		model.duration.hours, model.duration.minutes = item.hours, item.minutes
		const width, height = 32, 24
		model, _ = model.Update(Resize{Width: width, Height: height})
		for _, text := range []string{title, minimum, maximum} {
			if !strings.Contains(model.View(), text) {
				t.Fatal(model.View())
			}
		}
		model, action := model.Update(control(enter))
		if item.want == empty {
			if action != nil || model.error == empty {
				t.Fatal(item, model.error)
			}
			continue
		}
		if action == nil {
			t.Fatal(item)
		}
		action()
		if request := model.client.(*recordingClient).last; request.Duration != item.want {
			t.Fatal(request)
		}
	}
}

func TestCustomConfirmationCancelPreservesInput(t *testing.T) {
	const minutes, now, ttl, code, enter, escape = 30, 1000, 60, "K7MQ", "enter", "esc"
	model := readyModel().beginDuration()
	model.duration.minutes = minutes
	client := model.client.(*recordingClient)
	client.response = core.Response{OK: true, Pending: true, Now: now, ExpiresIn: ttl, Code: code}
	model, action := model.Update(control(enter))
	model, _ = model.Update(action())
	if model.screen != confirmation || model.previous != durationEdit {
		t.Fatal(model.screen, model.previous)
	}
	model, _ = model.Update(control(escape))
	if model.screen != durationEdit || model.duration.minutes != minutes {
		t.Fatal(model.screen, model.duration)
	}
}

func TestCustomInvalidDigitsAndPaste(t *testing.T) {
	const two, nine, four, six, zero, escapeCode = "2", "9", "4", "6", "0", "\x1b["
	const maxHour, sixMinutes, empty, width, height = 24, 6, "", 32, 24
	model := readyModel().beginDuration()
	for _, key := range []string{two, nine, four, six, zero} {
		model, _ = model.Update(runeKey(key))
	}
	if model.duration.hours != maxHour || model.duration.minutes != sixMinutes || model.error == empty {
		t.Fatal(model.duration, model.error)
	}
	paste := runeKey(two)
	paste.Paste = true
	before := model.duration
	model, _ = model.Update(paste)
	if model.duration != before {
		t.Fatal(model.duration, before)
	}
	model, _ = model.Update(Resize{Width: width, Height: height})
	for _, plain := range []bool{false, true} {
		view := model.Appearance(plain, true).View()
		if lipgloss.Width(view) > width || (plain && strings.Contains(view, escapeCode)) {
			t.Fatal(view)
		}
	}
}

func TestCustomSubmitsMarkedTargets(t *testing.T) {
	const first, second, enter, primary, extra = 0, 1, "enter", "example.com", "other.example"
	model := readyModel().beginDuration()
	model.status.Entries = []core.EntryStatus{{Domain: primary}, {Domain: extra}}
	model.marked = []string{primary, extra}
	model, action := model.Update(control(enter))
	if action == nil {
		t.Fatal(model)
	}
	action()
	request := model.client.(*recordingClient).last
	const count = 1
	if request.Entry == nil || *request.Entry != first || len(request.Targets) != count || request.Targets[first] != second {
		t.Fatal(request)
	}
}
