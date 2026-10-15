package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func TestClockGlyphsHaveEqualWidthsAndHeights(t *testing.T) {
	const symbols, colon, halfHeight, first, half = 11, 10, 3, 0, "half"
	for _, style := range clockStyles() {
		if style.glyphs == nil {
			continue
		}
		if len(style.glyphs) != symbols {
			t.Fatal(style.name)
		}
		height, width := len(style.glyphs[first]), lipgloss.Width(style.glyphs[first][first])
		for index, glyph := range style.glyphs {
			if len(glyph) != height {
				t.Fatal(style.name, index)
			}
			for _, line := range glyph {
				if index != colon && lipgloss.Width(line) != width {
					t.Fatal(style.name, index, line)
				}
			}
		}
		if style.name == half && height != halfHeight {
			t.Fatal(height)
		}
	}
}

func TestEveryClockRendersWithoutEscapesInPlainMode(t *testing.T) {
	const sample, escape, noProgress, empty = "01:54:21", "\x1b[", -1, ""
	for _, style := range clockStyles() {
		model := readyModel().Appearance(true, true).Configure(Settings{Clock: style.name}, nil)
		lines := model.clockLines(sample)
		if out := model.paintClock(lines, int64(time.Hour.Seconds()), noProgress); strings.Contains(out, escape) || out == empty {
			t.Fatal(style.name, out)
		}
	}
}

func TestBlinkHidesColonAndSlideMixesDigits(t *testing.T) {
	const sample, previous, colonRow, early, late = "01:54:21", "01:54:20", 1, 100, 700
	const year, month, day, hour, zero, block, blink, slide, newline = 2026, 10, 15, 9, 0, "block", "blink", "slide", "\n"
	base := time.Date(year, month, day, hour, zero, zero, zero, time.Local)
	model := readyModel().Configure(Settings{Clock: block, Effects: []string{blink}}, nil)
	model.appearance.now = base.Add(early * time.Millisecond)
	visible := model.clockLines(sample)[colonRow]
	model.appearance.now = base.Add(late * time.Millisecond)
	if hidden := model.clockLines(sample)[colonRow]; hidden == visible {
		t.Fatal(hidden)
	}
	model = model.Configure(Settings{Clock: block, Effects: []string{slide}}, nil)
	model.previousClock, model.clockText, model.changedAt = previous, sample, base
	model.appearance.now = base.Add(early * time.Millisecond)
	moving := strings.Join(model.clockLines(sample), newline)
	model.appearance.now = base.Add(late * time.Millisecond)
	if settled := strings.Join(model.clockLines(sample), newline); settled == moving {
		t.Fatal(settled)
	}
}

func TestMixBlendsHexAndRejectsAnsi(t *testing.T) {
	const black, white, middle, ansi = "#000000", "#ffffff", "#7f7f7f", "6"
	const half = 0.5
	if got, ok := mix(black, white, half); !ok || got != middle {
		t.Fatal(got)
	}
	if _, ok := mix(ansi, white, half); ok {
		t.Fatal(ansi)
	}
}

func TestLineClocksKeepEqualDigitHeights(t *testing.T) {
	const full, colon = "│║┃", 10
	const rounded, double, lcd, first, one = "rounded", "double", "lcd", 0, 1
	for _, name := range []string{rounded, double, lcd} {
		style, _ := findClock(name)
		for index, glyph := range style.glyphs {
			if index == colon {
				continue
			}
			if strings.ContainsAny(glyph[first], full) || strings.ContainsAny(glyph[len(glyph)-one], full) {
				t.Fatal(name, index, glyph)
			}
		}
	}
}

func TestSlideNeverPanicsOnClockSkewOrPreview(t *testing.T) {
	const sample, previous, current, early = "12:34:56", "01:54:20", "01:54:21", 200
	const year, month, day, hour, zero, slide, empty, rows = 2026, 10, 15, 9, 0, "slide", "", 3
	const before, half, done, after = -0.5, 0.5, 1, 1.5
	const a, b, c, x, y, z = "a", "b", "c", "x", "y", "z"
	base := time.Date(year, month, day, hour, zero, zero, zero, time.Local)
	for _, style := range clockStyles() {
		model := readyModel().Configure(Settings{Clock: style.name, Effects: []string{slide}}, nil)
		model.previousClock, model.clockText, model.changedAt = previous, current, base
		model.appearance.now = base.Add(-early * time.Millisecond)
		model.clockLines(current)
		model.appearance.now = base.Add(early * time.Millisecond)
		model.clockLines(current)
		if preview := strings.Join(model.clockLines(sample), empty); preview == empty {
			t.Fatal(style.name)
		}
	}
	for _, progress := range []float64{before, zero, half, done, after} {
		if got := slideGlyph([]string{a, b, c}, []string{x, y, z}, progress); len(got) != rows {
			t.Fatal(progress, got)
		}
	}
}
