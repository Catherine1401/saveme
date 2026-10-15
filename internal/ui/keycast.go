package ui

import (
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Bật ô hiện phím vừa nhấn, chỉ dùng khi quay video demo.
func (m Model) Keycast(enabled bool) Model {
	m.keycast = enabled
	return m
}

// Ghi phím vừa nhấn cùng ý nghĩa trên màn hình hiện tại.
func (m Model) recordKey(key KeyMsg, now time.Time) Model {
	if !m.keycast {
		return m
	}
	m.castLabel, m.castAt = m.keyMeaning(key), now
	return m
}

// Tên phím dễ đọc và ý nghĩa theo màn hình; màn nhập chữ chỉ ghi "typing".
func (m Model) keyMeaning(key KeyMsg) string {
	const enter, space, escape, up, down, typing, separator, none = "enter", " ", "esc", "up", "down", "typing", "  ", ""
	const enterName, spaceName, upName, downName = "↵", "space", "↑", "↓"
	if m.typing() && key.String() != enter && key.String() != escape {
		return typing
	}
	name := key.String()
	switch name {
	case enter:
		name = enterName
	case space:
		name = spaceName
	case up:
		name = upName
	case down:
		name = downName
	}
	if meaning := m.keyAction(key.String()); meaning != none {
		return name + separator + meaning
	}
	return name
}

// Ý nghĩa ngắn của phím trên màn hình hiện tại.
func (m Model) keyAction(key string) string {
	const enter, escape, quit, left, down, up, downArrow, upArrow, space = "enter", "esc", "q", "h", "j", "k", "down", "up", " "
	const add, search, settings, help, edit, remove, first, last, daily, timed = "a", "/", "s", "?", "e", "d", "g", "G", "1", "2"
	const selectText, backText, downText, upText, addText, openText, searchText, markText = "select", "back", "down", "up", "add domain", "open", "search", "mark"
	const settingsText, helpText, editText, deleteText, quitText, firstText, lastText = "settings", "help", "edit", "delete", "quit", "first", "last"
	const dailyText, timedText, toggleText, saveText, confirmText, cancelText, empty = "daily schedule", "timed lock", "toggle", "save", "confirm", "cancel", ""
	const changeText = "change"
	type action struct{ key, meaning string }
	actions := []action{{enter, selectText}, {escape, backText}, {quit, backText}, {left, backText}, {down, downText}, {up, upText}, {upArrow, upText}, {downArrow, downText}}
	switch m.screen {
	case mainScreen:
		actions = append([]action{{add, addText}, {enter, openText}, {search, searchText}, {space, markText}, {settings, settingsText}, {help, helpText}, {edit, editText}, {remove, deleteText}, {quit, quitText}, {first, firstText}, {last, lastText}}, actions...)
	case domainMenu:
		actions = append([]action{{daily, dailyText}, {timed, timedText}}, actions...)
	case pickerScreen:
		actions = append([]action{{space, toggleText}, {enter, saveText}}, actions...)
	case settingsScreen:
		actions = append([]action{{space, toggleText}, {enter, changeText}}, actions...)
	case confirmation:
		actions = append([]action{{enter, confirmText}, {escape, cancelText}}, actions...)
	}
	for _, item := range actions {
		if item.key == key {
			return item.meaning
		}
	}
	return m.choiceLabel(key)
}

// Nhãn của lựa chọn ứng với phím số trên màn chọn thời lượng hoặc khung giờ mẫu; phím 0 là mục thứ mười.
func (m Model) choiceLabel(key string) string {
	const empty, zero, one, tenth = "", 0, 1, 10
	digit, err := strconv.Atoi(key)
	if err != nil {
		return empty
	}
	if digit == zero {
		digit = tenth
	}
	switch {
	case m.screen == rangeList && digit <= len(durationChoices()):
		return durationChoices()[digit-one].Label
	case m.screen == presetList && digit <= len(presetChoices()):
		return presetChoices()[digit-one].label
	}
	return empty
}

// Ô hiện phím căn phải, hiện trong 1.5 giây sau khi nhấn.
func (m Model) keycastLine() string {
	const lifetime, newline, empty = 1500 * time.Millisecond, "\n", ""
	if !m.keycast || m.castAt.IsZero() || m.frameTime().Sub(m.castAt) > lifetime {
		return empty
	}
	const vertical, horizontal = 0, 1
	badge := m.style(toneConfirm).Bold(true).Reverse(!m.appearance.plain).Padding(vertical, horizontal).Render(m.castLabel)
	width := max(m.contentWidth(), lipgloss.Width(badge))
	return lipgloss.PlaceHorizontal(width, lipgloss.Right, badge) + newline
}
