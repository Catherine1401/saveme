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
	const label, block, newline = "Locked: 01:01:01 remaining", "━", "\n"
	model := readyModel().Appearance(true, true)
	model.screen, model.status = domainLocked, core.Response{OK: true, Now: now, Entries: []core.EntryStatus{{Domain: "example.com", Active: true, EndAt: end}}}
	model, _ = model.Update(Resize{Width: width, Height: height})
	before := model.status
	view := model.View()
	if !strings.Contains(view, label) || !strings.Contains(view, block) || lipgloss.Width(view) != width || lipgloss.Height(view) != height {
		t.Fatal(view)
	}
	lines := strings.Split(view, newline)
	if strings.TrimSpace(lines[0]) != "" || strings.TrimSpace(lines[height-1]) != "" {
		t.Fatal(view)
	}
	model, _ = model.Update(Resize{Width: narrow, Height: short})
	view = model.View()
	if strings.Contains(view, block) || !strings.Contains(strings.Join(strings.Fields(view), " "), label) || !reflect.DeepEqual(model.status, before) {
		t.Fatal(view, model.status)
	}
}

func TestLargeCountdownHandlesDaysAndAllDigits(t *testing.T) {
	const digits, width, height, days, label = "0123456789:", 100, 30, 86400, "Locked: 1d 00:00:00 remaining"
	if !strings.Contains(drawClock(digits), "━") {
		t.Fatal(drawClock(digits))
	}
	model := readyModel()
	model.screen, model.status = domainLocked, core.Response{OK: true, Entries: []core.EntryStatus{{Domain: "example.com", Active: true, EndAt: days}}}
	model, _ = model.Update(Resize{Width: width, Height: height})
	if !strings.Contains(model.View(), label) {
		t.Fatal(model.View())
	}
}
