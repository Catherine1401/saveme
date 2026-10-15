package ui

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"saveme/internal/core"
)

func TestEveryPaletteIsCompleteAndUnique(t *testing.T) {
	const ansi, empty, count, first, fallback = "ansi", "", 16, 0, "catppuccin"
	const hexPattern, codePattern = `^#[0-9A-Fa-f]{6}$`, `^[0-9]{1,2}$`
	hex := regexp.MustCompile(hexPattern)
	code := regexp.MustCompile(codePattern)
	var seen []string
	themes := palettes()
	if len(themes) != count || themes[first].name != fallback {
		t.Fatal(len(themes), themes[first].name)
	}
	for _, item := range themes {
		if slices.Contains(seen, item.name) || item.label == empty {
			t.Fatal(item.name)
		}
		seen = append(seen, item.name)
		for _, set := range []shades{item.dark, item.light} {
			for _, value := range []string{set.accent, set.confirm, set.shimmer, set.muted, set.fault, set.warning} {
				if !hex.MatchString(value) && !(item.name == ansi && code.MatchString(value)) {
					t.Fatal(item.name, value)
				}
			}
			if set.text == empty && item.name != ansi {
				t.Fatal(item.name)
			}
		}
		if found, ok := findPalette(item.name); !ok || found.name != item.name {
			t.Fatal(item.name)
		}
	}
}

func TestSettingsPickerPreviewsSavesAndCancels(t *testing.T) {
	const open, enter, down, escape, nord, mint, savedMark, first, none, one = "s", "enter", "down", "esc", "nord", "mint", "✓ saved", 0, 0, 1
	var saved []Settings
	model := readyModel().Configure(Settings{Theme: nord, Warning: true}, func(next Settings) error { saved = append(saved, next); return nil })
	model, _ = model.Update(runeKey(open))
	model, _ = model.Update(control(enter))
	if model.screen != pickerScreen || palettes()[model.selected].name != nord || !strings.Contains(model.View(), savedMark) {
		t.Fatal(model.screen, model.selected)
	}
	model.selected = first
	model, _ = model.Update(control(down))
	if model.appearance.palette.name != mint {
		t.Fatal(model.appearance.palette.name)
	}
	model, _ = model.Update(control(escape))
	if model.screen != settingsScreen || model.appearance.palette.name != nord || len(saved) != none {
		t.Fatal(model.screen, model.appearance.palette.name, saved)
	}
	model, _ = model.Update(control(enter))
	model.selected = first
	model, _ = model.Update(control(down))
	model, _ = model.Update(control(enter))
	if model.screen != settingsScreen || model.settings.Theme != mint || len(saved) != one || saved[first].Theme != mint || !saved[first].Warning {
		t.Fatal(model.settings, saved)
	}
}

func TestSettingsToggleWarningAndPickClockAndEffect(t *testing.T) {
	const open, enter, down, block, warning, clock, effect, toggle, checked, blink, slide, first, one = "s", "enter", "down", "block", 3, 1, 2, " ", "[x] Sliding digits", "blink", "slide", 0, 1
	var saved Settings
	model := readyModel().Configure(Settings{Warning: true}, func(next Settings) error { saved = next; return nil })
	model, _ = model.Update(runeKey(open))
	model.selected = warning
	model, _ = model.Update(control(enter))
	if model.settings.Warning || saved.Warning {
		t.Fatal(model.settings)
	}
	model.selected = clock
	model, _ = model.Update(control(enter))
	model, _ = model.Update(control(down))
	model, _ = model.Update(control(enter))
	if saved.Clock != block || model.appearance.clock.name != block {
		t.Fatal(saved)
	}
	model.selected = effect
	model, _ = model.Update(control(enter))
	model, _ = model.Update(runeKey(toggle))
	model, _ = model.Update(control(down))
	model, _ = model.Update(runeKey(toggle))
	if !strings.Contains(model.View(), checked) || model.settings.Effects[first] != blink {
		t.Fatal(model.View())
	}
	model, _ = model.Update(control(enter))
	if len(saved.Effects) != one || saved.Effects[first] != slide || model.screen != settingsScreen {
		t.Fatal(saved)
	}
}

func TestSettingsSurviveStatusRefreshAndPlainMode(t *testing.T) {
	const open, enter, escape = "s", "enter", "\x1b["
	model := readyModel().Appearance(true, true)
	model.status.Entries = nil
	model, _ = model.Update(runeKey(open))
	model, _ = model.Update(control(enter))
	model, _ = model.Update(reply{request: core.Request{Cmd: core.Status}, response: core.Response{OK: true}})
	if model.screen != pickerScreen || strings.Contains(model.View(), escape) {
		t.Fatal(model.screen)
	}
}

func TestUnknownSettingsFallBackToDefaults(t *testing.T) {
	const unknown, theme, clock, none = "unknown-name", "catppuccin", "half", 0
	model := readyModel().Configure(Settings{Theme: unknown, Clock: unknown, Effects: []string{unknown}}, nil)
	if model.settings.Theme != theme || model.settings.Clock != clock || len(model.settings.Effects) != none {
		t.Fatal(model.settings)
	}
}
