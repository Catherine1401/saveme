package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"saveme/internal/core"
)

func TestLockedCountdownCentersAndResizesWithoutChangingTime(t *testing.T) {
	const width, height, narrow, short, now, end = 80, 24, 24, 10, 1000, 4661
	const label, block, newline, heading, footer = "Locked: 01:01:01 remaining", "█", "\n", "✻ saveme", "q back"
	const domain, top, lastOffset, space = "example.com", 0, 1, " "
	model := readyModel().Appearance(true, true)
	model.screen, model.status = domainLocked, core.Response{OK: true, Now: now, Entries: []core.EntryStatus{{Domain: domain, Active: true, EndAt: end}}}
	model, _ = model.Update(Resize{Width: width, Height: height})
	before := model.status
	view := model.View()
	if strings.Contains(view, label) || !strings.Contains(view, block) || lipgloss.Width(view) != width || lipgloss.Height(view) != height {
		t.Fatal(view)
	}
	lines := strings.Split(view, newline)
	if !strings.Contains(lines[top], heading) || !strings.Contains(lines[height-lastOffset], footer) {
		t.Fatal(view)
	}
	model, _ = model.Update(Resize{Width: narrow, Height: short})
	view = model.View()
	if strings.Contains(view, block) || !strings.Contains(strings.Join(strings.Fields(view), space), label) || !reflect.DeepEqual(model.status, before) {
		t.Fatal(view, model.status)
	}
}

func TestLargeCountdownHandlesDaysAndAllDigits(t *testing.T) {
	const digits, width, height, days, label, dayLine = "0123456789:", 100, 30, 86400, "Locked: 1d 00:00:00 remaining", "1 day"
	const domain = "example.com"
	if !strings.Contains(drawClock(digits), "━") {
		t.Fatal(drawClock(digits))
	}
	model := readyModel()
	model.screen, model.status = domainLocked, core.Response{OK: true, Entries: []core.EntryStatus{{Domain: domain, Active: true, EndAt: days}}}
	model, _ = model.Update(Resize{Width: width, Height: height})
	if view := model.View(); strings.Contains(view, label) || !strings.Contains(view, dayLine) {
		t.Fatal(view)
	}
}
