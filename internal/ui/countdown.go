package ui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"saveme/internal/core"
)

// Độ dài khoảng thời gian dạng gọn: "3d", "12h", "1h 30m".
func spanLabel(seconds int64) string {
	const day, hour, minute, zero = 86400, 3600, 60, 0
	const dayFormat, hourFormat, minuteFormat, separator = "%dd", "%dh", "%dm", " "
	var parts []string
	if days := seconds / day; days > zero {
		parts = append(parts, fmt.Sprintf(dayFormat, days))
	}
	if hours := seconds % day / hour; hours > zero {
		parts = append(parts, fmt.Sprintf(hourFormat, hours))
	}
	if minutes := seconds % hour / minute; minutes > zero || len(parts) == zero {
		parts = append(parts, fmt.Sprintf(minuteFormat, minutes))
	}
	return strings.Join(parts, separator)
}

// Loại khóa kèm thời lượng hoặc khung giờ theo giờ máy.
func kindText(entry core.EntryStatus) string {
	const zero, timed, daily, timedFormat, dailyFormat, clock = 0, "Timed lock", "Daily schedule", "Timed lock · %s", "Daily schedule · %s → %s", "15:04"
	known := entry.StartAt != zero && entry.EndAt != zero
	switch {
	case entry.Kind == core.Range && known:
		return fmt.Sprintf(timedFormat, spanLabel(entry.EndAt-entry.StartAt))
	case entry.Kind == core.Range:
		return timed
	case known:
		return fmt.Sprintf(dailyFormat, time.Unix(entry.StartAt, zero).Local().Format(clock), time.Unix(entry.EndAt, zero).Local().Format(clock))
	}
	return daily
}

// Giờ mở khóa theo giờ máy: hôm nay, ngày mai hoặc ngày cụ thể.
func unlockText(now, end int64) string {
	const zero, one, clock, date = 0, 1, "15:04", "Mon 2 Jan"
	const todayFormat, tomorrowFormat, laterFormat = "Unlocks at %s today", "Unlocks tomorrow at %s", "Unlocks %s at %s"
	current, target := time.Unix(now, zero).Local(), time.Unix(end, zero).Local()
	sameDay := func(a, b time.Time) bool { return a.Year() == b.Year() && a.YearDay() == b.YearDay() }
	switch {
	case sameDay(current, target):
		return fmt.Sprintf(todayFormat, target.Format(clock))
	case sameDay(current.AddDate(zero, zero, one), target):
		return fmt.Sprintf(tomorrowFormat, target.Format(clock))
	}
	return fmt.Sprintf(laterFormat, target.Format(date), target.Format(clock))
}

// Thanh tiến độ đã khóa được bao nhiêu phần, kèm phần trăm và thời lượng.
func (m Model) progressBar(start, end int64) string {
	const maxCells, margin, minCells, percent, zero, filled, empty, format = 48, 24, 10, 100, 0, "━", "┄", " %d%% · %s of %s"
	const minTotal int64 = 1
	cells := maxCells
	if m.appearance.width > zero {
		cells = max(minCells, min(maxCells, m.contentWidth()-margin))
	}
	total, done := end-start, max(zero, min(m.status.Now-start, end-start))
	share := int(done * int64(cells) / max(total, minTotal))
	bar := m.ink(strings.Repeat(filled, share), toneAccent, true) + m.ink(strings.Repeat(empty, cells-share), toneMuted, false)
	return bar + m.emphasize(fmt.Sprintf(format, int(done*percent/max(total, minTotal)), spanLabel(done), spanLabel(total)))
}

// Các dòng giữa màn: số ngày, đồng hồ lớn, thời gian còn lại, tiến độ, giờ mở.
func (m Model) lockFacts(entry core.EntryStatus) string {
	const zero, day, reserved, margin, newline, gap, separator = 0, 86400, 13, 4, "\n", "\n\n", " "
	const label, daysFormat, oneDay = "Locked: %s remaining", "%d days", "1 day"
	const empty, single, last, sides, noProgress = "", 1, 1, 2, -1.0
	const unsynced, wholeDay = "Clock not synced — this lock stays on until the time is verified.", "This schedule covers the whole day, so it has no unlock time."
	if entry.EndAt == zero {
		if m.status.Error == core.ErrClockUnsync {
			return m.ink(unsynced, toneWarning, true) + newline
		}
		return m.ink(wholeDay, toneText, true) + newline
	}
	left := entry.EndAt - m.status.Now
	remaining := countdown(left)
	body := empty
	if days := left / day; days > zero {
		text := fmt.Sprintf(daysFormat, days)
		if days == single {
			text = oneDay
		}
		body += m.ink(text, toneText, true) + newline
	}
	parts := strings.Split(remaining, separator)
	clock := parts[len(parts)-last]
	style := m.clockStyle()
	// Đồng hồ lớn đã hiện thời gian còn lại; chỉ dùng dòng chữ khi terminal quá nhỏ.
	switch lines := m.clockLines(clock); {
	case style.plain:
		body += m.ink(clock, toneAccent, true) + gap
	case m.appearance.width >= lipgloss.Width(lines[zero])+margin+m.decorPad()*sides && m.appearance.height >= len(lines)+reserved+m.decorPad()*sides:
		progress := float64(noProgress)
		if entry.StartAt != zero && entry.EndAt > entry.StartAt {
			progress = float64(m.status.Now-entry.StartAt) / float64(entry.EndAt-entry.StartAt)
		}
		body += m.paintClock(lines, left, progress) + gap
	default:
		body += m.ink(fmt.Sprintf(label, remaining), toneText, true) + newline
	}
	if entry.StartAt != zero {
		body += m.progressBar(entry.StartAt, entry.EndAt) + newline
	}
	return body + m.emphasize(unlockText(m.status.Now, entry.EndAt)) + newline
}

func (m Model) lockedView() string {
	const message, newline, gap, quitKey, footer = "%s is blocked.", "\n", "\n\n", "q", "back"
	entry, _ := m.current()
	head := m.ink(fmt.Sprintf(message, entry.Domain), toneAccent, true) + newline + m.emphasize(kindText(entry)) + gap
	return head + m.lockFacts(entry) + m.keys(keyHint{key: quitKey, label: footer})
}

// Tô điểm dòng thông tin: giờ, phần trăm, thời lượng màu nhấn đậm; ký hiệu nối màu mờ; chữ còn lại màu thường.
func (m Model) emphasize(text string) string {
	const pattern, separators, start, all = `\d+%|\d{2}:\d{2}|\b\d+[dhm]\b`, "·→", 0, -1
	const from, to = 0, 1
	values := regexp.MustCompile(pattern)
	var out strings.Builder
	words := func(segment string) {
		var run strings.Builder
		current := toneText
		for _, char := range segment {
			tone := toneText
			if strings.ContainsRune(separators, char) {
				tone = toneMuted
			}
			if tone != current && run.Len() > start {
				out.WriteString(m.ink(run.String(), current, false))
				run.Reset()
			}
			current = tone
			run.WriteRune(char)
		}
		out.WriteString(m.ink(run.String(), current, false))
	}
	last := start
	for _, match := range values.FindAllStringIndex(text, all) {
		words(text[last:match[from]])
		out.WriteString(m.ink(text[match[from]:match[to]], toneAccent, true))
		last = match[to]
	}
	words(text[last:])
	return out.String()
}
