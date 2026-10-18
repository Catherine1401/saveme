package ui

import (
	"os"
	"strings"
	"testing"

	"saveme/internal/update"
)

func TestConfigurePreservesAutoUpdatesWhenOtherSettingsAreSaved(t *testing.T) {
	const enabled = true
	const disabled = false
	for _, automatic := range []bool{enabled, disabled} {
		const enter = "enter"
		var saved Settings
		model := readyModel().Configure(Settings{AutoUpdates: automatic}, func(next Settings) error {
			saved = next
			return nil
		})
		if model.settings.AutoUpdates != automatic {
			const configuredError = "configured auto updates changed"
			t.Fatal(configuredError, automatic, model.settings)
		}
		model.screen, model.selected = settingsScreen, warningSetting
		model, _ = model.Update(control(enter))
		if model.settings.AutoUpdates != automatic || saved.AutoUpdates != automatic {
			const savedError = "saving another setting changed auto updates"
			t.Fatal(savedError, automatic, model.settings, saved)
		}
	}
}

func TestAutoUpdateSettingSavesAndPreservesUIChoices(t *testing.T) {
	const enter, theme, clock = "enter", "mint", "plain"
	model := readyModel()
	model.settings.AutoUpdates = true
	before := model.settings
	var saved Settings
	model.saveSettings = func(next Settings) error { saved = next; return nil }
	model.screen, model.selected = settingsScreen, autoUpdateSetting
	model, action := model.Update(control(enter))
	if action != nil || model.settings.AutoUpdates || saved.AutoUpdates || saved.Theme != before.Theme || saved.Clock != before.Clock {
		t.Fatal(model.settings, saved)
	}
	model.settings.Theme, model.settings.Clock = theme, clock
	model.saveSettings = func(Settings) error { return os.ErrPermission }
	model, _ = model.Update(control(enter))
	if model.settings.AutoUpdates {
		t.Fatal(model.settings)
	}
}

func TestUpdateNoticeIsPresentationOnlyAndHonorsDisabledSetting(t *testing.T) {
	const ready, version, notice, disabled = "ready", "v0.2.0", "next launch", "auto updates are off"
	model := readyModel()
	before := model.status
	model.settings.AutoUpdates = true
	model, action := model.Update(updateStatus{status: update.Status{Phase: ready, Available: version}})
	if action != nil || model.status.Now != before.Now || !strings.Contains(model.View(), notice) {
		t.Fatal(model.View())
	}
	model.settings.AutoUpdates = false
	if !strings.Contains(model.View(), disabled) {
		t.Fatal(model.View())
	}
}
