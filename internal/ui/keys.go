package ui

import (
	"strconv"
	"strings"
	"time"

	"saveme/internal/core"
)

func (m Model) key(key KeyMsg) (Model, Cmd) {
	const quit, escape, shortQuit = "ctrl+c", "esc", "q"
	if key.String() == quit {
		m.clearConfirmation()
		return m, Sequence(ClearScreen, Quit)
	}
	if m.busy {
		return m, nil
	}
	if !m.ready || !m.usable() {
		if key.String() == shortQuit {
			return m, Quit
		}
		return m, nil
	}
	const vimBack = "h"
	if key.String() == vimBack && !key.Paste && m.screen != mainScreen && m.screen != domainEdit && m.screen != windowEdit && m.screen != confirmation {
		return m.back()
	}
	if key.String() == shortQuit && !key.Paste && m.screen != domainEdit && m.screen != windowEdit && m.screen != confirmation {
		return m.back()
	}
	if m.screen == confirmation {
		m.vim.prefix = false
		return m.confirmKey(key)
	}
	key = m.vimKey(key)
	if key.String() == escape {
		m.screen, m.selected, m.error = mainScreen, m.entry, ""
		return m, nil
	}
	switch m.screen {
	case mainScreen:
		return m.domainListKey(key)
	case domainMenu:
		return m.menuKey(key)
	case windowList:
		return m.windowListKey(key)
	case windowEdit:
		return m.windowKey(key)
	case rangeList:
		return m.rangeKey(key)
	case domainEdit:
		return m.domainKey(key)
	case lastWindow:
		const enter = "enter"
		if key.String() == enter {
			return m.submit(core.Request{Cmd: core.DeleteWindow, Entry: &m.entry, Index: &m.selected, Confirmed: true})
		}
	case domainDelete:
		const enter = "enter"
		if key.String() == enter {
			return m.submit(core.Request{Cmd: core.DeleteDomain, Index: &m.entry, Confirmed: true})
		}
	}
	return m, nil
}

// Giao diện vẫn dùng được khi chỉ thiếu NTP: xem khóa và thêm domain.
func (m Model) usable() bool {
	return m.status.OK || m.status.Error == core.ErrClockUnsync
}

func (m Model) submit(request core.Request) (Model, Cmd) {
	m.busy = true
	return m, m.call(request)
}

func move(key string, selected, count int) int {
	const up, down, zero, one = "up", "down", 0, 1
	if count == zero {
		return zero
	}
	if selected >= count {
		selected = count - one
	}
	if key == up {
		return (selected + count - one) % count
	}
	if key == down {
		return (selected + one) % count
	}
	return selected
}

func (m Model) rangeKey(key KeyMsg) (Model, Cmd) {
	const enter, one = "enter", 1
	durations := durationChoices()
	m.selected = move(key.String(), m.selected, len(durations))
	digit, err := strconv.Atoi(key.String())
	choose := key.String() == enter
	if err == nil && digit >= one && digit <= len(durations) {
		m.selected, choose = digit-one, true
	}
	if choose {
		return m.submit(core.Request{Cmd: core.CreateRange, Entry: &m.entry, Duration: durations[m.selected].Value})
	}
	return m, nil
}

func (m Model) confirmKey(key KeyMsg) (Model, Cmd) {
	const escape, enter, backspace, empty = "esc", "enter", "backspace", ""
	const length, one, zero, burst = 4, 1, 0, 35 * time.Millisecond

	const invalidCharacter = "Use uppercase code letters and digits 2–9."
	switch key.String() {
	case escape:
		return m.cancelPending()
	case enter:
		const (
			incompleteCode = "Type all 4 characters."
		)

		if len(m.codeInput) == length {
			return m.submit(core.Request{Cmd: core.Confirm, Code: m.codeInput})
		}
		m.error = incompleteCode
		return m, nil
	case backspace:
		m.charGeneration++
		if len(m.codeInput) > zero {
			m.codeInput = m.codeInput[:len(m.codeInput)-one]
		}
		return m, nil
	}
	if key.Type != KeyRunes {
		return m, nil
	}
	if key.Paste || len(key.Runes) != one {
		const (
			pasteRejected = "Paste rejected. Type each character."
		)

		m.charGeneration++
		m.codeInput = empty
		m.error = pasteRejected
		return m, nil
	}
	now := time.Now()

	alphabet := core.ConfirmationAlphabet()
	if !strings.ContainsRune(alphabet, key.Runes[zero]) {
		m.error = invalidCharacter
		return m, nil
	}
	fast := !m.lastChar.IsZero() && now.Sub(m.lastChar) < burst
	m.lastChar = now
	m.charGeneration++
	if fast {
		const (
			burstRejected = "Too fast. Type each character slowly."
		)

		m.codeInput = empty
		m.error = burstRejected
		return m, nil
	}
	generation, char := m.charGeneration, key.Runes[zero]
	return m, Tick(burst, func(time.Time) Msg { return charCommit{generation: generation, char: char} })
}
