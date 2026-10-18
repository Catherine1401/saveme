package main

import (
	"fmt"
	"slices"
	"strings"

	"saveme/internal/config"
	"saveme/internal/ui"
)

// Ghép tuỳ chọn đã lưu với cờ dòng lệnh; tên sai trong cờ thì báo lỗi kèm tên hợp lệ.
func sessionSettings(saved config.Preferences, theme, clock, effects string) (ui.Settings, error) {
	const empty, unknown, separator, list, none = "", "unknown %s %q; choose one of: %s", ", ", ",", "none"
	const themeKind, clockKind, effectKind = "theme", "clock", "clock effect"
	settings := ui.Settings{Theme: saved.Theme, Clock: saved.Clock, Effects: saved.Effects, Warning: saved.Warning == nil || *saved.Warning, AutoUpdates: saved.AutoUpdates == nil || *saved.AutoUpdates}
	if settings.Effects == nil && saved.Effect != empty {
		settings.Effects = []string{}
		if saved.Effect != none {
			settings.Effects = []string{saved.Effect}
		}
	}
	if effects != empty {
		settings.Effects = []string{}
		for _, effect := range strings.Split(effects, list) {
			if effect == none {
				continue
			}
			if !slices.Contains(ui.EffectNames(), effect) {
				return settings, fmt.Errorf(unknown, effectKind, effect, strings.Join(append(ui.EffectNames(), none), separator))
			}
			settings.Effects = append(settings.Effects, effect)
		}
	}
	overrides := []struct {
		value, kind string
		names       []string
		target      *string
	}{{theme, themeKind, ui.ThemeNames(), &settings.Theme}, {clock, clockKind, ui.ClockNames(), &settings.Clock}}
	for _, item := range overrides {
		if item.value == empty {
			continue
		}
		if !slices.Contains(item.names, item.value) {
			return settings, fmt.Errorf(unknown, item.kind, item.value, strings.Join(item.names, separator))
		}
		*item.target = item.value
	}
	return settings, nil
}

// Lưu tuỳ chọn giao diện vào file cấu hình của người dùng.
func saveSettings(settings ui.Settings) error {
	warning := settings.Warning
	automatic := settings.AutoUpdates
	effects := append([]string{}, settings.Effects...)
	return config.SavePreferences(config.Preferences{Theme: settings.Theme, Clock: settings.Clock, Effects: effects, Warning: &warning, AutoUpdates: &automatic})
}
