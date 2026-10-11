package ui

import (
	"strings"
	"unicode"

	"saveme/internal/core"
)

func (m Model) domainListKey(key KeyMsg) (Model, Cmd) {
	const add, edit, remove, empty, zero = "a", "e", "d", "", 0
	domains := m.status.State.Domains
	m.selected = move(key.String(), m.selected, len(domains))
	switch key.String() {
	case add:
		m.domainText, m.editingDomain, m.screen = empty, false, domainEdit
	case edit:
		if len(domains) > zero {
			m.domainText, m.editingDomain, m.screen = domains[m.selected], true, domainEdit
		}
	case remove:
		if len(domains) > zero {
			return m.submit(core.Request{Cmd: core.DeleteDomain, Index: &m.selected})
		}
	}
	return m, nil
}

func (m Model) domainKey(key KeyMsg) (Model, Cmd) {
	const enter, backspace, empty, zero, one, maxInput = "enter", "backspace", "", 0, 1, 4096
	switch key.String() {
	case enter:
		domain, err := m.normalizer.Normalize(m.domainText)
		if err != nil {
			const (
				invalidDomain = "Invalid domain."
			)

			m.error = invalidDomain
			return m, nil
		}
		request := core.Request{Cmd: core.AddDomain, Domain: domain}
		if m.editingDomain {
			request.Cmd, request.Index = core.EditDomain, &m.selected
		}
		return m.submit(request)
	case backspace:
		runes := []rune(m.domainText)
		if len(runes) > zero {
			m.domainText = string(runes[:len(runes)-one])
		}
	default:
		if key.Type == KeyRunes {
			text := string(key.Runes)
			if strings.IndexFunc(text, unicode.IsControl) < zero && len(m.domainText)+len(text) <= maxInput {
				m.domainText += text
			}
		}
	}
	if _, err := m.normalizer.Normalize(m.domainText); err != nil {
		const (
			invalidDomain = "Invalid domain."
		)

		m.error = invalidDomain
	} else {
		m.error = empty
	}
	return m, nil
}
