package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestClockDigitsHaveDistinctSegmentsAndStableWidth(t *testing.T) {
	const digits, height, width, block = "0123456789", 7, 5, "█"
	var previous []string
	for _, char := range digits {
		drawing := drawClock(string(char))
		if lipgloss.Height(drawing) != height || lipgloss.Width(drawing) != width || strings.Contains(drawing, block) {
			t.Fatal(char, drawing)
		}
		for _, other := range previous {
			if drawing == other {
				t.Fatal(char, drawing)
			}
		}
		previous = append(previous, drawing)
	}
}

func TestClockAllDigitsShareVisibleHeight(t *testing.T) {
	const digits, height, width, newline, empty = "0123456789", 7, 5, "\n", ""
	const first, last = 0, height - 1
	for _, char := range digits {
		drawing := drawClock(string(char))
		lines := strings.Split(drawing, newline)
		if len(lines) != height || strings.TrimSpace(lines[first]) == empty || strings.TrimSpace(lines[last]) == empty {
			t.Fatalf("digit %c has a shorter visible height:\n%s", char, drawing)
		}
		for _, line := range lines {
			if lipgloss.Width(line) != width {
				t.Fatal(char, line)
			}

		}
	}
}

func TestClockAndCodeUseHeavyStrokes(t *testing.T) {
	const digits, code, thin, block = "0123456789:", "S5B8", "─│┌┐└┘├┤", "█"
	const horizontal, vertical, dot, edge = "━", "┃", "●", "┏━━━┓"
	clock, confirmation := drawClock(digits), drawCode(code)
	for _, drawing := range []string{clock, confirmation} {
		if strings.ContainsAny(drawing, thin) || strings.Contains(drawing, block) {
			t.Fatal(drawing)
		}
	}
	if !strings.Contains(clock, horizontal) || !strings.Contains(clock, vertical) || !strings.Contains(clock, dot) || !strings.Contains(confirmation, edge) {
		t.Fatal(clock, confirmation)
	}
}

func TestDigitOneUsesAlignedEndCaps(t *testing.T) {
	const digit, newline, first, last, top, bottom = "1", "\n", 0, 6, "    ━", "    ━"
	lines := strings.Split(drawClock(digit), newline)
	if lines[first] != top || lines[last] != bottom {
		t.Fatal(lines)
	}
}

func TestConfirmationUsesRealCharactersInSeparateCells(t *testing.T) {
	const code, block, codeWidth, newline, one, narrow = "S5B8", "█", 26, "\n", 1, 20
	model := readyModel().Appearance(true, true)
	model.pending.Code = code
	drawing := model.confirmCode()
	if strings.Contains(drawing, block) || lipgloss.Width(drawing) != codeWidth {
		t.Fatal(drawing)
	}
	line := strings.Split(drawing, newline)[one]
	for _, char := range code {
		if !strings.Contains(line, string(char)) {
			t.Fatal(char, line)
		}
	}
	model, _ = model.Update(Resize{Width: narrow})
	if model.confirmCode() != code {
		t.Fatal(model.confirmCode())
	}
}
