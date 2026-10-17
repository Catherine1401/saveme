package ui

func (m Model) back() (Model, Cmd) {
	const zero, empty = 0, ""
	m.error, m.vim.prefix = empty, false
	switch m.screen {
	case durationEdit:
		m.screen, m.selected = rangeList, len(durationChoices())
	case mainScreen:
		return m, Quit
	case lastWindow:
		m.screen = windowList
	case helpScreen:
		m.screen = mainScreen
	case settingsScreen:
		m.screen = mainScreen
	case pickerScreen:
		m = m.cancelPicker()
	case presetList:
		m.screen, m.selected = domainMenu, zero
		if entry, ok := m.current(); ok && scheduled(entry) {
			m.screen = windowList
		}
	case windowList, rangeList:
		m.screen, m.selected = domainMenu, zero
	default:
		m.screen, m.selected = mainScreen, m.position(m.entry)
	}
	return m, nil
}
