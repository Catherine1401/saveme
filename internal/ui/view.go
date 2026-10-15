package ui

import (
	"fmt"
	"strings"
	"time"

	"saveme/internal/core"
)

type durationChoice struct {
	Value string
	Label string
}

func durationChoices() []durationChoice {
	const hourOne, hourTwo, hourFour, hourSix = "1h", "2h", "4h", "6h"
	const labelOne, labelTwo, labelFour, labelSix = "1 hour", "2 hours", "4 hours", "6 hours"
	const half, one, three, seven, fourteen, thirty = "12h", "1d", "3d", "7d", "14d", "30d"
	const halfLabel, oneLabel, threeLabel, sevenLabel, fourteenLabel, thirtyLabel = "12 hours", "1 day", "3 days", "7 days", "14 days", "30 days"
	return []durationChoice{{Value: hourOne, Label: labelOne}, {Value: hourTwo, Label: labelTwo}, {Value: hourFour, Label: labelFour}, {Value: hourSix, Label: labelSix}, {Value: half, Label: halfLabel}, {Value: one, Label: oneLabel}, {Value: three, Label: threeLabel}, {Value: seven, Label: sevenLabel}, {Value: fourteen, Label: fourteenLabel}, {Value: thirty, Label: thirtyLabel}}
}

func countdown(seconds int64) string {
	const day, hour, minute, zero = 86400, 3600, 60, 0
	const shortFormat, longFormat = "%02d:%02d:%02d", "%dd %s"
	if seconds < zero {
		seconds = zero
	}
	clock := fmt.Sprintf(shortFormat, seconds%day/hour, seconds%hour/minute, seconds%minute)
	if seconds >= day {
		return fmt.Sprintf(longFormat, seconds/day, clock)
	}
	return clock
}

func (m Model) View() string {
	const (
		textEmpty = ""
	)

	const space, newline = " ", "\n"
	const errorFormat, busy = "\n ⎿ %s\n", " Working…\n"
	const quitKey, quitLabel = "q", "quit"
	quit := keyHint{key: quitKey, label: quitLabel}
	var body string
	if !m.ready {
		const loading = " Connecting to daemon…\n"
		body = space + m.spinner() + loading + m.keys(quit)
	} else if m.screen == domainLocked {
		return m.lockedView()
	} else if !m.usable() {
		const fault = " Changes unavailable.\n"
		body = fault + m.keys(quit)
	} else {
		body = m.screenView()
	}
	if m.error != textEmpty {
		body += m.ink(fmt.Sprintf(errorFormat, m.error), toneError, true)
	}
	if m.busy {
		body += newline + space + m.spinner() + m.ink(busy, toneMuted, false)
	}
	return m.header() + newline + m.decorate(body)
}

func (m Model) screenView() string {

	const empty = ""
	switch m.screen {
	case mainScreen:
		return m.mainView()
	case domainMenu:
		return m.domainMenuView()
	case domainDelete:
		return m.domainDeleteView()
	case domainEdit:
		return m.domainEditView()
	case rangeList:
		const header, digitKey, digitLabel, enterKey, enterLabel = " Select duration\n\n", "1–9", "choose", "↵", "select"
		body := header
		for index, choice := range durationChoices() {
			body += m.row(index, choice.Label)
		}
		return body + m.keys(keyHint{key: digitKey, label: digitLabel}, keyHint{key: enterKey, label: enterLabel}, moveHint(), backHint())
	case windowList:
		return m.windowListView()
	case windowEdit:
		return m.windowEditView()
	case confirmation:
		return m.confirmView()
	case lastWindow:
		const text, enterKey, enterLabel = " Deleting the last window removes the schedule.\n", "↵", "confirm"
		return text + m.keys(keyHint{key: enterKey, label: enterLabel}, backHint())
	}

	return empty
}

func (m Model) row(index int, text string) string {
	const normal, selected, format, one = " ", "❯", " %s %d. %s\n", 1
	marker := normal
	if index == m.selected {
		marker = selected
	}
	line := fmt.Sprintf(format, marker, index+one, text)
	if index == m.selected {
		return m.selectedRow(line)
	}
	return m.ink(line, toneText, false)
}

func (m Model) domainEditView() string {
	const title, canonical, enterKey, enterLabel, escapeKey, escapeLabel = " Enter domain or URL:\n\n", " Will save: %s\n", "↵", "save", "esc", "back"
	body := title + m.inputBox(m.domainText)
	if domain, err := m.normalizer.Normalize(m.domainText); err == nil {
		body += fmt.Sprintf(canonical, domain)
	}
	return body + m.keys(keyHint{key: enterKey, label: enterLabel}, keyHint{key: escapeKey, label: escapeLabel})
}

