package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

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
	index, ok := m.selectedEntry()
	if !ok {
		return m, nil
	}
	m.entry, m.error, m.screen = index, empty, domainMenu
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
			m = m.openPresets()
			return m, nil
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
	const header = " Domains\n\n"
	const welcome = " No domains yet.\n\n 1. Press a and type a site to block, e.g. youtube.com.\n 2. Open it with ↵.\n 3. Choose a daily schedule or a timed lock.\n"
	const addKey, addLabel, openKey, openLabel, helpKey, helpLabel, quitKey, quitLabel = "a", "add", "↵", "open", "?", "help", "q", "quit"
	const zero, empty = 0, ""
	const searchFormat, cursor, noMatch = " / %s%s\n\n", "_", " No matching domains.\n"
	const searchKey, searchLabel = "/", "search"
	add, help, quit := keyHint{key: addKey, label: addLabel}, keyHint{key: helpKey, label: helpLabel}, keyHint{key: quitKey, label: quitLabel}
	entries := m.status.Entries
	if len(entries) == zero {
		return welcome + m.keys(add, help, quit)
	}
	body := header
	if m.searching || m.filter != empty {
		mark := empty
		if m.searching {
			mark = cursor
		}
		body += fmt.Sprintf(searchFormat, m.filter, mark)
	}
	order := m.order()
	if len(order) == zero {
		body += noMatch
	} else {
		body += m.domainTable(order, m.contentWidth())
	}
	if len(m.marked) > zero {
		const markedFormat = "\n %d marked · open one to lock them together · esc clears\n"
		body += fmt.Sprintf(markedFormat, len(m.marked))
	}
	if m.searching {
		const enterKey, enterLabel, escapeKey, escapeLabel = "↵", "keep filter", "esc", "clear"
		return body + m.keys(keyHint{key: enterKey, label: enterLabel}, keyHint{key: escapeKey, label: escapeLabel})
	}
	return body + m.keys(add, keyHint{key: openKey, label: openLabel}, keyHint{key: searchKey, label: searchLabel}, help, quit)
}

func (m Model) domainMenuView() string {
	const header, digitKey, digitLabel, enterKey, enterLabel = " %s  %s\n\n", "1–2", "choose", "↵", "select"
	const dailyLabel, rangeLabel, timedIndex, reason, zero = "Daily schedule", "Timed lock", 1, "delete the daily schedule first", 0
	entry, _ := m.current()
	body := fmt.Sprintf(header, entry.Domain, m.entryLabel(entry))
	if extra := len(m.targets()); extra > zero && !scheduled(entry) {
		const groupFormat = " Also locks %d marked domain(s).\n\n"
		body += fmt.Sprintf(groupFormat, extra)
	}
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
	const plain, withSchedule, enterKey, enterLabel = " Delete %s?\n", " Deleting %s also removes its schedule.\n", "↵", "delete"
	entry, _ := m.current()
	text := plain
	if scheduled(entry) {
		text = withSchedule
	}
	return fmt.Sprintf(text, entry.Domain) + m.keys(keyHint{key: enterKey, label: enterLabel}, backHint())
}

// Màn trợ giúp liệt kê mọi phím và quy tắc chặn domain; phím tô màu nhấn, mô tả màu chữ thường.
func (m Model) helpView() string {
	const title, rule, keyWidth, newline, space = " Keys\n\n", " Domains include subdomains; facebook.com blocks only itself and www.\n", 9, "\n", " "
	const addKey, addLabel, editKey, editLabel = "a", "add a domain", "e", "edit the selected domain (unlocked only)"
	const deleteKey, deleteLabel, openKey, openLabel = "d", "delete the selected domain (unlocked only)", "↵ / l", "open the selected domain"
	const filterKey, filterLabel, markKey, markLabel = "/", "filter domains; esc clears", "space", "mark domains to lock together; esc clears"
	const moveKey, moveLabel, backKey, backLabel = "j/k ↑↓", "move · gg / G first / last", "h / q", "back · q on the list quits"
	const settingsKey, settingsLabel, helpKey, helpLabel = "s", "settings: theme, clock style and effects", "?", "this help"
	rows := []keyHint{
		{key: addKey, label: addLabel},
		{key: editKey, label: editLabel},
		{key: deleteKey, label: deleteLabel},
		{key: openKey, label: openLabel},
		{key: filterKey, label: filterLabel},
		{key: markKey, label: markLabel},
		{key: moveKey, label: moveLabel},
		{key: backKey, label: backLabel},
		{key: settingsKey, label: settingsLabel},
		{key: helpKey, label: helpLabel},
	}
	body := title
	for _, row := range rows {
		padded := row.key + strings.Repeat(space, max(keyWidth-lipgloss.Width(row.key), len(space)))
		body += space + m.ink(padded, toneAccent, true) + m.ink(row.label, toneText, false) + newline
	}
	return body + newline + m.ink(rule, toneMuted, false) + m.keys(backHint())
}
