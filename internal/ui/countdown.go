package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) lockedView() string {
	const zero, minHeight, margin = 0, 14, 4
	const title, infinite, format, message, footer = "saveme", "Locked with no release time.", "Locked: %s remaining", "%s is blocked.", "back"
	const newline, gap, separator, quitKey = "\n", "\n\n", " ", "q"
	entry, _ := m.current()
	label, large := infinite, ""
	if entry.EndAt != zero {
		remaining := countdown(entry.EndAt - m.status.Now)
		label = fmt.Sprintf(format, remaining)
		parts := strings.Split(remaining, separator)
		clock := drawClock(parts[len(parts)-1])
		if m.appearance.width >= lipgloss.Width(clock)+margin && m.appearance.height >= minHeight {
			large = m.ink(clock, toneAccent, true) + gap
		}
	}
	content := m.ink(title, toneAccent, true) + gap + large + m.ink(label, toneText, true) + newline + m.ink(fmt.Sprintf(message, entry.Domain), toneMuted, false) + gap + strings.TrimSpace(m.keys(keyHint{key: quitKey, label: footer}))
	const empty, working = "", " Working…"
	if m.error != empty {
		const errorPrefix = "⎿ "
		content += gap + m.ink(errorPrefix+m.error, toneError, true)
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
