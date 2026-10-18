package ui

import (
	"strings"
	"time"

	"saveme/internal/core"
	"saveme/internal/update"
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
	domainEdit
	confirmation
	lastWindow
	domainMenu
	domainLocked
	domainDelete
	helpScreen
	settingsScreen
	pickerScreen
	presetList
	durationEdit
)

type Model struct {
	observeUpdates func() update.Status
	updateState    update.Status
	vim            vimMode
	appearance     appearance
	client         Caller
	normalizer     interface{ Normalize(string) (string, error) }
	status         core.Response
	screen         screen
	previous       screen
	selected       int
	entry          int
	error          string
	busy           bool
	ready          bool
	windows        []core.Window
	fields         [timeFieldCount]int
	field          int
	digits         string
	editingWindow  bool
	duration       durationInput
	editingDomain  bool
	domainText     string
	pending        core.Response
	codeInput      string
	lastChar       time.Time
	charGeneration uint64
	marked         []string
	keycast        bool
	castLabel      string
	castAt         time.Time
	settings       Settings
	saveSettings   func(Settings) error
	picking        int
	draft          []string
	clockText      string
	previousClock  string
	changedAt      time.Time
	searching      bool
	filter         string
	seenError      string
	seenAt         time.Time
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
	return Batch(m.call(core.Request{Cmd: core.Status}), nextTick(), m.animate(), m.pollUpdates())
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
	case updateStatus:
		m.updateState = value.status
	case Resize:
		m.appearance.width, m.appearance.height = value.Width, value.Height
	case animationFrame:
		m.appearance.now = time.Time(value)
		m.appearance.frame = (m.appearance.frame + one) % frames
		if m.appearance.flash > zero {
			m.appearance.flash--
		}
		return m, m.animate()
	case tick:
		m = m.expireError(time.Time(value))
		if m.busy {
			return m, nextTick()
		}
		return m, Batch(m.call(core.Request{Cmd: core.Status}), nextTick(), m.pollUpdates())
	case reply:
		return m.receive(value)
	case charCommit:
		const codeLength = 4
		if m.screen == confirmation && value.generation == m.charGeneration && len(m.codeInput) < codeLength {
			m.codeInput += string(value.char)
			m.appearance.flash = pulse
		}
	case KeyMsg:
		return m.recordKey(value, time.Now()).key(value)
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
		m.status, m.ready = value.response, true
		if !value.response.OK && value.response.Error != core.ErrClockUnsync {
			m.error = value.response.Message
		}
		m = m.trackClock(time.Now())
		if model, action, changed := m.follow(); changed {
			return model, action
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
	if !succeeded(value) {
		m.error = string(value.response.Error)
		if m.screen == confirmation {
			m.screen = m.previous
			m.clearConfirmation()
			return m, Batch(ClearScreen, m.call(core.Request{Cmd: core.Status}))
		}
		return m, m.call(core.Request{Cmd: core.Status})
	}
	m.error = empty
	if value.request.Cmd != core.Cancel {
		m.status = value.response
	}
	switch value.request.Cmd {
	case core.Confirm:
		const pulse = 10
		m.appearance.flash = pulse
		m.clearConfirmation()
		m.screen, m.selected, m.marked = mainScreen, m.position(m.entry), nil
	case core.Cancel:
		m.clearConfirmation()
		m.screen = m.previous
	case core.AddDomain, core.EditDomain, core.DeleteDomain:
		m.screen, m.selected = mainScreen, move(empty, m.selected, len(m.order()))
	case core.DeleteWindow:
		m.screen = windowList
	}
	return m, Batch(ClearScreen, m.call(core.Request{Cmd: core.Status}))
}

// Thêm domain khi NTP chưa tin cậy vẫn thành công và trả status kèm ERR_CLOCK_UNSYNC.
func succeeded(value reply) bool {
	const zero = 0
	unsynchronized := value.request.Cmd == core.AddDomain && value.response.Error == core.ErrClockUnsync && len(value.response.Entries) > zero
	return value.response.OK || unsynchronized
}

// Đồng bộ màn hình theo trạng thái khóa mới nhất của domain đang thao tác.
func (m Model) follow() (Model, Cmd, bool) {
	const zero, pulse = 0, 10
	scoped := m.screen != mainScreen && m.screen != domainEdit && m.screen != helpScreen && m.screen != settingsScreen && m.screen != pickerScreen
	entry, ok := m.current()
	switch {
	case scoped && !ok:
		m.clearConfirmation()
		m.screen, m.selected = mainScreen, zero
	case scoped && entry.Active && m.screen != domainLocked:
		m.clearConfirmation()
		m.screen, m.appearance.flash = domainLocked, pulse
	case m.screen == domainLocked && !entry.Active:
		m.screen, m.selected = domainMenu, zero
	default:
		return m, nil, false
	}
	return m, ClearScreen, true
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

// Ẩn lỗi thao tác sau vài giây; lỗi trạng thái daemon thì giữ nguyên.
func (m Model) expireError(now time.Time) Model {
	const lifetime, empty = 5 * time.Second, ""
	if m.error == empty || (!m.status.OK && m.error == m.status.Message) {
		return m
	}
	if m.error != m.seenError {
		m.seenError, m.seenAt = m.error, now
		return m
	}
	if now.Sub(m.seenAt) >= lifetime {
		m.error, m.seenError = empty, empty
	}
	return m
}

// Ghi nhớ chuỗi giờ trước đó và lúc đổi để hiệu ứng trượt chữ số.
func (m Model) trackClock(now time.Time) Model {
	const zero, separator, last = 0, " ", 1
	entry, ok := m.current()
	if !ok || entry.EndAt == zero {
		return m
	}
	parts := strings.Split(countdown(entry.EndAt-m.status.Now), separator)
	if text := parts[len(parts)-last]; text != m.clockText {
		m.previousClock, m.clockText, m.changedAt = m.clockText, text, now
	}
	return m
}
