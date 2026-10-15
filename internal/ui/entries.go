package ui

import (
	"fmt"
	"strconv"

	"saveme/internal/core"
)

// Trả về trạng thái domain đang thao tác; false khi chỉ số không còn hợp lệ.
func (m Model) current() (core.EntryStatus, bool) {
	const zero = 0
	if m.entry < zero || m.entry >= len(m.status.Entries) {
		return core.EntryStatus{}, false
	}
	return m.status.Entries[m.entry], true
}

// Domain có lock đã lưu (đang khóa hoặc lịch chưa tới giờ).
func scheduled(entry core.EntryStatus) bool {
	const none = ""
	return entry.Kind != none
}

func (m Model) openDomain() (Model, Cmd) {
	const zero, empty = 0, ""
	if m.selected >= len(m.status.Entries) {
		return m, nil
	}
	m.entry, m.error, m.screen = m.selected, empty, domainMenu
	if m.status.Entries[m.entry].Active {
		m.screen = domainLocked
	}
	m.selected = zero
	return m, nil
}

func (m Model) menuKey(key KeyMsg) (Model, Cmd) {
	const enter, choices, zero, one = "enter", 2, 0, 1
	entry, ok := m.current()
	if !ok {
		m.screen, m.selected = mainScreen, zero
		return m, nil
	}
	m.selected = move(key.String(), m.selected, choices)
	digit, err := strconv.Atoi(key.String())
	choose := key.String() == enter
	if err == nil && digit >= one && digit <= choices {
		m.selected, choose = digit-one, true
	}
	if !choose {
		return m, nil
	}
	if m.selected == zero {
		if scheduled(entry) {
			m.screen = windowList
		} else {
			m.beginWindow()
		}
	} else {
		if scheduled(entry) {
			const removeScheduleFirst = "Delete the daily schedule before creating a timed lock."
			m.error = removeScheduleFirst
			return m, nil
		}
		m.screen = rangeList
	}
	m.selected = zero
	return m, nil
}

func (m Model) entryLabel(entry core.EntryStatus) string {
	const zero, locked, remaining, next, unlocked = 0, "● locked", "● locked · %s left", "◷ next lock in %s", "○ unlocked"
	switch {
	case entry.Active && entry.EndAt != zero:
		return fmt.Sprintf(remaining, countdown(entry.EndAt-m.status.Now))
	case entry.Active:
		return locked
	case entry.NextAt != zero:
		return fmt.Sprintf(next, countdown(entry.NextAt-m.status.Now))
	}
	return unlocked
}

func (m Model) mainView() string {
	const header, none, rowFormat = " Domains include subdomains; facebook.com blocks only itself and www\n\n", " No domains.\n", "%-*s  %s"
	const addKey, addLabel, editKey, editLabel, deleteKey, deleteLabel = "a", "add", "e", "edit", "d", "delete"
	const openKey, openLabel, quitKey, quitLabel, noDomains, locked, empty = "↵", "open", "q", "quit", "no domains", "locked", ""
	const zero = 0
	add, quit := keyHint{key: addKey, label: addLabel}, keyHint{key: quitKey, label: quitLabel}
	body := header
	entries := m.status.Entries
	if len(entries) == zero {
		return body + none + m.keys(add, keyHint{key: editKey, label: editLabel, reason: noDomains}, keyHint{key: deleteKey, label: deleteLabel, reason: noDomains}, quit)
	}
	width := len(empty)
	for _, entry := range entries {
		width = max(width, len(entry.Domain))
	}
	for index, entry := range entries {
		body += m.row(index, fmt.Sprintf(rowFormat, width, entry.Domain, m.entryLabel(entry)))
	}
	reason := empty
	if m.selected < len(entries) && entries[m.selected].Active {
		reason = locked
	}
	return body + m.keys(add, keyHint{key: editKey, label: editLabel, reason: reason}, keyHint{key: deleteKey, label: deleteLabel, reason: reason}, keyHint{key: openKey, label: openLabel}, moveHint(), quit)
}

func (m Model) domainMenuView() string {
	const header, digitKey, digitLabel, enterKey, enterLabel = " %s  %s\n\n", "1–2", "choose", "↵", "select"
	const dailyLabel, rangeLabel, timedIndex, reason = "Daily schedule", "Timed lock", 1, "delete the daily schedule first"
	entry, _ := m.current()
	body := fmt.Sprintf(header, entry.Domain, m.entryLabel(entry))
	for index, label := range []string{dailyLabel, rangeLabel} {
		if index == timedIndex && scheduled(entry) {
			body += m.disabledRow(index, label, reason)
		} else {
			body += m.row(index, label)
		}
	}
	return body + m.keys(keyHint{key: digitKey, label: digitLabel}, keyHint{key: enterKey, label: enterLabel}, moveHint(), backHint())
}

func (m Model) domainDeleteView() string {
	const text, enterKey, enterLabel = " Deleting %s also removes its schedule.\n", "↵", "confirm"
	entry, _ := m.current()
	return fmt.Sprintf(text, entry.Domain) + m.keys(keyHint{key: enterKey, label: enterLabel}, backHint())
}
