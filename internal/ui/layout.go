package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Bề rộng tối đa của nội dung; terminal rộng hơn thì căn giữa.
func (m Model) contentWidth() int {
	const maxWidth = 120
	return min(m.appearance.width, maxWidth)
}

// Tách dòng phím ở cuối mỗi màn để ghim xuống đáy.
func splitFooter(body string) (string, string) {
	const newline, none, empty = "\n", -1, ""
	trimmed := strings.TrimRight(body, newline)
	index := strings.LastIndex(trimmed, newline)
	if index == none {
		return trimmed, empty
	}
	return trimmed[:index+len(newline)], trimmed[index+len(newline):] + newline
}

// Dòng heading: tên, số domain, số domain đang khóa.
func (m Model) statusLine() string {
	const title, summary, separator, plural, empty, one, zero = "✻ saveme", "%d domain%s · %d locked", " · ", "s", "", 1, 0
	const newline, space = "\n", " "
	color := toneAccent
	if m.status.Active && m.appearance.flash > zero && !m.appearance.still {
		color = toneShimmer
	}
	line := space + m.ink(title, color, true)
	if m.ready {
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
		line += m.ink(separator+fmt.Sprintf(summary, len(m.status.Entries), suffix, locked), tone, false)
	}
	if m.appearance.width > zero {
		line = lipgloss.NewStyle().Width(m.contentWidth()).Render(line)
	}
	return line + newline + newline
}

// Vùng thông báo ngay trên footer: cảnh báo NTP, lỗi, trạng thái đang xử lý.
func (m Model) messages() string {
	const errorFormat, busy, space, empty = " ⎿ %s\n", " Working…\n", " ", ""
	text := empty
	if m.ready && m.usable() && m.screen != domainLocked {
		text += m.clockWarning()
	}
	if m.error != empty {
		text += m.ink(fmt.Sprintf(errorFormat, m.error), toneError, true)
	}
	if m.busy {
		text += space + m.spinner() + m.ink(busy, toneMuted, false)
	}
	return text
}

// Màn xác nhận và xóa hiện như hộp thoại bo tròn ở giữa.
func (m Model) modal() bool {
	return m.screen == domainDelete || m.screen == lastWindow || m.screen == confirmation
}

// Màn countdown đặt nội dung ở giữa màn hình.
func (m Model) centered() bool {
	return m.screen == domainLocked
}

// Ghép trang: dòng heading, nội dung, khoảng đệm, thông báo, footer ghim đáy.
func (m Model) compose(body string) string {
	const newline, zero, two = "\n", 0, 2
	content, footer := splitFooter(body)
	if m.centered() && m.appearance.width > zero {
		content = lipgloss.NewStyle().Width(m.contentWidth()).Align(lipgloss.Center).Render(strings.TrimRight(content, newline)) + newline
	} else {
		content = strings.TrimRight(m.decorate(content), newline) + newline
	}
	if m.modal() && m.appearance.width > zero {
		content = lipgloss.PlaceHorizontal(m.contentWidth(), lipgloss.Center, m.box(strings.Split(strings.TrimRight(content, newline), newline)))
		content = strings.TrimRight(content, newline) + newline
	}
	head, notes, foot := m.statusLine(), m.messages()+m.keycastLine(), strings.TrimRight(m.decorate(strings.TrimRight(footer, newline)), newline)
	page := head + content
	if m.appearance.height > zero {
		lines := func(text string) int { return strings.Count(text, newline) }
		used := lines(head) + lines(content) + lines(notes) + lipgloss.Height(foot)
		if fill := m.appearance.height - used; fill > zero {
			above := zero
			if m.modal() || m.centered() {
				above = fill / two
			}
			page = head + strings.Repeat(newline, above) + content + strings.Repeat(newline, fill-above)
		}
	}
	page += notes + foot
	if m.appearance.width > m.contentWidth() {
		return lipgloss.PlaceHorizontal(m.appearance.width, lipgloss.Center, page)
	}
	return page
}
