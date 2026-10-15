package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type appearance struct {
	plain   bool
	still   bool
	frame   int
	flash   int
	width   int
	height  int
	palette palette
	clock   clockStyle
	effects []string
	warning bool
	now     time.Time
}

type animationFrame time.Time
type Resize struct{ Width, Height int }

func (m Model) Appearance(plain, still bool) Model {
	m.appearance.plain, m.appearance.still = plain, still
	return m
}

func nextAnimation() Cmd {
	const interval = 100 * time.Millisecond
	return Tick(interval, func(now time.Time) Msg { return animationFrame(now) })
}

func (m Model) animate() Cmd {
	if m.appearance.still {
		return nil
	}
	return nextAnimation()
}

type tone int

const (
	toneAccent tone = iota
	toneConfirm
	toneMuted
	toneError
	toneWarning
	toneShimmer
	toneText
)

func (m Model) style(role tone) lipgloss.Style {
	const none, first = "", 0
	theme := m.appearance.palette
	if theme.name == none {
		theme = palettes()[first]
	}
	pick := func(set shades) string {
		switch role {
		case toneText:
			return set.text
		case toneAccent:
			return set.accent
		case toneConfirm:
			return set.confirm
		case toneError:
			return set.fault
		case toneWarning:
			return set.warning
		case toneShimmer:
			return set.shimmer
		}
		return set.muted
	}
	dark, light := pick(theme.dark), pick(theme.light)
	if dark == none {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: light, Dark: dark})
}

func (m Model) ink(text string, role tone, bold bool) string {
	const newline = "\n"
	if m.appearance.plain {
		return text
	}
	content := strings.TrimRight(text, newline)
	return m.style(role).Bold(bold).Render(content) + text[len(content):]
}

func (m Model) spinner() string {
	const dot, half, two = "●", 4, 2
	const frames = "·✧✦✧"
	if m.appearance.still {
		return m.ink(dot, toneAccent, false)
	}
	color := toneAccent
	if m.appearance.frame%half >= half/two {
		color = toneShimmer
	}
	characters := []rune(frames)
	return m.ink(string(characters[m.appearance.frame%len(characters)]), color, false)
}

func (m Model) decorate(body string) string {
	const newline, empty, space, ansi, code, block = "\n", "", " ", "\x1b[", "Code:", "█"
	const zero = 0
	lines := strings.Split(body, newline)
	for index, line := range lines {
		if line == empty || strings.Contains(line, ansi) {
			continue
		}
		color, bold := toneMuted, false
		if index == zero {
			color, bold = toneAccent, true
		}
		if m.screen == confirmation && (strings.Contains(line, block) || strings.Contains(line, code)) {
			color, bold = toneConfirm, true
			if m.appearance.flash > zero && !m.appearance.still {
				color = toneAccent
			}
		}
		lines[index] = m.ink(line, color, bold)
	}
	content := strings.Join(lines, newline)
	if m.appearance.width > zero {
		return lipgloss.NewStyle().Width(m.contentWidth()).Render(content)
	}
	return space + strings.ReplaceAll(content, newline, newline+space)
}

func (m Model) selectedRow(line string) string {
	const newline = "\n"
	if m.appearance.plain {
		return line
	}
	return m.style(toneAccent).Bold(true).Render(strings.TrimSuffix(line, newline)) + newline
}

func (m Model) confirmCode() string {
	const zero, margin = 0, 4
	code := drawCode(m.pending.Code)
	if m.appearance.width > zero && lipgloss.Width(code)+margin > m.appearance.width {
		code = m.pending.Code
	}
	return m.ink(code, toneConfirm, true)
}
