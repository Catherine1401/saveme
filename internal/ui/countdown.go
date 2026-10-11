package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
			large = m.ink(clock, toneAccent, true) + gap
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
