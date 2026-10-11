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
	if !m.ready || !m.status.OK {
		if key.String() == shortQuit {
			return m, Quit
		}
		return m, nil
	}
	if m.status.Active {
		if key.String() == shortQuit || key.String() == escape {
			return m, Quit
		}
		return m, nil
	}
	if m.screen == confirmation {
		return m.confirmKey(key)
	}
	if key.String() == escape {
		const (
			number0 = 0
		)

		m.screen, m.selected = mainScreen, number0
		return m, nil
	}
	switch m.screen {
	case mainScreen:
		return m.mainKey(key)
	case windowList:
		return m.windowListKey(key)
	case windowEdit:
		return m.windowKey(key)
	case rangeList:
		return m.rangeKey(key)
	case domainList:
		return m.domainListKey(key)
	case domainEdit:
		return m.domainKey(key)
	case lastWindow:
		const enter = "enter"
		if key.String() == enter {
			return m.submit(core.Request{Cmd: core.DeleteWindow, Index: &m.selected, Confirmed: true})
		}
	}
	return m, nil
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

func (m Model) mainKey(key KeyMsg) (Model, Cmd) {
	const quit, enter, choices, zero, one = "q", "enter", 3, 0, 1
	if key.String() == quit {
		return m, Quit
	}
	m.selected = move(key.String(), m.selected, choices)
	digit, err := strconv.Atoi(key.String())
	choose := key.String() == enter
	if err == nil && digit >= one && digit <= choices {
		m.selected, choose = digit-one, true
	}
	if !choose {
		return m, nil
	}
	switch m.selected {
	case zero:
		if m.status.State.Lock != nil {
			m.screen = windowList
		} else {
			m.beginWindow()
		}
	case one:
		if m.status.State.Lock != nil {
			const (
				removeScheduleFirst = "Delete the daily schedule before creating a timed lock."
			)

			m.error = removeScheduleFirst
			return m, nil
		}
		m.screen = rangeList
	default:
		m.screen = domainList
	}
	m.selected = zero
	return m, nil
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
		return m.submit(core.Request{Cmd: core.CreateRange, Duration: durations[m.selected].Value})
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
