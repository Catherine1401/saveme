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

	const header, errorFormat, busy = " saveme\n\n", "\n %s\n", "\n Working…\n"
	var body string
	if !m.ready {
		const loading = " Connecting to daemon…\n q quit\n"
		body = loading
	} else if m.status.Active {
		const infinite, remaining, message = " Locked with no release time.\n", " Locked: %s remaining\n", "\n Website blocking is active.\n\n q quit\n"
		const zero = 0
		body = infinite
		if m.status.EndAt != zero {
			body = fmt.Sprintf(remaining, countdown(m.status.EndAt-m.status.Now))
		}
		body += message
	} else if !m.status.OK {
		const fault = " Changes unavailable.\n q quit\n"
		body = fault
	} else {
		body = m.screenView()
	}
	if m.error != textEmpty {
		body += fmt.Sprintf(errorFormat, m.error)
	}
	if m.busy {
		body += busy
	}
	return header + body
}

func (m Model) screenView() string {

	const empty = ""
	switch m.screen {
	case mainScreen:
		return m.mainView()
	case domainList:
		return m.domainListView()
	case domainEdit:
		return m.domainEditView()
	case rangeList:
		const header, footer = " Select duration\n\n", "\n ↑↓ / 1–6 / Enter select; Esc back\n"
		body := header
		for index, choice := range durationChoices() {
			body += m.row(index, choice.Label)
		}
		return body + footer
	case windowList:
		return m.windowListView()
	case windowEdit:
		return m.windowEditView()
	case confirmation:
		return m.confirmView()
	case lastWindow:
		const text = " Deleting the last window removes the schedule.\n Enter confirm; Esc back\n"
		return text
	}

	return empty
}

func (m Model) row(index int, text string) string {
	const normal, selected, format, one = " ", "›", " %s %d  %s\n", 1
	marker := normal
	if index == m.selected {
		marker = selected
	}
	return fmt.Sprintf(format, marker, index+one, text)
}

func (m Model) mainView() string {
	const idle, next, footer, zero = " No lock\n\n", " Next lock in %s\n\n", "\n ↑↓ / 1–3 / Enter select; q quit\n", 0
	const dailyLabel, rangeLabel, domainLabel, unavailable = "Daily schedule", "Timed lock", "Edit domains", " (delete the daily schedule first)"
	body := idle
	if m.status.NextAt != zero {
		body = fmt.Sprintf(next, countdown(m.status.NextAt-m.status.Now))
	}
	rangeText := rangeLabel
	if m.status.Kind == core.Daily {
		const faint, reset = "\x1b[2m", "\x1b[0m"
		rangeText = faint + rangeText + unavailable + reset
	}
	labels := []string{dailyLabel, rangeText, domainLabel}
	for index, label := range labels {
		body += m.row(index, label)
	}
	return body + footer
}

func (m Model) domainListView() string {
	const header, footer, empty = " Domains include subdomains; facebook.com blocks only itself and www\n\n", "\n a add; e edit; d delete; ↑↓ select; Esc back\n", " No domains.\n"
	const zero = 0
	body := header
	if len(m.status.State.Domains) == zero {
		body += empty
	}
	for index, domain := range m.status.State.Domains {
		body += m.row(index, domain)
	}
	return body + footer
}

func (m Model) domainEditView() string {
	const format, canonical, footer = " Enter domain or URL:\n\n %s_\n", "\n Will save: %s\n", "\n Enter save; Esc back\n"
	body := fmt.Sprintf(format, m.domainText)
	if domain, err := m.normalizer.Normalize(m.domainText); err == nil {
		body += fmt.Sprintf(canonical, domain)
	}
	return body + footer
}

func (m Model) windowListView() string {
	const header, footer, format = " Daily schedule\n\n", "\n a add; d delete; ↑↓ select; Esc back\n", "%s to %s — %s"
	const active, waiting = "active", "waiting"
	body := header
	if m.status.State.Lock == nil {
		return body + footer
	}
	for index, window := range m.status.State.Lock.Windows {

		const zero = 0
		label := waiting
		lock := &core.Lock{Kind: core.Daily, Zone: m.status.State.Lock.Zone, Windows: []core.Window{window}}

		if lock.Active(time.Unix(m.status.Now, zero)) {
			label = active
		}
		body += m.row(index, fmt.Sprintf(format, formatMinute(window.Start), formatMinute(window.End), label))
	}
	return body + footer
}

func (m Model) windowEditView() string {
	const (
		textEmpty = ""
	)

	const header, format, appended, footer = " Daily window\n\n", " %s  %s : %s  to  %s : %s\n", " Added: %s to %s\n", "\n ←→ field; ↑↓ adjust; 2 digits advance\n a add window; Enter save; Esc back\n"
	const plain, focus, zero, one, two, three = "%02d", "[%02d]", 0, 1, 2, 3
	var fields [timeFieldCount]string
	for index, value := range m.fields {
		fieldFormat := plain
		if index == m.field {
			fieldFormat = focus
		}
		fields[index] = fmt.Sprintf(fieldFormat, value)
	}
	body := header
	for _, window := range m.windows {
		body += fmt.Sprintf(appended, formatMinute(window.Start), formatMinute(window.End))
	}
	body += fmt.Sprintf(format, textEmpty, fields[zero], fields[one], fields[two], fields[three])
	return body + footer
}

func (m Model) confirmView() string {
	const header, daily, rangeFormat, domains, domainFormat, notice, inputFormat = " Confirm lock\n\n", " Daily: %s to %s\n", " Duration: %s\n Start: %s\n Estimated end: %s\n", " Domains: %d, including subdomains; facebook.com blocks only itself and www\n", "   %s\n", "\n While the confirmed lock is active\n you cannot add, edit or delete anything.\n\n Type the code below (no paste):\n", "\n Code: %s — %s remaining\n Enter confirm; Esc cancel\n"
	const placeholder, separator, codeLength, zero = "_", " ", 4, 0
	if m.pending.Preview == nil {
		return header
	}
	state := m.pending.Preview
	body := header
	if state.Lock.Kind == core.Daily {
		for _, window := range state.Lock.Windows {
			body += fmt.Sprintf(daily, formatMinute(window.Start), formatMinute(window.End))
		}
	} else {
		span := state.Lock.EndAt - state.Lock.StartAt
		body += fmt.Sprintf(rangeFormat, countdown(span), time.Unix(state.Lock.StartAt, zero).Format(time.RFC3339), time.Unix(state.Lock.EndAt, zero).Format(time.RFC3339))
	}
	body += fmt.Sprintf(domains, len(state.Domains))
	for _, domain := range state.Domains {
		body += fmt.Sprintf(domainFormat, domain)
	}
	body += notice + drawCode(m.pending.Code)
	var slots []string
	for index := zero; index < codeLength; index++ {
		char := placeholder
		if index < len(m.codeInput) {
			char = string(m.codeInput[index])
		}
		slots = append(slots, char)
	}
	remaining := m.pending.Now + m.pending.ExpiresIn - m.status.Now
	return body + fmt.Sprintf(inputFormat, strings.Join(slots, separator), countdown(remaining))
}
