package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func effectModel(effects ...string) Model {
	const clock = "block"
	return readyModel().Appearance(true, false).Configure(Settings{Clock: clock, Effects: effects}, nil)
}

func TestShapeEffectsChangeGlyphs(t *testing.T) {
	const previous, current, hinge, early = "01:54:20", "01:54:21", "▁", 100
	const year, month, day, hour, zero, later, flip, glitchName, empty = 2026, 10, 15, 9, 0, 1000, "flip", "glitch", ""
	base := time.Date(year, month, day, hour, zero, zero, zero, time.Local)
	model := effectModel(flip)
	model.previousClock, model.clockText, model.changedAt = previous, current, base
	model.appearance.now = base.Add(early * time.Millisecond)
	if lines := strings.Join(model.clockLines(current), empty); !strings.Contains(lines, hinge) {
		t.Fatal(lines)
	}
	glitch := effectModel(glitchName)
	glitch.appearance.now = time.UnixMilli(zero)
	inside := strings.Join(glitch.clockLines(current), empty)
	glitch.appearance.now = time.UnixMilli(later)
	outside := strings.Join(glitch.clockLines(current), empty)
	if strings.ContainsAny(outside, glitchNoise()) || inside == outside && strings.ContainsAny(inside, glitchNoise()) {
		t.Fatal(inside, outside)
	}
}

func TestDecorationsDrawAroundTheClock(t *testing.T) {
	const sample, glow, stars, drops = "01:54:21", "░", "✦✧", "⠂⠄⡀"
	const neonName, sparkle, rain, seconds, idle, zero, limit, step, failure = "neon", "sparkle", "rain", 3600, -1, 0, 20000, 100, "no sparkle and rain frame"
	neon := effectModel(neonName)
	neon.appearance.still = true
	if out := neon.paintClock(neon.clockLines(sample), seconds, idle); !strings.Contains(out, glow) {
		t.Fatal(out)
	}
	found := false
	for millis := int64(zero); millis < limit && !found; millis += step {
		model := effectModel(sparkle, rain)
		model.appearance.now = time.UnixMilli(millis)
		out := model.paintClock(model.clockLines(sample), seconds, idle)
		found = strings.ContainsAny(out, stars) && strings.ContainsAny(out, drops)
	}
	if !found {
		t.Fatal(failure)
	}
}

func TestStillModeStopsMotionEffects(t *testing.T) {
	const sample = "01:54:21"
	const sparkle, rain, glitch, flip, seconds, idle, zero, limit, step, motion = "sparkle", "rain", "glitch", "flip", 3600, -1, 0, 6000, 100, "✦✧⠂⠄⡀▓▒▁"
	model := effectModel(sparkle, rain, glitch, flip)
	model.appearance.still = true
	for millis := int64(zero); millis < limit; millis += step {
		model.appearance.now = time.UnixMilli(millis)
		if out := model.paintClock(model.clockLines(sample), seconds, idle); strings.ContainsAny(out, motion) {
			t.Fatal(millis, out)
		}
	}
}

func TestAllEffectsWithEveryClockKeepEqualWidths(t *testing.T) {
	const sample, escape, newline = "01:54:21", "\x1b[", "\n"
	const zero, limit, step, previous, seconds, ratio, first = 0, 3000, 250, "01:54:20", 3600, 0.6, 0
	for _, style := range clockStyles() {
		for millis := int64(zero); millis < limit; millis += step {
			model := readyModel().Appearance(true, false).Configure(Settings{Clock: style.name, Effects: EffectNames(), Warning: true}, nil)
			model.previousClock, model.clockText, model.changedAt = previous, sample, time.UnixMilli(zero)
			model.appearance.now = time.UnixMilli(millis)
			out := model.paintClock(model.clockLines(sample), seconds, ratio)
			lines := strings.Split(out, newline)
			for _, line := range lines {
				if lipgloss.Width(line) != lipgloss.Width(lines[first]) || strings.Contains(line, escape) {
					t.Fatal(style.name, millis, out)
				}
			}
		}
	}
}
