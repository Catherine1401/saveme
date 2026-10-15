package ui

import (
	"strings"
	"testing"
	"time"

	"saveme/internal/core"
)

func lockedModel(entry core.EntryStatus, now int64, width, height int) Model {
	model := readyModel().Appearance(true, true)
	model.screen, model.status = domainLocked, core.Response{OK: true, Now: now, Entries: []core.EntryStatus{entry}}
	model, _ = model.Update(Resize{Width: width, Height: height})
	return model
}

func TestCountdownShowsKindProgressAndUnlockTime(t *testing.T) {
	const width, height, hours, elapsed, domain = 100, 30, 12, 3, "example.com"
	const kind, progress, unlock = "Timed lock · 12h", "25% · 3h of 12h", "Unlocks at 21:00 today"
	const year, month, day, hour, zero = 2026, 10, 15, 9, 0
	start := time.Date(year, month, day, hour, zero, zero, zero, time.Local)
	now := start.Add(elapsed * time.Hour)
	entry := core.EntryStatus{Domain: domain, Active: true, Kind: core.Range, StartAt: start.Unix(), EndAt: start.Add(hours * time.Hour).Unix()}
	view := lockedModel(entry, now.Unix(), width, height).View()
	for _, want := range []string{kind, progress, unlock} {
		if !strings.Contains(view, want) {
			t.Fatal(want, view)
		}
	}
}

func TestCountdownShowsDaysAndLaterUnlockDay(t *testing.T) {
	const width, height, days, domain, dayLine, laterDay = 100, 30, 3, "example.com", "2 days", "Unlocks Sun 18 Oct at 08:00"
	const year, month, day, hour, dayHours, zero = 2026, 10, 15, 9, 24, 0
	now := time.Date(year, month, day, hour, zero, zero, zero, time.Local)
	end := now.Add(days*dayHours*time.Hour - time.Hour)
	entry := core.EntryStatus{Domain: domain, Active: true, Kind: core.Range, StartAt: now.Unix(), EndAt: end.Unix()}
	view := lockedModel(entry, now.Unix(), width, height).View()
	if !strings.Contains(view, dayLine) || !strings.Contains(view, unlockText(now.Unix(), end.Unix())) || !strings.Contains(view, laterDay) {
		t.Fatal(view)
	}
}

func TestDailyCountdownShowsWindowAndTomorrow(t *testing.T) {
	const width, height, domain, window, unlock = 100, 30, "example.com", "Daily schedule · 22:00 → 06:00", "Unlocks tomorrow at 06:00"
	const year, month, day, hour, span, zero = 2026, 10, 15, 22, 8, 0
	start := time.Date(year, month, day, hour, zero, zero, zero, time.Local)
	entry := core.EntryStatus{Domain: domain, Active: true, Kind: core.Daily, StartAt: start.Unix(), EndAt: start.Add(span * time.Hour).Unix()}
	view := lockedModel(entry, start.Add(time.Hour).Unix(), width, height).View()
	if !strings.Contains(view, window) || !strings.Contains(view, unlock) {
		t.Fatal(view)
	}
}

func TestCountdownExplainsMissingUnlockTime(t *testing.T) {
	const width, height, now, domain, unsynced, whole = 80, 24, 1000, "example.com", "Clock not synced", "covers the whole day"
	entry := core.EntryStatus{Domain: domain, Active: true, Kind: core.Daily}
	model := lockedModel(entry, now, width, height)
	if view := model.View(); !strings.Contains(view, whole) {
		t.Fatal(view)
	}
	model.status.OK, model.status.Error = false, core.ErrClockUnsync
	if view := model.View(); !strings.Contains(view, unsynced) {
		t.Fatal(view)
	}
}

func TestInfoLinesKeepTheirText(t *testing.T) {
	const line = "Daily schedule · 05:00 → 13:00 · 80% · 6h 24m of 8h"
	if plain := readyModel().Appearance(true, true).emphasize(line); plain != line {
		t.Fatal(plain)
	}
}
