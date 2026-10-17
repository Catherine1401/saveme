package ui

import (
	"fmt"
	"strconv"

	"saveme/internal/core"
)

type durationInput struct {
	hours   int
	minutes int
	field   int
	digits  string
}

func (m Model) beginDuration() Model {
	const hour, empty = 1, ""
	m.screen, m.error = durationEdit, empty
	m.duration = durationInput{hours: hour}
	return m
}

func (m Model) durationKey(key KeyMsg) (Model, Cmd) {
	const left, right, up, down, enter = "left", "right", "up", "down", "enter"
	const zero, one, fieldCount, hourValues, minuteValues, empty = 0, 1, 2, 25, 60, ""
	if key.String() == enter {
		return m.saveDuration()
	}
	if key.String() == left || key.String() == right {
		m.duration.field, m.duration.digits = (m.duration.field+one)%fieldCount, empty
		return m, nil
	}
	value, limit := &m.duration.hours, hourValues
	if m.duration.field != zero {
		value, limit = &m.duration.minutes, minuteValues
	}
	if key.String() == up || key.String() == down {
		delta := one
		if key.String() == down {
			delta = -one
		}
		*value, m.duration.digits, m.error = (*value+limit+delta)%limit, empty, empty
		return m, nil
	}
	const low, high, twoDigits = '0', '9', 2
	if key.Paste || key.Type != KeyRunes || len(key.Runes) != one || key.Runes[zero] < low || key.Runes[zero] > high {
		return m, nil
	}
	text := m.duration.digits + string(key.Runes)
	number, err := strconv.Atoi(text)
	if err != nil || number >= limit {
		const invalid = "Time out of range."
		m.error = invalid
		return m, nil
	}
	*value, m.duration.digits, m.error = number, text, empty
	if len(text) == twoDigits {
		m.duration.field, m.duration.digits = (m.duration.field+one)%fieldCount, empty
	}
	return m, nil
}

func (m Model) saveDuration() (Model, Cmd) {
	const minutesPerHour, format = 60, "%dm"
	text := fmt.Sprintf(format, m.duration.hours*minutesPerHour+m.duration.minutes)
	if _, err := core.Duration(text); err != nil {
		const invalid = "Enter a duration from 00:01 to 24:00."
		m.error = invalid
		return m, nil
	}
	return m.submit(core.Request{Cmd: core.CreateRange, Entry: &m.entry, Targets: m.targets(), Duration: text})
}

func (m Model) durationView() string {
	const title = " Custom time\n\n Hours : Minutes\n"
	const plain, focus, row, zero = "%02d", "[%02d]", "%s : %s", 0
	formatHour, formatMinute := plain, focus
	if m.duration.field == zero {
		formatHour, formatMinute = focus, plain
	}
	body := title + m.inputBox(fmt.Sprintf(row, fmt.Sprintf(formatHour, m.duration.hours), fmt.Sprintf(formatMinute, m.duration.minutes)))
	const bounds = " From 00:01 to 24:00 (max 24 hours).\n"
	const fields, fieldLabel, adjust, adjustLabel, digits, typeLabel = "h/l ←→", "field", "j/k ↑↓", "adjust", "0–9", "type"
	const enter, next, escape, back = "↵", "confirm", "esc", "back"
	return body + bounds + m.keys(keyHint{key: fields, label: fieldLabel}, keyHint{key: adjust, label: adjustLabel}, keyHint{key: digits, label: typeLabel}, keyHint{key: enter, label: next}, keyHint{key: escape, label: back})
}
