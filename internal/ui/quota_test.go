package ui

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"saveme/internal/core"
)

const quotaDomain = "example.com"

// skipWithoutQuota skips cases that need an enabled quota menu; the menu is
// disabled on platforms without a connection meter.
func skipWithoutQuota(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("quota is disabled on this platform")
	}
}

func quotaEntry(active bool, used, limit int64, endAt int64) core.EntryStatus {
	return core.EntryStatus{Domain: quotaDomain, Active: active, EndAt: endAt, Quota: &core.QuotaStatus{LimitSec: limit, UsedSec: used, RemainingSec: max(limit-used, 0), ResetAt: endAt}}
}

func TestMainListShowsQuotaLeftAndUsedUp(t *testing.T) {
	const now, limit, left, usedUp, opens = 1000, 1800, "▓▓▓░░░ 17m left", "● used up", "opens in 01:00:00"
	model := readyModel().Appearance(true, true)
	model.status.Now = now
	model.status.Entries = []core.EntryStatus{quotaEntry(false, 754, limit, 0)}
	if view := model.View(); !strings.Contains(view, left) {
		t.Fatal(view)
	}
	model.status.Entries = []core.EntryStatus{quotaEntry(true, limit, limit, now+3600)}
	view := model.View()
	if !strings.Contains(view, usedUp) || !strings.Contains(view, opens) {
		t.Fatal(view)
	}
	model.status.Entries = []core.EntryStatus{{Domain: quotaDomain}}
	if view := model.View(); strings.Contains(view, "quota") {
		t.Fatal("quota text without quota", view)
	}
}

func TestLockedViewDistinguishesQuotaFromSchedule(t *testing.T) {
	const width, height, now, limit, outOf, kind, unlocks = 100, 30, 1000, 1800, "example.com is out of quota.", "Daily quota · 30m used up", "Unlocks"
	view := lockedModel(quotaEntry(true, limit, limit, now+3600), now, width, height).View()
	for _, want := range []string{outOf, kind, unlocks} {
		if !strings.Contains(view, want) {
			t.Fatal(want, view)
		}
	}
	if strings.Contains(view, "is blocked") || strings.Contains(view, "Daily schedule") {
		t.Fatal(view)
	}
}

func openMenu(entry core.EntryStatus) Model {
	model := readyModel()
	model.status.Entries = []core.EntryStatus{entry}
	model.screen, model.entry = domainMenu, 0
	return model
}

func TestQuotaMenuSetsQuotaFromPresetAndCustom(t *testing.T) {
	skipWithoutQuota(t)
	const three, two, enter = "3", "2", "enter"
	model, _ := openMenu(core.EntryStatus{Domain: quotaDomain}).Update(runeKey(three))
	if model.screen != quotaList || !strings.Contains(model.View(), "Daily quota") {
		t.Fatal(model.screen)
	}
	preset, action := model.Update(runeKey(two))
	if action == nil {
		t.Fatal("preset did not submit")
	}
	action()
	request := preset.client.(*recordingClient).last
	if request.Cmd != core.SetQuota || request.Duration != "2h" || request.Entry == nil || *request.Entry != 0 || len(request.Targets) != 0 {
		t.Fatal(request)
	}
	custom := model
	custom.selected = len(quotaChoices())
	custom, _ = custom.Update(control(enter))
	if custom.screen != durationEdit || !custom.quotaMode {
		t.Fatal(custom.screen, custom.quotaMode)
	}
	custom, action = custom.Update(control(enter))
	action()
	if request = custom.client.(*recordingClient).last; request.Cmd != core.SetQuota || request.Duration != "60m" {
		t.Fatal(request)
	}
	back, _ := custom.back()
	if back.screen != quotaList || back.quotaMode {
		t.Fatal(back.screen, back.quotaMode)
	}
}

func TestQuotaChoicesStayWithinOneDay(t *testing.T) {
	const count, last = 6, "1 day"
	choices := quotaChoices()
	if len(choices) != count || choices[count-1].Label != last {
		t.Fatal(choices)
	}
}

