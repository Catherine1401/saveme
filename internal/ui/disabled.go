package ui

import "fmt"

func (m Model) disabledRow(index int, label, reason string) string {
	const normal, focused, format, one = "×", "› ×", " %s %d  %s [disabled] — %s\n", 1
	marker := normal
	if index == m.selected {
		marker = focused
	}
	return m.ink(fmt.Sprintf(format, marker, index+one, label, reason), toneMuted, false)
}
