package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"saveme/internal/core"
)

func manyDomains(count int) []core.EntryStatus {
	const format = "site%02d.com"
	entries := make([]core.EntryStatus, count)
	for index := range entries {
		entries[index] = core.EntryStatus{Domain: fmt.Sprintf(format, index)}
	}
	return entries
}

func TestFooterIsPinnedToTheLastLine(t *testing.T) {
	const width, height, footer, newline, lastOffset = 80, 24, "a add · ↵ open", "\n", 1
	model := readyModel().Appearance(true, true)
	model, _ = model.Update(Resize{Width: width, Height: height})
	lines := strings.Split(strings.TrimRight(model.View(), newline), newline)
	if len(lines) != height || !strings.Contains(lines[len(lines)-lastOffset], footer) {
		t.Fatal(len(lines), lines[len(lines)-lastOffset])
	}
}

func TestLongListScrollsAroundSelection(t *testing.T) {
	const width, height, count, last, above, below, first, lastOffset = 80, 24, 40, "site39.com", "↑", "↓", "site00.com", 1
	model := readyModel().Appearance(true, true)
	model.status.Entries = manyDomains(count)
	model, _ = model.Update(Resize{Width: width, Height: height})
	view := model.View()
	if !strings.Contains(view, below) || strings.Contains(view, above) || strings.Contains(view, last) || lipgloss.Height(view) > height {
		t.Fatal(view)
	}
	model.selected = count - lastOffset
	view = model.View()
	if !strings.Contains(view, last) || !strings.Contains(view, above) || strings.Contains(view, first) {
		t.Fatal(view)
	}
}

func TestWideTerminalKeepsOneColumn(t *testing.T) {
	const wide, height, separator, header, columns = 120, 30, "│", "✻ saveme · 1 domain · 0 locked", "STATUS"
	model := readyModel().Appearance(true, true)
	model, _ = model.Update(Resize{Width: wide, Height: height})
	if view := model.View(); strings.Contains(view, separator) || strings.Contains(view, columns) || !strings.Contains(view, header) {
		t.Fatal(view)
	}
}

func TestLongDomainIsTruncatedToFit(t *testing.T) {
	const width, height, ellipsis, letter, suffix = 40, 20, "…", "a", ".com"
	domain := strings.Repeat(letter, width) + suffix
	model := readyModel().Appearance(true, true)
	model.status.Entries = []core.EntryStatus{{Domain: domain}}
	model, _ = model.Update(Resize{Width: width, Height: height})
	view := model.View()
	if !strings.Contains(view, ellipsis) || lipgloss.Width(view) > width {
		t.Fatal(view)
	}
}

func TestDeleteDialogIsBoxedAndContentIsCenteredOnWideTerminals(t *testing.T) {
	const width, height, corner, question, maxWidth, newline, empty, half = 200, 30, "╭", "Delete example.com?", 120, "\n", "", 2
	model := readyModel().Appearance(true, true)
	model.screen = domainDelete
	model, _ = model.Update(Resize{Width: width, Height: height})
	view := model.View()
	if !strings.Contains(view, corner) || !strings.Contains(view, question) || lipgloss.Width(view) != width {
		t.Fatal(view)
	}
	for _, line := range strings.Split(view, newline) {
		if trimmed := strings.TrimSpace(line); trimmed != empty && strings.Index(line, trimmed) < (width-maxWidth)/half {
			t.Fatal(line)
		}
	}
}

func TestEveryScreenEndsWithKeyHints(t *testing.T) {
	const width, height, newline, separator, back, lastOffset = 100, 30, "\n", " · ", "back", 1
	screens := []screen{mainScreen, domainMenu, domainDelete, helpScreen, domainEdit, rangeList, windowList, windowEdit, lastWindow}
	for _, current := range screens {
		model := readyModel().Appearance(true, true)
		model.screen = current
		model, _ = model.Update(Resize{Width: width, Height: height})
		lines := strings.Split(strings.TrimRight(model.View(), newline), newline)
		last := lines[len(lines)-lastOffset]
		if len(lines) != height || (!strings.Contains(last, separator) && !strings.Contains(last, back)) {
			t.Fatal(current, len(lines), last)
		}
	}
}

func TestNumberedRowsStayAlignedPastNine(t *testing.T) {
	const open, enter, effects, labelStart, missing, newline, dot = "s", "enter", 2, "[", -1, "\n", "."
	model := readyModel().Appearance(true, true)
	model, _ = model.Update(runeKey(open))
	model.selected = effects
	model, _ = model.Update(control(enter))
	column := missing
	for _, line := range strings.Split(model.View(), newline) {
		if index := strings.Index(line, labelStart); index > missing && strings.Contains(line, dot) {
			width := lipgloss.Width(line[:index])
			if column > missing && width != column {
				t.Fatal(line)
			}
			column = width
		}
	}
	if column == missing {
		t.Fatal(model.View())
	}
}
