package daemon

import (
	"errors"
	"sync"
	"time"

	"saveme/internal/core"
)

type Repository interface{ Save(core.State) error }
type Enforcer interface{ Apply(core.State, bool) error }
type Normalizer interface{ Normalize(string) (string, error) }

type Pending struct {
	Op        core.Command
	Candidate core.State
	Code      string
	ExpireAt  int64
	Failures  int
	Owner     uint64
}

type Engine struct {
	mu           sync.Mutex
	state        core.State
	pending      *Pending
	clock        core.Clock
	repo         Repository
	enforcer     Enforcer
	normalizer   Normalizer
	zone         string
	lastSaved    int64
	health       error
	storageFault bool
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
	if e.active(e.state, now) {
		return failure(core.ErrLocked, now)
	}
	if !e.clock.Trusted() {
		return failure(core.ErrClockUnsync, now)
	}
	if refreshErr != nil {
		return failure(refreshErr, now)
	}
	if e.health != nil {
		return failure(e.health, now)
	}
	switch req.Cmd {
	case core.CreateDaily, core.CreateRange, core.AddWindow:
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
	const zero, unsynchronized = 0, "Clock not synchronized with NTP by systemd."
	response := core.Response{OK: true, Now: now.Unix(), Active: e.active(e.state, now)}
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
	if e.state.Lock != nil {
		response.Kind = e.state.Lock.Kind
		transition := e.state.Lock.Transition(now)
		if response.Active && e.clock.Trusted() {
			response.EndAt = transition
		} else if !response.Active {
			response.NextAt = transition
		}
	}
	if !response.Active {
		state := e.state.Clone()
		response.State = &state
	}
	if response.Now < zero {
		response.OK = false
		response.Error = core.ErrClockUnsync
	}
	return response
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
	if err := e.enforcer.Apply(candidate, e.active(candidate, now)); err != nil {
		e.health = core.ErrEnforcement
		return core.ErrEnforcement
	}
	return nil
}

func (e *Engine) refresh(now time.Time, periodic bool) error {
	const saveInterval = 60
	candidate := e.state.Clone()
	expired := e.clock.Trusted() && candidate.Lock != nil && candidate.Lock.Kind == core.Range && now.Unix() >= candidate.Lock.EndAt
	if expired {
		candidate.Lock = nil
	}
	if err := e.enforcer.Apply(candidate, e.active(candidate, now)); err != nil {
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

func (e *Engine) active(state core.State, now time.Time) bool {
	return state.Lock != nil && (!e.clock.Trusted() || state.Lock.Active(now))
}

func (e *Engine) Release(owner uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.pending != nil && e.pending.Owner == owner {
		e.pending = nil
	}
}
