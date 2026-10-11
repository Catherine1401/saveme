package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type appearance struct {
	plain  bool
	still  bool
	frame  int
	flash  int
	width  int
	height int
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
	const mint, violet, faint, red, warning, shimmer = "#78E2BF", "#BEA5F6", "#89909F", "#F38BA8", "#E9C46A", "#C8F8E8"
	const lightMint, lightViolet, lightFaint, lightRed, lightWarning, lightShimmer = "#176B50", "#7042AB", "#626978", "#A72A47", "#855D0E", "#247C61"
	color := lipgloss.AdaptiveColor{Light: lightFaint, Dark: faint}
	switch role {
	case toneText:
		const darkText, lightText = "#DEE3EB", "#212121"
		color = lipgloss.AdaptiveColor{Light: lightText, Dark: darkText}
	case toneAccent:
		color = lipgloss.AdaptiveColor{Light: lightMint, Dark: mint}
	case toneConfirm:
		color = lipgloss.AdaptiveColor{Light: lightViolet, Dark: violet}
	case toneError:
		color = lipgloss.AdaptiveColor{Light: lightRed, Dark: red}
	case toneWarning:
		color = lipgloss.AdaptiveColor{Light: lightWarning, Dark: warning}
	case toneShimmer:
		color = lipgloss.AdaptiveColor{Light: lightShimmer, Dark: shimmer}
	}
	return lipgloss.NewStyle().Foreground(color)
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
	const newline, empty, space, footer, escape, quit, ansi, code, block = "\n", "", " ", "Enter", "Esc", "q quit", "\x1b[", "Code:", "█"
	const zero = 0
	const arrows = "↑↓"
	lines := strings.Split(body, newline)
	for index, line := range lines {
		if line == empty || strings.Contains(line, ansi) {
			continue
		}
		color, bold := toneMuted, false
		if index == zero {
			color, bold = toneAccent, true
			if m.ready && !m.status.OK {
				color = toneWarning
			}
		}
		if strings.Contains(line, footer) || strings.Contains(line, escape) || strings.Contains(line, quit) || strings.Contains(line, arrows) {
			lines[index] = m.hint(line)
			continue
		}
		if m.screen == confirmation && (strings.Contains(line, block) || strings.Contains(line, code)) {
			color, bold = toneConfirm, true
			if m.appearance.flash > zero && !m.appearance.still {
				color = toneAccent
			}
		}
		if m.error != empty && strings.TrimSpace(line) == m.error {
			color, bold = toneError, true
		}
		lines[index] = m.ink(line, color, bold)
	}
	content := strings.Join(lines, newline)
	if m.appearance.width > zero {
		return lipgloss.NewStyle().Width(m.appearance.width).Render(content)
	}
	return space + strings.ReplaceAll(content, newline, newline+space)
}

func (m Model) selectedRow(line string) string {
	const newline = "\n"
	const dark, light = "#19342F", "#DAF5E9"
	if m.appearance.plain {
		return line
	}
	return m.style(toneAccent).Background(lipgloss.AdaptiveColor{Dark: dark, Light: light}).Bold(true).Render(strings.TrimSuffix(line, newline)) + newline
}

func (m Model) confirmCode() string {
	const zero, margin = 0, 4
	code := drawCode(m.pending.Code)
	if m.appearance.width > zero && lipgloss.Width(code)+margin > m.appearance.width {
		code = m.pending.Code
	}
	return m.ink(code, toneConfirm, true)
}

func (m Model) hint(line string) string {
	const space = " "
	const enter, escape, upDown, leftRight, quit = "Enter", "Esc", "↑↓", "←→", "q"
	const add, edit, remove, menuDigits, durationDigits = "a", "e", "d", "1–3", "1–9"
	const vimVertical, vimHorizontal = "j/k / ↑↓", "h/l / ←→"
	const verticalKeys, horizontalKeys, backKey, openKey = "j/k", "h/l", "h", "l"
	if strings.Contains(line, upDown) && m.screen != windowEdit {
		const menuNavigation, mainNavigation = "h back; l open; ", "l open; "
		navigation := menuNavigation
		if m.screen == mainScreen {
			navigation = mainNavigation
		}
		line = navigation + strings.TrimSpace(line)
	}
	line = strings.ReplaceAll(line, upDown, vimVertical)
	line = strings.ReplaceAll(line, leftRight, vimHorizontal)
	words := strings.Fields(line)
	for index, word := range words {
		color, bold := toneMuted, false
		switch word {
		case enter, escape, upDown, leftRight, quit, add, edit, remove, menuDigits, durationDigits, verticalKeys, horizontalKeys, backKey, openKey:
			color, bold = toneAccent, true
		}
		words[index] = m.ink(word, color, bold)
	}
	return space + strings.Join(words, space)
}
