package ui

type vimMode struct {
	prefix bool
}

func (m *Model) vimKey(key KeyMsg) KeyMsg {
	const down, up, left, right, first, last = "j", "k", "h", "l", "g", "G"
	const downKey, upKey, leftKey, rightKey, enterKey, editKey, empty, one, zero = "down", "up", "left", "right", "enter", "e", "", 1, 0
	if m.screen == domainEdit || key.Paste || key.Type != KeyRunes || len(key.Runes) != one {
		m.vim.prefix = false
		return key
	}
	text := key.String()
	previous := m.vim.prefix
	m.vim.prefix = false
	switch text {
	case down:
		return KeyMsg{Name: downKey, Type: KeyControl}
	case up:
		return KeyMsg{Name: upKey, Type: KeyControl}
	case left, right:
		if m.screen == windowEdit {
			name := leftKey
			if text == right {
				name = rightKey
			}
			return KeyMsg{Name: name, Type: KeyControl}
		}
		if text == right {
			if m.screen == windowList {
				return KeyMsg{Type: KeyRunes, Runes: []rune(editKey)}
			}
			return KeyMsg{Name: enterKey, Type: KeyControl}
		}
	case first, last:
		count := m.listCount()
		if count > zero {
			if text == last {
				m.selected = count - one
			} else if previous {
				m.selected = zero
			} else {
				m.vim.prefix = true
			}
			return KeyMsg{Name: empty, Type: KeyControl}
		}
	}
	return key
}

func (m Model) listCount() int {
	const menuChoices, zero = 2, 0
	switch m.screen {
	case mainScreen:
		return len(m.status.Entries)
	case domainMenu:
		return menuChoices
	case rangeList:
		return len(durationChoices())
	case windowList:
		entry, _ := m.current()
		return len(entry.Windows)
	}
	return zero
}
