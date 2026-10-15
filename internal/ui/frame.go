package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Một mục gợi ý phím ở footer; reason khác rỗng nghĩa là phím đang bị vô hiệu.
type keyHint struct {
	key    string
	label  string
	reason string
}

// Vẽ khung bo tròn kiểu Claude Code, co theo độ rộng terminal.
func (m Model) box(lines []string) string {
	const zero, one, horizontalBorder, newline = 0, 1, 2, "\n"
	style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(zero, one)
	if !m.appearance.plain {
		style = style.BorderForeground(m.style(toneMuted).GetForeground())
	}
	content := strings.Join(lines, newline)
	if m.appearance.width > zero && lipgloss.Width(style.Render(content)) > m.appearance.width {
		style = style.Width(m.appearance.width - horizontalBorder)
	}
	return style.Render(content) + newline
}

// Khung tiêu đề kèm dòng tóm tắt số domain và số domain đang khóa.
func (m Model) header() string {
	const title, summary, one, plural = "✻ saveme", "%d domain%s · %d locked", 1, "s"
	const zero, empty = 0, ""
	color := toneAccent
	if m.status.Active && m.appearance.flash > zero && !m.appearance.still {
		color = toneShimmer
	}
	lines := []string{m.ink(title, color, true)}
	if !m.ready {
		return m.box(lines)
	}
	locked, suffix := zero, plural
	for _, entry := range m.status.Entries {
		if entry.Active {
			locked++
		}
	}
	if len(m.status.Entries) == one {
		suffix = empty
	}
	tone := toneMuted
	if !m.status.OK {
		tone = toneWarning
	}
	return m.box(append(lines, m.ink(fmt.Sprintf(summary, len(m.status.Entries), suffix, locked), tone, false)))
}

// Footer gợi ý phím dạng "a add · e edit (locked) · q quit".
func (m Model) keys(hints ...keyHint) string {
	const separator, space, prefix, newline, empty, reasonFormat = " · ", " ", "\n ", "\n", "", " (%s)"
	parts := make([]string, len(hints))
	for index, hint := range hints {
		if hint.reason != empty {
			parts[index] = m.ink(hint.key+space+hint.label+fmt.Sprintf(reasonFormat, hint.reason), toneMuted, false)
			continue
		}
		parts[index] = m.ink(hint.key, toneAccent, true) + m.ink(space+hint.label, toneMuted, false)
	}
	return prefix + strings.Join(parts, m.ink(separator, toneMuted, false)) + newline
}

// Ô nhập bo tròn với dấu nhắc "> " và con trỏ cuối dòng.
func (m Model) inputBox(text string) string {
	const prompt, cursor = "> ", "_"
	return m.box([]string{m.ink(prompt, toneAccent, true) + m.ink(text+cursor, toneText, false)})
}
