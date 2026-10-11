package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func clockPattern(char rune) string {
	const zero, one, colon = "0111010001101011000101110", "0010001100001000010001110", "0000000100000000010000000"
	const zeroChar, oneChar, colonChar = '0', '1', ':'
	switch char {
	case zeroChar:
		return zero
	case oneChar:
		return one
	case colonChar:
		return colon
	default:
		return codePattern(char)
	}
}

func drawClock(clock string) string {
	const width, height, zero, on, block, blank, gap, newline = 5, 5, 0, '1', "█", " ", " ", "\n"
	var output strings.Builder
	for row := zero; row < height; row++ {
		for _, char := range clock {
			pattern := clockPattern(char)
			for column := zero; column < width; column++ {
				pixel := blank
				if pattern[row*width+column] == on {
					pixel = block
				}
				output.WriteString(pixel)
			}
			output.WriteString(gap)
		}
		if row < height-1 {
			output.WriteString(newline)
		}
	}
	return output.String()
}

func (m Model) lockedView() string {
	const zero, minHeight, margin = 0, 14, 4
	const title, infinite, format, message, footer = "saveme", "Locked with no release time.", "Locked: %s remaining", "Website blocking is active.", "q quit"
	const newline, gap, separator = "\n", "\n\n", " "
	label, large := infinite, ""
	if m.status.EndAt != zero {
		remaining := countdown(m.status.EndAt - m.status.Now)
		label = fmt.Sprintf(format, remaining)
		parts := strings.Split(remaining, separator)
		clock := drawClock(parts[len(parts)-1])
		if m.appearance.width >= lipgloss.Width(clock)+margin && m.appearance.height >= minHeight {
			large = m.ink(clock, toneAccent, false) + gap
		}
	}
	content := m.ink(title, toneAccent, true) + gap + large + m.ink(label, toneText, true) + newline + m.ink(message, toneMuted, false) + gap + m.hint(footer)
	const empty, working = "", " Working…"
	if m.error != empty {
		content += gap + m.ink(m.error, toneError, true)
	}
	if m.busy {
		content += gap + m.spinner() + m.ink(working, toneMuted, false)
	}
	if m.appearance.width > zero && m.appearance.height > zero {
		content = lipgloss.NewStyle().Width(m.appearance.width).Align(lipgloss.Center).Render(content)
		return lipgloss.Place(m.appearance.width, m.appearance.height, lipgloss.Center, lipgloss.Center, content)
	}
	return content
}
