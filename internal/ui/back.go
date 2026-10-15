package ui

func (m Model) back() (Model, Cmd) {
	const zero, empty = 0, ""
	m.error, m.vim.prefix = empty, false
	switch m.screen {
	case mainScreen:
		return m, Quit
	case lastWindow:
		m.screen = windowList
	case windowList, rangeList:
		m.screen, m.selected = domainMenu, zero
	default:
		m.screen, m.selected = mainScreen, m.entry
	}
	return m, nil
}