func TestQuotaScreenRemovesQuotaOnlyWhenPresent(t *testing.T) {
	skipWithoutQuota(t)
	const three, remove, menuSummary, removedRow = "3", "d", "▓▓▓▓▓░ 1m of 10m used", "Remove quota"
	const idle, usedSec, limitSec, noEndAt, firstEntry = false, 60, 600, 0, 0
	const current, removeHint, submitFailed, noQuotaSubmit = "Current: ▓▓▓▓▓░ 1m of 10m used", "d remove quota", "remove did not submit", "remove submitted without a quota"
	menu := openMenu(quotaEntry(idle, usedSec, limitSec, noEndAt))
	if view := menu.View(); !strings.Contains(view, menuSummary) || strings.Contains(view, removedRow) {
		t.Fatal(view)
	}
	withQuota, _ := menu.Update(runeKey(three))
	if view := withQuota.View(); withQuota.screen != quotaList || !strings.Contains(view, current) || !strings.Contains(view, removeHint) {
		t.Fatal(withQuota.screen, withQuota.View())
	}
	model, action := withQuota.Update(runeKey(remove))
	if action == nil {
		t.Fatal(submitFailed)
	}
	action()
	if request := model.client.(*recordingClient).last; request.Cmd != core.ClearQuota || request.Entry == nil || *request.Entry != firstEntry {
		t.Fatal(request)
	}
	fresh, _ := openMenu(core.EntryStatus{Domain: quotaDomain}).Update(runeKey(three))
	if view := fresh.View(); strings.Contains(view, removeHint) {
		t.Fatal(view)
	}
	if _, action := fresh.Update(runeKey(remove)); action != nil {
		t.Fatal(noQuotaSubmit)
	}
}

func TestQuotaMenuIsDisabledWithReasonWhenClockUnsynced(t *testing.T) {
	skipWithoutQuota(t)
	const three, reason = "3", "disabled · clock not synced"
	model := openMenu(core.EntryStatus{Domain: quotaDomain})
	model.status.Error = core.ErrClockUnsync
	if view := model.View(); !strings.Contains(view, reason) {
		t.Fatal(view)
	}
	model, action := model.Update(runeKey(three))
	if action != nil || model.screen != domainMenu || !strings.Contains(model.error, "clock not synced") {
		t.Fatal(model.screen, model.error)
	}
}

func TestQuotaNeedsConfirmationCodeLikeLock(t *testing.T) {
	const code, limit = "AB23", 7200
	model := openMenu(core.EntryStatus{Domain: quotaDomain})
	prepared := core.Response{OK: true, Pending: true, Code: code, ExpiresIn: 60, Now: 1000, Preview: &core.Entry{Domain: quotaDomain, Quota: &core.Quota{LimitSec: limit}}}
	model, _ = model.Update(reply{request: core.Request{Cmd: core.SetQuota}, response: prepared})
	view := model.View()
	for _, want := range []string{"Confirm quota", "Limit: 2h per day", quotaDomain, "When used up: stays blocked until 00:00", "Code:"} {
		if !strings.Contains(view, want) {
			t.Fatal(want, view)
		}
	}
	if model.screen != confirmation {
		t.Fatal(model.screen)
	}
	for _, char := range code {
		model.lastChar = time.Now().Add(-time.Second)
		var commit Cmd
		model, commit = model.Update(runeKey(string(char)))
		model, _ = model.Update(commit())
	}
	model, action := model.Update(control("enter"))
	if action == nil {
		t.Fatal("confirm did not submit", model.codeInput, model.error)
	}
	action()
	if request := model.client.(*recordingClient).last; request.Cmd != core.Confirm || request.Code != code {
		t.Fatal(request)
	}
	model, _ = model.Update(reply{request: core.Request{Cmd: core.Confirm}, response: core.Response{OK: true, Now: 1000, Entries: []core.EntryStatus{quotaEntry(false, 0, limit, 0)}}})
	if model.screen != mainScreen || !strings.Contains(model.View(), "▓▓▓▓▓▓ 2h left") {
		t.Fatal(model.screen, model.View())
	}
}

func TestQuotaMenuLocksQuotaRowWhenUsedUp(t *testing.T) {
	const active, endAt = true, 4000
	const three, limit, disabledRow, lockedUntil = "3", 600, "disabled · locked until 00:00", "locked until 00:00"
	model := openMenu(quotaEntry(active, limit, limit, endAt))
	if view := model.View(); !strings.Contains(view, disabledRow) {
		t.Fatal(view)
	}
	model, action := model.Update(runeKey(three))
	if action != nil || model.screen != domainMenu || !strings.Contains(model.error, lockedUntil) {
		t.Fatal(model.screen, model.error)
	}
}

func TestQuotaListShowsUsedTodayAndFlagsReachedLevels(t *testing.T) {
	skipWithoutQuota(t)
	const idle, limitSec, noEndAt = false, 14400, 0
	const three, used, current, reached = "3", 7200, "Current:", "2 hours (blocks now)"
	model, _ := openMenu(quotaEntry(idle, used, limitSec, noEndAt)).Update(runeKey(three))
	view := model.View()
	for _, want := range []string{current, reached} {
		if !strings.Contains(view, want) {
			t.Fatal(want, view)
		}
	}
}

func TestQuotaBlockedErrorExplainsHowToFix(t *testing.T) {
	skipWithoutQuota(t)
	const three, fix = "3", "wait for time sync, then retry"
	model := openMenu(core.EntryStatus{Domain: quotaDomain})
	model.status.Error = core.ErrClockUnsync
	model, _ = model.Update(runeKey(three))
	if !strings.Contains(model.error, fix) {
		t.Fatal(model.error)
	}
}
