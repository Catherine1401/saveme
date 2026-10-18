package ui

import (
	"fmt"
	"slices"
	"strings"
)

// Tuỳ chọn giao diện được lưu giữa các phiên.
type Settings struct {
	Theme, Clock string
	Effects      []string
	Warning      bool
	AutoUpdates  bool
}

// Thứ tự các mục trên màn Settings.
const (
	themeSetting = iota
	clockSetting
	effectSetting
	warningSetting
	autoUpdateSetting
	settingCount
)

// Áp tuỳ chọn khởi đầu và hàm lưu; tên lạ thì dùng mặc định.
func (m Model) Configure(settings Settings, save func(Settings) error) Model {
	const first = 0
	theme, ok := findPalette(settings.Theme)
	if !ok {
		theme = palettes()[first]
	}
	clock, ok := findClock(settings.Clock)
	if !ok {
		clock = clockStyles()[first]
	}
	effects := []string{clockEffects()[first].name}
	if settings.Effects != nil {
		effects = []string{}
		for _, effect := range clockEffects() {
			if slices.Contains(settings.Effects, effect.name) {
				effects = append(effects, effect.name)
			}
		}
	}
	m.settings = Settings{Theme: theme.name, Clock: clock.name, Effects: effects, Warning: settings.Warning, AutoUpdates: settings.AutoUpdates}
	m.saveSettings = save
	return m.applySettings(m.settings)
}

// Đưa tuỳ chọn vào phần hiển thị.
func (m Model) applySettings(settings Settings) Model {
	if theme, ok := findPalette(settings.Theme); ok {
		m.appearance.palette = theme
	}
	if clock, ok := findClock(settings.Clock); ok {
		m.appearance.clock = clock
	}
	m.appearance.effects, m.appearance.warning = settings.Effects, settings.Warning
	return m
}

// Lựa chọn của từng mục có danh sách chọn.
func (m Model) pickerOptions() []option {
	var options []option
	switch m.picking {
	case themeSetting:
		for _, item := range palettes() {
			options = append(options, option{name: item.name, label: item.label})
		}
	case clockSetting:
		for _, style := range clockStyles() {
			options = append(options, option{name: style.name, label: style.label})
		}
	case effectSetting:
		options = clockEffects()
	}
	return options
}

// Giá trị đang chọn (đã lưu) của một mục.
func (m Model) settingValue(item int) string {
	const empty = ""
	switch item {
	case themeSetting:
		return m.settings.Theme
	case clockSetting:
		return m.settings.Clock
	}
	return empty
}

// Bản tuỳ chọn với một mục đổi sang giá trị mới.
func (m Model) withSetting(item int, value string) Settings {
	next := m.settings
	switch item {
	case themeSetting:
		next.Theme = value
	case clockSetting:
		next.Clock = value
	}
	return next
}

// Lưu tuỳ chọn mới; lỗi lưu thì giữ nguyên và báo ở dòng lỗi.
func (m Model) commitSettings(next Settings) Model {
	const empty = ""
	if m.saveSettings != nil {
		if err := m.saveSettings(next); err != nil {
			m.error = err.Error()
			return m.applySettings(m.settings)
		}
	}
	m.settings, m.error = next, empty
	return m.applySettings(next)
}

func (m Model) openSettings() Model {
	const zero = 0
	m.screen, m.selected = settingsScreen, zero
	return m
}

// Màn Settings: Enter mở màn chọn, mục bật/tắt thì đổi và lưu ngay.
func (m Model) settingsKey(key KeyMsg) (Model, Cmd) {
	const first = 0
	enter, space := enterName(), spaceName()
	m.selected = move(key.String(), m.selected, settingCount)
	if key.String() != enter && key.String() != space {
		return m, nil
	}
	if m.selected == autoUpdateSetting {
		next := m.settings
		next.AutoUpdates = !next.AutoUpdates
		return m.commitSettings(next), nil
	}
	if m.selected == warningSetting {
		next := m.settings
		next.Warning = !next.Warning
		return m.commitSettings(next), nil
	}
	m.screen, m.picking, m.draft = pickerScreen, m.selected, slices.Clone(m.settings.Effects)
	m.selected = first
	for index, item := range m.pickerOptions() {
		if item.name == m.settingValue(m.picking) {
			m.selected = index
		}
	}
	return m, nil
}

// Màn chọn: di chuyển là xem trước, Enter lưu rồi về Settings.
func (m Model) pickerKey(key KeyMsg) (Model, Cmd) {
	if m.picking == effectSetting {
		return m.effectsKey(key)
	}
	options := m.pickerOptions()
	m.selected = move(key.String(), m.selected, len(options))
	preview := m.withSetting(m.picking, options[m.selected].name)
	return m.previewOrSave(key, preview)
}

