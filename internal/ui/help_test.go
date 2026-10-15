package ui

import (
	"strings"
	"testing"
	"time"

	"saveme/internal/core"
)

func TestHelpScreenOpensAndReturns(t *testing.T) {
	const help, quit, keys, rule = "?", "q", "Keys", "facebook.com blocks only itself"
	model := readyModel()
	model, _ = model.Update(runeKey(help))
	if model.screen != helpScreen || !strings.Contains(model.View(), keys) || !strings.Contains(model.View(), rule) {
		t.Fatal(model.View())
	}
	model, action := model.Update(runeKey(quit))
	if action != nil || model.screen != mainScreen {
		t.Fatal(model.screen)
	}
}

func TestZeroSelectsTenthDuration(t *testing.T) {
	const zero, thirtyDays = "0", "30d"
	model := readyModel()
	model.screen = rangeList
	model, action := model.Update(runeKey(zero))
	if action == nil {
		t.Fatal(model)
	}
	action()
	if request := model.client.(*recordingClient).last; request.Duration != thirtyDays {
		t.Fatal(request)
	}
}

func TestActionErrorsExpireButDaemonFaultsStay(t *testing.T) {
	const message, fault, wait, epoch, empty = "Invalid domain.", "storage failed", 6 * time.Second, 0, ""
	start := time.Unix(epoch, epoch)
	model := readyModel()
	model.error = message
	model = model.expireError(start)
	model = model.expireError(start.Add(wait))
	if model.error != empty {
		t.Fatal(model.error)
	}
	model.status.OK, model.status.Message, model.error = false, fault, fault
	model = model.expireError(start)
	model = model.expireError(start.Add(wait))
	if model.error != fault {
		t.Fatal(model.error)
	}
}

func TestClockWarningReplacesErrorLine(t *testing.T) {
	const warning, message, empty = "Clock not synced", "Clock not synchronized with NTP by systemd.", ""
	model := readyModel()
	model, _ = model.Update(reply{request: core.Request{Cmd: core.Status}, response: core.Response{Error: core.ErrClockUnsync, Message: message, Entries: model.status.Entries}})
	if view := model.View(); !strings.Contains(view, warning) || model.error != empty {
		t.Fatal(view)
	}
}

func TestPresetMenuFillsEditorAndScheduleShowsNextChange(t *testing.T) {
	const enter, daily, work, workStart, workEnd, minutes, now, zero = "enter", "1", "2", 9, 17, 60, 0, 0
	const waiting, domain, zone, span, bar, quit, midnight = "waiting · starts in 09:00:00", "example.com", "UTC", "8h", "━", "q", "crosses midnight"
	model := readyModel()
	model, _ = model.Update(control(enter))
	model, _ = model.Update(runeKey(daily))
	if model.screen != presetList || !strings.Contains(model.View(), midnight) {
		t.Fatal(model.screen, model.View())
	}
	model, _ = model.Update(runeKey(work))
	view := model.View()
	if model.screen != windowEdit || model.fields != [timeFieldCount]int{workStart, zero, workEnd, zero} || !strings.Contains(view, span) || !strings.Contains(view, bar) {
		t.Fatal(model.fields, view)
	}
	model.status.Now = now
	model.status.Entries[zero] = core.EntryStatus{Domain: domain, Kind: core.Daily, Zone: zone, Windows: []core.Window{{Start: workStart * minutes, End: workEnd * minutes}}}
	model.screen = windowList
	if view := model.View(); !strings.Contains(view, waiting) {
		t.Fatal(view)
	}
	model.screen = presetList
	if model, _ = model.Update(runeKey(quit)); model.screen != windowList {
		t.Fatal(model.screen)
	}
}

func TestTimelineMarksLockedHalfHours(t *testing.T) {
	const night, morning, hour, filled, cells = 22, 6, 60, "━", 16
	model := readyModel().Appearance(true, true)
	if got := strings.Count(model.timeline(night*hour, morning*hour), filled); got != cells {
		t.Fatal(got)
	}
}

func TestHelpStaysOpenOnEmptyListAfterStatusRefresh(t *testing.T) {
	const help = "?"
	model := readyModel()
	model.status.Entries = nil
	model, _ = model.Update(runeKey(help))
	model, _ = model.Update(reply{request: core.Request{Cmd: core.Status}, response: core.Response{OK: true}})
	if model.screen != helpScreen {
		t.Fatal(model.screen)
	}
}
