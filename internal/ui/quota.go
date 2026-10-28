package ui

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"saveme/internal/core"
)

const (
	quotaMenuIndex = 2
	clockUnsynced  = "clock not synced"
	linuxOnly      = "Linux only"
)

// Số lựa chọn của menu domain: lịch, khóa theo giờ và quota.
func menuChoices() int {
	const count = 3
	return count
}

// Lý do không đặt được quota; chuỗi rỗng nghĩa là đặt được.
func (m Model) quotaBlocked() string {
	const none, windows = "", "windows"
	switch {
	case runtime.GOOS == windows:
		return linuxOnly
	case m.status.Error == core.ErrClockUnsync:
		return clockUnsynced
	}
	return none
}

// Các mức quota có sẵn: chọn từ danh sách thời lượng những mức không quá 24 giờ.
func quotaChoices() []durationChoice {
	const maxQuotaSeconds = core.MaxQuotaSec
	var result []durationChoice
	for _, choice := range durationChoices() {
		if length, err := core.Duration(choice.Value); err == nil && int64(length.Seconds()) <= maxQuotaSeconds {
			result = append(result, choice)
		}
	}
	return result
}

// Thanh tiến độ phần quota còn lại, ví dụ "▓▓▓░░░".
func quotaBar(quota *core.QuotaStatus) string {
	const cells, filled, empty, zero = 6, "▓", "░", 0
	share := zero
	if quota.LimitSec > zero {
		share = int(quota.RemainingSec * cells / quota.LimitSec)
	}
	share = max(zero, min(cells, share))
	return strings.Repeat(filled, share) + strings.Repeat(empty, cells-share)
}

// Nhãn quota của domain: thanh còn lại, đã dùng và giới hạn, ví dụ "▓▓▓░░░ 18m of 30m used".
func quotaSummary(quota *core.QuotaStatus) string {
	const format = "%s %s of %s used"
	return fmt.Sprintf(format, quotaBar(quota), spanLabel(quota.UsedSec), spanLabel(quota.LimitSec))
}

// Câu "opens in ..." của domain đã hết quota, dùng chung cho bảng chính và menu.
func (m Model) quotaOpens(entry core.EntryStatus) string {
	const opens = "opens in %s"
	return fmt.Sprintf(opens, countdown(entry.EndAt-m.status.Now))
}

// Gợi ý cách khắc phục kèm lý do không đặt được quota.
func quotaHint(reason string) string {
	const clockHint, linuxHint = "clock not synced — wait for time sync, then retry", "needs Linux"
	switch reason {
	case clockUnsynced:
		return clockHint
	case linuxOnly:
		return linuxHint
	}
	return reason
}

// Domain đang chặn vì quota đã dùng hết.
func quotaUsedUp(entry core.EntryStatus) bool {
	const none = 0
	return entry.Active && entry.Quota != nil && entry.Quota.RemainingSec == none
}

// Thêm thông tin quota vào hai cột trạng thái và thời gian của bảng chính.
func (m Model) withQuota(entry core.EntryStatus, status, remaining string) (string, string) {
	const usedUp, none, joiner, quotaLeft, zero = "● used up", "—", " · ", "%s %s left", 0
	if entry.Quota == nil {
		return status, remaining
	}
	if quotaUsedUp(entry) && entry.EndAt != zero {
		return usedUp, m.quotaOpens(entry)
	}
	if quotaUsedUp(entry) {
		return status, remaining
	}
	text := fmt.Sprintf(quotaLeft, quotaBar(entry.Quota), spanLabel(entry.Quota.RemainingSec))
	if remaining != none {
		text = remaining + joiner + text
	}
	return status, text
}