// Màn hiệu ứng: space bật/tắt mục đang chọn trên bản nháp và xem trước ngay, Enter lưu bản nháp.
func (m Model) effectsKey(key KeyMsg) (Model, Cmd) {
	space := spaceName()
	options := clockEffects()
	m.selected = move(key.String(), m.selected, len(options))
	if key.String() == space {
		name := options[m.selected].name
		next := []string{}
		for _, effect := range options {
			if (effect.name == name) != slices.Contains(m.draft, effect.name) {
				next = append(next, effect.name)
			}
		}
		m.draft = next
	}
	preview := m.settings
	preview.Effects = m.draft
	return m.previewOrSave(key, preview)
}

// Xem trước cài đặt; Enter thì lưu rồi về màn Settings.
func (m Model) previewOrSave(key KeyMsg, preview Settings) (Model, Cmd) {
	m = m.applySettings(preview)
	if key.String() != enterName() {
		return m, nil
	}
	m = m.commitSettings(preview)
	m.screen, m.selected = settingsScreen, m.picking
	return m, nil
}

// Tên phím Enter dùng chung cho các màn chọn mới.
func enterName() string {
	const enter = "enter"
	return enter
}

// Tên phím Space dùng chung cho các màn chọn mới.
func spaceName() string {
	const space = " "
	return space
}

// Nhãn các hiệu ứng đang bật, nối bằng dấu cộng.
func effectsLabel(effects []string) string {
	const none, separator, noLabels = "None", " + ", 0
	var labels []string
	for _, effect := range clockEffects() {
		if slices.Contains(effects, effect.name) {
			labels = append(labels, effect.label)
		}
	}
	if len(labels) == noLabels {
		return none
	}
	return strings.Join(labels, separator)
}

// Thoát màn chọn, trả lại tuỳ chọn đã lưu.
func (m Model) cancelPicker() Model {
	m = m.applySettings(m.settings)
	m.screen, m.selected = settingsScreen, m.picking
	return m
}

// Nhãn hiển thị của một giá trị.
func labelOf(options []option, name string) string {
	for _, item := range options {
		if item.name == name {
			return item.label
		}
	}
	return name
}

func (m Model) settingsView() string {
	const title, rowFormat, on, off = " Settings\n\n", "%-22s %s", "On", "Off"
	const themeLabel, clockLabel, effectLabel, warningLabel, updateLabel = "Theme", "Clock style", "Clock effect", "Final-minute warning", "Auto updates"
	const enterKey, enterLabel = "↵", "change"
	saved := m.settings
	warning := off
	if saved.Warning {
		warning = on
	}
	automatic := off
	if saved.AutoUpdates {
		automatic = on
	}
	rows := []struct{ label, value string }{
		{themeLabel, labelOf(m.withPicking(themeSetting).pickerOptions(), saved.Theme)},
		{clockLabel, labelOf(m.withPicking(clockSetting).pickerOptions(), saved.Clock)},
		{effectLabel, effectsLabel(saved.Effects)},
		{warningLabel, warning},
		{updateLabel, automatic},
	}
	body := title
	for index, row := range rows {
		body += m.row(index, fmt.Sprintf(rowFormat, row.label, row.value))
	}
	return body + m.keys(keyHint{key: enterKey, label: enterLabel}, moveHint(), backHint())
}

func (m Model) withPicking(item int) Model {
	m.picking = item
	return m
}

// Màn chọn kèm bản xem trước: dòng mẫu màu cho theme, đồng hồ mẫu cho kiểu và hiệu ứng.
func (m Model) pickerView() string {
	const rowFormat, savedMark, newline, sample, empty = "%-32s %s", "✓ saved", "\n", "12:34:56", ""
	const themeTitle, clockTitle, effectTitle, indent = " Theme\n\n", " Clock style\n\n", " Clock effect\n\n", " "
	const sampleLine = " a add  ● locked  K7MQ  ⎿ error"
	const enterKey, enterLabel, escapeKey, escapeLabel = "↵", "save", "esc", "cancel"
	titles := []string{themeTitle, clockTitle, effectTitle}
	body := titles[m.picking]
	const checked, unchecked, toggleKey, toggleLabel = "[x] ", "[ ] ", "space", "toggle"
	for index, item := range m.pickerOptions() {
		mark := empty
		if item.name == m.settingValue(m.picking) {
			mark = savedMark
		}
		label := item.label
		if m.picking == effectSetting {
			label = unchecked + label
			if slices.Contains(m.draft, item.name) {
				label = checked + item.label
			}
		}
		body += m.row(index, fmt.Sprintf(rowFormat, label, mark))
	}
	preview := m.ink(sampleLine, toneText, false)
	if m.picking != themeSetting {
		preview = sample
		if !m.clockStyle().plain {
			const sampleProgress = 0.6
			preview = m.paintClock(m.clockLines(sample), int64(len(sample)*len(sample)), sampleProgress)
		}
	}
	indented := strings.ReplaceAll(preview, newline, newline+indent)
	hints := []keyHint{{key: enterKey, label: enterLabel}, moveHint(), {key: escapeKey, label: escapeLabel}}
	if m.picking == effectSetting {
		hints = append([]keyHint{{key: toggleKey, label: toggleLabel}}, hints...)
	}
	return body + newline + indent + indented + newline + m.keys(hints...)
}