func (m Model) windowListView() string {
	const header, format = " Daily schedule\n\n", "%s to %s — %s"
	const addKey, addLabel, editKey, editLabel, deleteKey, deleteLabel = "a", "add", "e", "edit", "d", "delete"
	const active, waiting = "active", "waiting"
	body := header
	entry, _ := m.current()
	for index, window := range entry.Windows {

		const zero = 0
		label := waiting
		lock := &core.Lock{Kind: core.Daily, Zone: entry.Zone, Windows: []core.Window{window}}

		if lock.Active(time.Unix(m.status.Now, zero)) {
			label = active
		}
		body += m.row(index, fmt.Sprintf(format, formatMinute(window.Start), formatMinute(window.End), label))
	}
	return body + m.keys(keyHint{key: addKey, label: addLabel}, keyHint{key: editKey, label: editLabel}, keyHint{key: deleteKey, label: deleteLabel}, moveHint(), backHint())
}

func (m Model) windowEditView() string {
	const (
		textEmpty = ""
	)

	const header, format, appended = " Daily window\n\n", " %s  %s : %s  to  %s : %s\n", " Added: %s to %s\n"
	const plain, focus, zero, one, two, three = "%02d", "[%02d]", 0, 1, 2, 3
	var fields [timeFieldCount]string
	for index, value := range m.fields {
		fieldFormat := plain
		if index == m.field {
			fieldFormat = focus
		}
		fields[index] = fmt.Sprintf(fieldFormat, value)
		if index == m.field {
			fields[index] = m.ink(fields[index], toneConfirm, true)
		}
	}
	body := header
	if m.editingWindow {
		const editHeader = " Edit daily window\n\n"
		body = editHeader
	}
	for _, window := range m.windows {
		body += fmt.Sprintf(appended, formatMinute(window.Start), formatMinute(window.End))
	}
	body += fmt.Sprintf(format, textEmpty, fields[zero], fields[one], fields[two], fields[three])
	const fieldKey, fieldLabel, adjustKey, adjustLabel, digitKey, digitLabel = "h/l ←→", "field", "j/k ↑↓", "adjust", "0–9", "type"
	const addKey, addLabel, enterKey, enterLabel, escapeKey, escapeLabel = "a", "add window", "↵", "save", "esc", "back"
	hints := []keyHint{{key: fieldKey, label: fieldLabel}, {key: adjustKey, label: adjustLabel}, {key: digitKey, label: digitLabel}}
	if !m.editingWindow {
		hints = append(hints, keyHint{key: addKey, label: addLabel})
	}
	return body + m.keys(append(hints, keyHint{key: enterKey, label: enterLabel}, keyHint{key: escapeKey, label: escapeLabel})...)
}

func (m Model) confirmView() string {
	const header, daily, rangeFormat, domainFormat, notice, inputFormat = " Confirm lock\n\n", " Daily: %s to %s\n", " Duration: %s\n Start: %s\n Estimated end: %s\n", " Domain: %s, including subdomains; facebook.com blocks only itself and www\n", "\n While the confirmed lock is active\n you cannot edit or delete this domain or its schedule.\n\n Type the code below (no paste):\n", "\n Code: %s — %s remaining\n"
	const placeholder, separator, codeLength, zero = "_", " ", 4, 0
	const enterKey, enterLabel, escapeKey, escapeLabel = "↵", "confirm", "esc", "cancel"
	if m.pending.Preview == nil {
		return header
	}
	state := m.pending.Preview
	body := header
	if state.Lock == nil {
		return header
	}
	if state.Lock.Kind == core.Daily {
		for _, window := range state.Lock.Windows {
			body += fmt.Sprintf(daily, formatMinute(window.Start), formatMinute(window.End))
		}
	} else {
		span := state.Lock.EndAt - state.Lock.StartAt
		body += fmt.Sprintf(rangeFormat, countdown(span), time.Unix(state.Lock.StartAt, zero).Format(time.RFC3339), time.Unix(state.Lock.EndAt, zero).Format(time.RFC3339))
	}
	body += fmt.Sprintf(domainFormat, state.Domain)
	body += notice + m.confirmCode()
	var slots []string
	for index := zero; index < codeLength; index++ {
		char := placeholder
		if index < len(m.codeInput) {
			char = string(m.codeInput[index])
		}
		slots = append(slots, char)
	}
	remaining := m.pending.Now + m.pending.ExpiresIn - m.status.Now
	return body + fmt.Sprintf(inputFormat, strings.Join(slots, separator), countdown(remaining)) + m.keys(keyHint{key: enterKey, label: enterLabel}, keyHint{key: escapeKey, label: escapeLabel})
}

// Gợi ý di chuyển dọc dùng chung cho các danh sách.
func moveHint() keyHint {
	const key, label = "j/k ↑↓", "move"
	return keyHint{key: key, label: label}
}

// Gợi ý quay lại một cấp dùng chung cho các màn con.
func backHint() keyHint {
	const key, label = "h/q", "back"
	return keyHint{key: key, label: label}
}
