package ui

import (
	"fmt"
	"strconv"
	"time"

	"saveme/internal/core"
)

// Cột trạng thái và cột thời gian của một domain trong bảng.
func (m Model) statusColumns(entry core.EntryStatus) (string, string) {
	const zero, locked, scheduledText, unlocked, none, left, starts = 0, "● locked", "◷ scheduled", "○ unlocked", "—", "%s left", "starts in %s"
	switch {
	case entry.Active && entry.EndAt != zero:
		return locked, fmt.Sprintf(left, countdown(entry.EndAt-m.status.Now))
	case entry.Active:
		return locked, none
	case scheduled(entry) && entry.NextAt != zero:
		return scheduledText, fmt.Sprintf(starts, countdown(entry.NextAt-m.status.Now))
	case scheduled(entry):
		return scheduledText, none
	}
	return unlocked, none
}

// Cắt chữ quá dài và thêm dấu "…".
func truncate(text string, width int) string {
	const ellipsis, one = "…", 1
	runes := []rune(text)
	if len(runes) <= width || width <= one {
		return text
	}
	return string(runes[:width-one]) + ellipsis
}

// Dòng đầu và số dòng của danh sách được vẽ, giữ dòng đang chọn trong tầm nhìn.
func (m Model) scrollWindow(count int) (int, int) {
	const zero, one, reserved, section, minimum, empty = 0, 1, 9, 2, 3, ""
	if m.appearance.height == zero {
		return zero, count
	}
	used := reserved
	if m.searching || m.filter != empty {
		used += section
	}
	if len(m.marked) > zero {
		used += section
	}
	rows := max(minimum, m.appearance.height-used)
	if rows >= count {
		return zero, count
	}
	top := zero
	if m.selected >= rows {
		top = m.selected - rows + one
	}
	return top, rows
}

// Bảng domain thẳng cột tên, trạng thái, thời gian; cuộn khi dài hơn màn hình.
func (m Model) domainTable(order []int, width int) string {
	const statusWidth, timeWidth, minimumDomain, gaps, columns = 11, 19, 8, 4, "%-*s  %-*s  %s"
	const markOn, markOff, empty, zero = "✓ ", "  ", "", 0
	const moreAbove, moreBelow, prefixFixed = "   ↑ %d more\n", "   ↓ %d more\n", 5
	entries := m.status.Entries
	prefix := prefixFixed + len(strconv.Itoa(len(order))) + len(markOff)
	domainWidth := zero
	for _, index := range order {
		domainWidth = max(domainWidth, len([]rune(entries[index].Domain)))
	}
	if width > zero {
		domainWidth = max(minimumDomain, min(domainWidth, width-prefix-statusWidth-timeWidth-gaps))
	}
	body := empty
	top, rows := m.scrollWindow(len(order))
	if top > zero {
		body += m.ink(fmt.Sprintf(moreAbove, top), toneMuted, false)
	}
	for position := top; position < top+rows; position++ {
		entry := entries[order[position]]
		mark := markOff
		if m.isMarked(entry.Domain) {
			mark = markOn
		}
		status, remaining := m.statusColumns(entry)
		body += m.row(position, mark+fmt.Sprintf(columns, domainWidth, truncate(entry.Domain, domainWidth), statusWidth, status, remaining))
	}
	if rest := len(order) - top - rows; rest > zero {
		body += m.ink(fmt.Sprintf(moreBelow, rest), toneMuted, false)
	}
	return body
}

// Nhãn một khung giờ: đang chạy thì còn bao lâu kết thúc, chưa chạy thì bao lâu nữa bắt đầu.
func (m Model) windowLabel(entry core.EntryStatus, window core.Window) string {
	const zero, activeFormat, waitingFormat = 0, "active · ends in %s", "waiting · starts in %s"
	now := time.Unix(m.status.Now, zero)
	lock := &core.Lock{Kind: core.Daily, Zone: entry.Zone, Windows: []core.Window{window}}
	remaining := countdown(lock.Transition(now) - m.status.Now)
	if lock.Active(now) {
		return fmt.Sprintf(activeFormat, remaining)
	}
	return fmt.Sprintf(waitingFormat, remaining)
}
