package ui

import (
	"fmt"
	"strconv"

	"saveme/internal/core"
)

func (m *Model) beginWindow() {
	const ()

	const defaultHour, endHour, zero, empty = 22, 6, 0, ""
	m.screen, m.windows, m.field, m.digits = windowEdit, nil, zero, empty
	m.fields = [timeFieldCount]int{defaultHour, zero, endHour, zero}
}

func (m Model) windowListKey(key KeyMsg) (Model, Cmd) {
	const add, remove, one = "a", "d", 1
	if m.status.State.Lock == nil {
		m.screen = mainScreen
		return m, nil
	}
	windows := m.status.State.Lock.Windows
	m.selected = move(key.String(), m.selected, len(windows))
	switch key.String() {
	case add:
		m.beginWindow()
	case remove:
		if len(windows) == one {
			m.screen = lastWindow
			return m, nil
		}
		return m.submit(core.Request{Cmd: core.DeleteWindow, Index: &m.selected})
	}
	return m, nil
}

func (m Model) currentWindow() core.Window {
	const minutes, startHour, startMinute, endHour, endMinute = 60, 0, 1, 2, 3
	return core.Window{Start: m.fields[startHour]*minutes + m.fields[startMinute], End: m.fields[endHour]*minutes + m.fields[endMinute]}
}

func formatMinute(minute int) string {
	const format, minutes = "%02d:%02d", 60
	return fmt.Sprintf(format, minute/minutes, minute%minutes)
}

func (m Model) windowKey(key KeyMsg) (Model, Cmd) {
	const left, right, up, down, add, enter = "left", "right", "up", "down", "a", "enter"
	const zero, one, fields, hourLimit, minuteLimit, parity, empty = 0, 1, 4, 24, 60, 2, ""
	limit := minuteLimit
	if m.field%parity == zero {
		limit = hourLimit
	}
	switch key.String() {
	case left:
		m.field, m.digits = (m.field+fields-one)%fields, empty
	case right:
		m.field, m.digits = (m.field+one)%fields, empty
	case up:
		m.fields[m.field], m.digits = (m.fields[m.field]+one)%limit, empty
	case down:
		m.fields[m.field], m.digits = (m.fields[m.field]+limit-one)%limit, empty
	case add, enter:
		return m.saveWindow(key.String() == enter)
	default:
		const (
			number10 = 10
		)

		const low, high, twoDigits = '0', '9', 2
		if key.Type != KeyRunes || len(key.Runes) != one || key.Runes[zero] < low || key.Runes[zero] > high {
			return m, nil
		}
		text := m.digits + string(key.Runes)
		value, err := strconv.Atoi(text)
		if err != nil || value >= limit || (len(text) == one && value > (limit-one)/number10) {
			const (
				invalidTime = "Time out of range."
			)

			m.error = invalidTime
			return m, nil
		}
		m.fields[m.field], m.digits, m.error = value, text, empty
		if len(text) == twoDigits {
			m.field, m.digits = (m.field+one)%fields, empty
		}
	}
	return m, nil
}

func (m Model) saveWindow(save bool) (Model, Cmd) {
	const zero, empty = 0, ""
	current := m.currentWindow()
	windows := append(append([]core.Window{}, m.windows...), current)
	validation := append([]core.Window{}, windows...)
	if m.status.State.Lock != nil {
		validation = append(validation, m.status.State.Lock.Windows...)
	}
	if err := core.ValidateWindows(validation); err != nil {
		m.error = string(err.(core.ErrorCode))
		return m, nil
	}
	if m.status.State.Lock != nil {
		const (
			confirmSingleWindow = "Press Enter to confirm the new window."
		)

		if save {
			return m.submit(core.Request{Cmd: core.AddWindow, Start: formatMinute(current.Start), End: formatMinute(current.End)})
		}
		m.error = confirmSingleWindow
		return m, nil
	}
	if !save {
		const ()

		m.windows, m.fields, m.field, m.digits, m.error = windows, [timeFieldCount]int{}, zero, empty, empty
		return m, nil
	}
	var inputs []core.InputWindow
	for _, window := range windows {
		inputs = append(inputs, core.InputWindow{Start: formatMinute(window.Start), End: formatMinute(window.End)})
	}
	return m.submit(core.Request{Cmd: core.CreateDaily, Windows: inputs})
}
