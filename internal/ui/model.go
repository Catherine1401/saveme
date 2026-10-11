package ui

import (
	"time"

	"saveme/internal/core"
)

type Caller interface {
	Call(core.Request) (core.Response, error)
}
type screen int

const timeFieldCount = 4

const (
	mainScreen screen = iota
	windowList
	windowEdit
	rangeList
	domainList
	domainEdit
	confirmation
	lastWindow
)

type Model struct {
	appearance     appearance
	client         Caller
	normalizer     interface{ Normalize(string) (string, error) }
	status         core.Response
	screen         screen
	previous       screen
	selected       int
	error          string
	busy           bool
	ready          bool
	windows        []core.Window
	fields         [timeFieldCount]int
	field          int
	digits         string
	editingWindow  bool
	editingDomain  bool
	domainText     string
	pending        core.Response
	codeInput      string
	lastChar       time.Time
	charGeneration uint64
}

type reply struct {
	request  core.Request
	response core.Response
	err      error
}
type tick time.Time
type charCommit struct {
	generation uint64
	char       rune
}

func New(client Caller, normalizer interface{ Normalize(string) (string, error) }) Model {
	return Model{client: client, normalizer: normalizer}
}

func (m Model) Init() Cmd {
	return Batch(m.call(core.Request{Cmd: core.Status}), nextTick(), m.animate())
}

func nextTick() Cmd {
	return Tick(time.Second, func(now time.Time) Msg { return tick(now) })
}

func (m Model) call(request core.Request) Cmd {
	return func() Msg {
		response, err := m.client.Call(request)
		return reply{request: request, response: response, err: err}
	}
}

func (m Model) Update(message Msg) (Model, Cmd) {
	const frames, pulse, zero, one = 8, 6, 0, 1
	switch value := message.(type) {
	case Resize:
		m.appearance.width, m.appearance.height = value.Width, value.Height
	case animationFrame:
		m.appearance.frame = (m.appearance.frame + one) % frames
		if m.appearance.flash > zero {
			m.appearance.flash--
		}
		return m, m.animate()
	case tick:
		if m.busy {
			return m, nextTick()
		}
		return m, Batch(m.call(core.Request{Cmd: core.Status}), nextTick())
	case reply:
		return m.receive(value)
	case charCommit:
		const codeLength = 4
		if m.screen == confirmation && value.generation == m.charGeneration && len(m.codeInput) < codeLength {
			m.codeInput += string(value.char)
			m.appearance.flash = pulse
		}
	case KeyMsg:
		return m.key(value)
	}
	return m, nil
}

func (m Model) receive(value reply) (Model, Cmd) {
	const empty, zero = "", 0
	if value.request.Cmd != core.Status {
		m.busy = false
	}
	if value.err != nil {
		m.ready = false
		m.error = value.err.Error()
		return m, nil
	}
	if value.request.Cmd == core.Status {
		wasActive := m.status.Active
		m.status, m.ready = value.response, true
		if !value.response.OK {
			m.error = value.response.Message
		}
		if m.status.Active {
			m.clearConfirmation()
			m.screen = mainScreen
			if !wasActive {
				const pulse = 10
				m.appearance.flash = pulse
				return m, ClearScreen
			}
			return m, nil
		}
		if m.screen == confirmation && m.status.Now >= m.pending.Now+m.pending.ExpiresIn {
			const (
				expiredCode = "Code expired."
			)

			m.error = expiredCode
			return m.cancelPending()
		}
		return m, nil
	}
	if value.response.Pending {
		if m.screen != confirmation {
			m.previous = m.screen
		}
		m.pending, m.screen, m.codeInput = value.response, confirmation, empty
		m.charGeneration++
		if !value.response.OK {
			m.error = string(value.response.Error)
		} else {
			m.error = empty
		}
		return m, ClearScreen
	}
	if !value.response.OK {
		m.error = string(value.response.Error)
		if m.screen == confirmation {
			m.screen = m.previous
			m.clearConfirmation()
			return m, Batch(ClearScreen, m.call(core.Request{Cmd: core.Status}))
		}
		return m, m.call(core.Request{Cmd: core.Status})
	}
	m.error = empty
	if value.response.State != nil || value.response.Active {
		m.status = value.response
	}
	switch value.request.Cmd {
	case core.Confirm:
		const pulse = 10
		m.appearance.flash = pulse
		m.clearConfirmation()
		m.screen, m.selected = mainScreen, zero
	case core.Cancel:
		m.clearConfirmation()
		m.screen = m.previous
	case core.AddDomain, core.EditDomain:
		m.screen = domainList
	case core.DeleteWindow:
		m.screen = windowList
	}
	return m, Batch(ClearScreen, m.call(core.Request{Cmd: core.Status}))
}

func (m *Model) clearConfirmation() {
	const empty = ""
	m.pending, m.codeInput = core.Response{}, empty
	m.charGeneration++
}

func (m Model) cancelPending() (Model, Cmd) {
	m.clearConfirmation()
	m.screen = m.previous
	model, action := m.submit(core.Request{Cmd: core.Cancel})
	return model, Sequence(ClearScreen, action)
}