func (m Model) quotaKey(key KeyMsg) (Model, Cmd) {
	const enter, one, zero, remove = "enter", 1, 0, "d"
	choices := quotaChoices()
	if entry, ok := m.current(); key.String() == remove && ok && entry.Quota != nil {
		return m.submit(core.Request{Cmd: core.ClearQuota, Entry: &m.entry})
	}
	m.selected = move(key.String(), m.selected, len(choices)+one)
	digit, err := strconv.Atoi(key.String())
	choose := key.String() == enter
	if err == nil && digit >= one && digit <= len(choices) {
		m.selected, choose = digit-one, true
	}
	if !choose {
		return m, nil
	}
	if m.selected == len(choices) {
		m.quotaMode = true
		return m.beginDuration(), nil
	}
	return m.submit(core.Request{Cmd: core.SetQuota, Entry: &m.entry, Duration: choices[m.selected].Value})
}

func (m Model) quotaView() string {
	const title, newline, digitKey, digitLabel, enterKey, enterLabel, customLabel = " Daily quota — time allowed per day\n", "\n", "1–7", "choose", "↵", "select", "Custom time"
	const usedFormat, blocksNow, removeKey, removeLabel = " Current: %s\n", "%s (blocks now)", "d", "remove quota"
	body := title
	entry, _ := m.current()
	if entry.Quota != nil {
		body += fmt.Sprintf(usedFormat, quotaSummary(entry.Quota))
	}
	body += newline
	for index, choice := range quotaChoices() {
		label := choice.Label
		if length, err := core.Duration(choice.Value); entry.Quota != nil && err == nil && int64(length.Seconds()) <= entry.Quota.UsedSec {
			label = fmt.Sprintf(blocksNow, label)
		}
		body += m.row(index, label)
	}
	body += m.row(len(quotaChoices()), customLabel)
	hints := []keyHint{{key: digitKey, label: digitLabel}, {key: enterKey, label: enterLabel}}
	if entry.Quota != nil {
		hints = append(hints, keyHint{key: removeKey, label: removeLabel})
	}
	return body + m.keys(append(hints, moveHint(), backHint())...)
}

// Dòng quota của menu domain như các dòng lock; bị vô hiệu kèm lý do khi không đặt được.
func (m Model) quotaRows(entry core.EntryStatus) string {
	const lockedReason, joiner, none, menuLabel = "locked until 00:00", " · ", "", "Daily quota"
	label := menuLabel
	if entry.Quota != nil {
		label += joiner + quotaSummary(entry.Quota)
	}
	if quotaUsedUp(entry) {
		return m.disabledRow(quotaMenuIndex, label, lockedReason)
	}
	if reason := m.quotaBlocked(); reason != none {
		return m.disabledRow(quotaMenuIndex, label, reason)
	}
	return m.row(quotaMenuIndex, label)
}

// Chọn dòng quota trong menu domain: mở màn quota, nơi đặt mức mới hoặc xóa quota.
func (m Model) quotaMenuChoice(entry core.EntryStatus) (Model, Cmd) {
	const zero, none, usedUpError, unavailablePrefix, sentenceEnd = 0, "", "Quota used up — it stays locked until 00:00.", "Quota unavailable: ", "."
	if quotaUsedUp(entry) {
		m.error = usedUpError
		return m, nil
	}
	if reason := m.quotaBlocked(); reason != none {
		m.error = unavailablePrefix + quotaHint(reason) + sentenceEnd
		return m, nil
	}
	m.screen, m.selected, m.quotaMode = quotaList, zero, false
	return m, nil
}

// Màn xác nhận đặt quota: giới hạn mỗi ngày và hậu quả khi dùng hết.
func (m Model) quotaConfirmView(state *core.Entry) string {
	const header, limitFormat, domainFormat = " Confirm quota\n\n", " Limit: %s per day\n", " Domain: %s, including subdomains; facebook.com blocks only itself and www\n"
	const warning, codePrompt = "\n ⚠ When used up: stays blocked until 00:00, cannot edit or delete.\n", "\n Type the code below (no paste):\n"
	body := header + fmt.Sprintf(limitFormat, spanLabel(state.Quota.LimitSec)) + fmt.Sprintf(domainFormat, state.Domain)
	return body + m.ink(warning, toneWarning, true) + codePrompt + m.codeEntry()
}
