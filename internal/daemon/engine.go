package daemon

import (
	"errors"
	"sync"
	"time"

	"saveme/internal/core"
)

type Repository interface{ Save(core.State) error }
type Enforcer interface{ Apply([]string) error }
type Normalizer interface{ Normalize(string) (string, error) }

type Pending struct {
	Op       core.Command
	Entry    int
	Domain   string
	Targets  []target
	Lock     core.Lock
	Code     string
	ExpireAt int64
	Failures int
	Owner    uint64
}

type Engine struct {
	ClientCount   func() int
	UpdatePending func() bool
	mu            sync.Mutex
	state         core.State
	pending       *Pending
	clock         core.Clock
	repo          Repository
	enforcer      Enforcer
	normalizer    Normalizer
	zone          string
	lastSaved     int64
	health        error
	storageFault  bool
	fixedCode     string
}

func New(state core.State, clock core.Clock, repo Repository, enforcer Enforcer, normalizer Normalizer, zone string) (*Engine, error) {
	if err := state.Validate(); err != nil {
		return nil, err
	}
	if _, err := time.LoadLocation(zone); err != nil {
		return nil, err
	}
	return &Engine{state: state.Clone(), clock: clock, repo: repo, enforcer: enforcer, normalizer: normalizer, zone: zone, lastSaved: state.MaxSeen}, nil
}

func failure(err error, now time.Time) core.Response {
	code := core.ErrInvalid
	var declared core.ErrorCode
	if errors.As(err, &declared) {
		code = declared
	}
	return core.Response{Error: code, Message: err.Error(), Now: now.Unix()}
}

func (e *Engine) Handle(req core.Request, owner uint64) core.Response {
	e.mu.Lock()
	defer e.mu.Unlock()
	now := e.clock.Now()
	refreshErr := e.refresh(now, false)
	if req.Cmd == core.Status {
		return e.status(now)
	}
	if e.UpdatePending != nil && e.UpdatePending() {
		return failure(core.ErrLocked, now)
	}
	if e.targetActive(req, owner, now) {
		return failure(core.ErrLocked, now)
	}
	if !e.clock.Trusted() && req.Cmd != core.AddDomain {
		return failure(core.ErrClockUnsync, now)
	}
	if refreshErr != nil {
		return failure(refreshErr, now)
	}
	if e.health != nil {
		return failure(e.health, now)
	}
	switch req.Cmd {
	case core.CreateDaily, core.CreateRange, core.AddWindow, core.EditWindow:
		return e.prepare(req, owner, now)
	case core.Confirm:
		return e.confirm(req.Code, owner, now)
	case core.Cancel:
		if e.pending != nil && e.pending.Owner == owner {
			e.pending = nil
		}
		return core.Response{OK: true, Now: now.Unix()}
	default:
		candidate := e.state.Clone()
		if err := e.mutate(&candidate, req); err != nil {
			return failure(err, now)
		}
		if err := e.commit(candidate, now); err != nil {
			return failure(err, now)
		}
		e.pending = nil
		return e.status(now)
	}
}

func (e *Engine) status(now time.Time) core.Response {
	const zero, unsynchronized = 0, "Clock not synchronized with NTP."
	response := core.Response{OK: true, Now: now.Unix()}
	if e.ClientCount != nil {
		response.Clients = e.ClientCount()
	}
	if !e.clock.Trusted() {
		response.OK, response.Error, response.Message = false, core.ErrClockUnsync, unsynchronized
	}
	if e.health != nil {
		response.OK = false
		response.Error = core.ErrEnforcement
		response.Message = e.health.Error()
	}
	if e.storageFault {
		response.OK = false
		response.Error = core.ErrStorage
	}
	for _, entry := range e.state.Entries {
		item := e.entryStatus(entry, now)
		response.Active = response.Active || item.Active
		response.Entries = append(response.Entries, item)
	}
	if response.Now < zero {
		response.OK = false
		response.Error = core.ErrClockUnsync
	}
	return response
}

func (e *Engine) entryStatus(entry core.Entry, now time.Time) core.EntryStatus {
	item := core.EntryStatus{Domain: entry.Domain, Active: e.entryActive(entry, now)}
	if entry.Lock == nil {
		return item
	}
	item.Kind, item.Zone = entry.Lock.Kind, entry.Lock.Zone
	transition := entry.Lock.Transition(now)
	if item.Active && e.clock.Trusted() {
		item.StartAt, item.EndAt = entry.Lock.WindowStart(now), transition
	} else if !item.Active {
		item.NextAt = transition
		item.Windows = append([]core.Window{}, entry.Lock.Windows...)
	}
	return item
}

func (e *Engine) commit(candidate core.State, now time.Time) error {
	candidate.MaxSeen = now.Unix()
	if err := candidate.Validate(); err != nil {
		return err
	}
	if err := e.repo.Save(candidate); err != nil {
		e.storageFault, e.health = true, core.ErrStorage
		return core.ErrStorage
	}
	e.state, e.lastSaved, e.health = candidate.Clone(), now.Unix(), nil
	if err := e.enforcer.Apply(e.blocked(candidate, now)); err != nil {
		e.health = core.ErrEnforcement
		return core.ErrEnforcement
	}
	return nil
}

func (e *Engine) refresh(now time.Time, periodic bool) error {
	const saveInterval = 60
	candidate := e.state.Clone()
	expired := false
	for index, entry := range candidate.Entries {
		if e.clock.Trusted() && entry.Lock != nil && entry.Lock.Kind == core.Range && now.Unix() >= entry.Lock.EndAt {
			candidate.Entries[index].Lock, expired = nil, true
		}
	}
	if err := e.enforcer.Apply(e.blocked(candidate, now)); err != nil {
		e.health = core.ErrEnforcement
		return e.health
	}
	if e.storageFault {
		e.health = core.ErrStorage
		return e.health
	}
	e.health = nil
	if expired || (periodic && now.Unix()-e.lastSaved >= saveInterval) {
		candidate.MaxSeen = now.Unix()
		if err := e.repo.Save(candidate); err != nil {
			e.storageFault, e.health = true, core.ErrStorage
			return e.health
		}
		e.state, e.lastSaved = candidate, now.Unix()
	}
	return nil
}

func (e *Engine) Tick() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.refresh(e.clock.Now(), true)
}

// Lock vẫn coi là đang khóa khi đồng hồ chưa tin cậy.
func (e *Engine) entryActive(entry core.Entry, now time.Time) bool {
	return entry.Lock != nil && (!e.clock.Trusted() || entry.Lock.Active(now))
}

func (e *Engine) blocked(state core.State, now time.Time) []string {
	var domains []string
	for _, entry := range state.Entries {
		if e.entryActive(entry, now) {
			domains = append(domains, entry.Domain)
		}
	}
	return domains
}

// Xác định domain đích của lệnh có đang khóa không.
func (e *Engine) targetActive(req core.Request, owner uint64, now time.Time) bool {
	var target *int
	switch req.Cmd {
	case core.EditDomain, core.DeleteDomain:
		target = req.Index
	case core.CreateDaily, core.AddWindow, core.EditWindow, core.DeleteWindow, core.CreateRange:
		target = req.Entry
	case core.Confirm:
		if e.pending != nil && e.pending.Owner == owner {
			target = &e.pending.Entry
		}
	}
	return validIndex(target, len(e.state.Entries)) && e.entryActive(e.state.Entries[*target], now)
}

func (e *Engine) Release(owner uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.pending != nil && e.pending.Owner == owner {
		e.pending = nil
	}
}
