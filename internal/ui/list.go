package ui

import (
	"sort"
	"strings"
	"unicode"

	"saveme/internal/core"
)

// Thứ tự hiển thị: đang khóa, rồi có lịch, rồi chưa khóa; lọc theo chuỗi tìm kiếm.
func (m Model) order() []int {
	const locked, scheduledRank, free = 0, 1, 2
	rank := func(entry core.EntryStatus) int {
		switch {
		case entry.Active:
			return locked
		case scheduled(entry):
			return scheduledRank
		}
		return free
	}
	query := strings.ToLower(m.filter)
	var result []int
	for index, entry := range m.status.Entries {
		if strings.Contains(entry.Domain, query) {
			result = append(result, index)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return rank(m.status.Entries[result[i]]) < rank(m.status.Entries[result[j]])
	})
	return result
}

// Vị trí của domain trong danh sách đang hiển thị; 0 nếu không thấy.
func (m Model) position(entry int) int {
	const zero = 0
	for index, value := range m.order() {
		if value == entry {
			return index
		}
	}
	return zero
}

// Chỉ số domain của dòng đang chọn trong danh sách hiển thị.
func (m Model) selectedEntry() (int, bool) {
	const zero, none = 0, -1
	order := m.order()
	if m.selected < zero || m.selected >= len(order) {
		return none, false
	}
	return order[m.selected], true
}

// Đang gõ chữ: phím q/h là ký tự, không phải lệnh điều hướng.
func (m Model) typing() bool {
	return m.searching || m.screen == domainEdit || m.screen == windowEdit || m.screen == durationEdit || m.screen == confirmation
}

// Gõ chuỗi tìm kiếm; Enter giữ bộ lọc, Esc xóa bộ lọc.
func (m Model) searchKey(key KeyMsg) (Model, Cmd) {
	const enter, escape, backspace, zero, one, empty, maxInput = "enter", "esc", "backspace", 0, 1, "", 253
	switch key.String() {
	case enter:
		m.searching = false
	case escape:
		m.searching, m.filter = false, empty
	case backspace:
		runes := []rune(m.filter)
		if len(runes) > zero {
			m.filter = string(runes[:len(runes)-one])
		}
	default:
		text := string(key.Runes)
		if key.Type == KeyRunes && strings.IndexFunc(text, unicode.IsControl) < zero && len(m.filter)+len(text) <= maxInput {
			m.filter += text
		}
	}
	m.selected = zero
	return m, nil
}
