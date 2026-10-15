package ui

import (
	"fmt"
	"strconv"
	"strings"
)

// Khung giờ mẫu tính bằng phút trong ngày; custom giữ giờ mặc định của trình sửa.
type preset struct {
	label      string
	start, end int
	custom     bool
}

func presetChoices() []preset {
	const hour, night, morning, work, evening, late, early, noon, afternoon = 60, 22, 6, 9, 17, 23, 7, 12, 13
	const nightLabel, workLabel, lateLabel, lunchLabel, customLabel = "Night", "Work hours", "Late night", "Lunch", "Custom…"
	return []preset{
		{label: nightLabel, start: night * hour, end: morning * hour},
		{label: workLabel, start: work * hour, end: evening * hour},
		{label: lateLabel, start: late * hour, end: early * hour},
		{label: lunchLabel, start: noon * hour, end: afternoon * hour},
		{label: customLabel, custom: true},
	}
}

// Thời lượng khung giờ và ghi chú qua nửa đêm, ví dụ "8h, crosses midnight".
func spanText(start, end int) string {
	const day, hour, hours, minutes, midnight, separator = 1440, 60, "%dh", " %dm", ", crosses midnight", ""
	const none = 0
	length := (end - start + day) % day
	text := fmt.Sprintf(hours, length/hour)
	if rest := length % hour; rest != none {
		text += fmt.Sprintf(minutes, rest)
	}
	if end <= start {
		text += midnight
	}
	return text + separator
}

// Mở bước chọn khung giờ mẫu trước khi sửa giờ.
func (m Model) openPresets() Model {
	const zero, empty = 0, ""
	m.screen, m.selected, m.error = presetList, zero, empty
	return m
}

// Chọn mẫu bằng phím số hoặc Enter rồi vào trình sửa giờ đã điền sẵn.
func (m Model) presetKey(key KeyMsg) (Model, Cmd) {
	const one, hour = 1, 60
	choices := presetChoices()
	m.selected = move(key.String(), m.selected, len(choices))
	digit, err := strconv.Atoi(key.String())
	choose := key.String() == enterName()
	if err == nil && digit >= one && digit <= len(choices) {
		m.selected, choose = digit-one, true
	}
	if !choose {
		return m, nil
	}
	chosen := choices[m.selected]
	m.beginWindow()
	if !chosen.custom {
		m.fields = [timeFieldCount]int{chosen.start / hour, chosen.start % hour, chosen.end / hour, chosen.end % hour}
	}
	return m, nil
}

func (m Model) presetView() string {
	const title, rowFormat, customFormat = " Daily window — choose a starting point\n\n", "%-12s %s → %s   %s", "%-12s set your own hours"
	const digitKey, digitLabel, enterKey, enterLabel = "1–5", "choose", "↵", "select"
	body := title
	for index, choice := range presetChoices() {
		text := fmt.Sprintf(rowFormat, choice.label, formatMinute(choice.start), formatMinute(choice.end), spanText(choice.start, choice.end))
		if choice.custom {
			text = fmt.Sprintf(customFormat, choice.label)
		}
		body += m.row(index, text)
	}
	return body + m.keys(keyHint{key: digitKey, label: digitLabel}, keyHint{key: enterKey, label: enterLabel}, moveHint(), backHint())
}

// Thanh 24 giờ: mỗi ô 30 phút, phần bị khóa tô đậm, kèm mốc giờ phía trên.
func (m Model) timeline(start, end int) string {
	const cells, cellMinutes, day, perHour, step, width, newline, space, markFormat = 48, 30, 1440, 2, 6, 2, "\n", " ", "%02d"
	const filled, empty, lastHour, firstHour, firstCell = "━", "┄", 24, 0, 0
	labels := []byte(strings.Repeat(space, cells+width))
	for hour := firstHour; hour <= lastHour; hour += step {
		position := min(hour*perHour, cells)
		copy(labels[position:], fmt.Sprintf(markFormat, hour))
	}
	var bar strings.Builder
	for cell := firstCell; cell < cells; cell++ {
		minute := cell * cellMinutes
		if (minute-start+day)%day < (end-start+day)%day {
			bar.WriteString(m.ink(filled, toneAccent, true))
		} else {
			bar.WriteString(m.ink(empty, toneMuted, false))
		}
	}
	return space + m.ink(strings.TrimRight(string(labels), space), toneMuted, false) + newline + space + bar.String() + newline
}
