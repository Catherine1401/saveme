package ui

import "slices"

// Bật/tắt đánh dấu domain để khóa chung; chỉ domain chưa khóa và chưa có lịch.
func (m Model) toggleMark() Model {
	const onlyFree = "Only unlocked domains without a schedule can be locked together."
	const first, next = 0, 1
	index, ok := m.selectedEntry()
	if !ok {
		return m
	}
	entry := m.status.Entries[index]
	if position := slices.Index(m.marked, entry.Domain); position >= first {
		m.marked = slices.Delete(slices.Clone(m.marked), position, position+next)
		return m
	}
	if entry.Active || scheduled(entry) {
		m.error = onlyFree
		return m
	}
	m.marked = append(slices.Clone(m.marked), entry.Domain)
	return m
}

// Chỉ số các domain đánh dấu khác domain đang mở; bỏ domain không còn hợp lệ.
func (m Model) targets() []int {
	var result []int
	for index, entry := range m.status.Entries {
		if index != m.entry && slices.Contains(m.marked, entry.Domain) && !entry.Active && !scheduled(entry) {
			result = append(result, index)
		}
	}
	return result
}

func (m Model) isMarked(domain string) bool {
	return slices.Contains(m.marked, domain)
}
