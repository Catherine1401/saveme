package ui

import (
	"fmt"

	"saveme/internal/update"
)

type updateStatus struct{ status update.Status }

func (m Model) Updates(observe func() update.Status) Model {
	m.observeUpdates = observe
	return m
}

func (m Model) pollUpdates() Cmd {
	if m.observeUpdates == nil {
		return nil
	}
	return func() Msg { return updateStatus{status: m.observeUpdates()} }
}

func (m Model) updateMessage() string {
	const empty, ready, checking, deferred = "", "ready", "checking", "other_sessions"
	state := m.updateState
	text := empty
	if state.Phase == ready {
		const readyFormat = " Update %s ready; it will be applied on the next launch.\n"
		text = fmt.Sprintf(readyFormat, state.Available)
		if !m.settings.AutoUpdates {
			const disabledFormat = " Update %s ready; auto updates are off.\n"
			text = fmt.Sprintf(disabledFormat, state.Available)
		}
	}
	if state.Phase == checking {
		const downloading = " Checking for updates in the background.\n"
		text = downloading
	}
	if state.Deferred == deferred {
		const deferredText = " Update deferred until other saveme windows close.\n"
		text = deferredText
	}
	if state.LastError != empty {
		const errorFormat = " Last update failed: %s\n"
		return m.ink(fmt.Sprintf(errorFormat, state.LastError), toneWarning, false)
	}
	return m.ink(text, toneMuted, false)
}
