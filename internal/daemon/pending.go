package daemon

import (
	"crypto/rand"
	"crypto/subtle"
	"time"

	"saveme/internal/core"
)

func randomCode() (string, error) {
	const size, mask, empty = 4, 31, ""
	alphabet := core.ConfirmationAlphabet()
	var bytes [size]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return empty, err
	}
	for index, value := range bytes {
		bytes[index] = alphabet[value&mask]
	}
	return string(bytes[:]), nil
}

func replacementCode(previous string) (string, error) {
	for {
		code, err := randomCode()
		if err != nil || code != previous {
			return code, err
		}
	}
}

func (e *Engine) candidate(req core.Request, now time.Time) (core.Lock, error) {
	const zero = 0
	if !validIndex(req.Entry, len(e.state.Entries)) {
		return core.Lock{}, core.ErrInvalid
	}
	current := e.state.Entries[*req.Entry].Lock.Clone()
	if req.Cmd != core.AddWindow && req.Cmd != core.EditWindow && current != nil {
		return core.Lock{}, core.ErrHasLock
	}
	switch req.Cmd {
	case core.CreateDaily:
		if len(req.Windows) == zero {
			return core.Lock{}, core.ErrInvalid
		}
		lock := core.Lock{Kind: core.Daily, Zone: e.zone}
		for _, input := range req.Windows {
			window, err := core.ParseWindow(input)
			if err != nil {
				return core.Lock{}, err
			}
			lock.Windows = append(lock.Windows, window)
		}
		return lock, core.ValidateWindows(lock.Windows)
	case core.AddWindow, core.EditWindow:
		if current == nil || current.Kind != core.Daily {
			return core.Lock{}, core.ErrInvalid
		}
		window, err := core.ParseWindow(core.InputWindow{Start: req.Start, End: req.End})
		if err != nil {
			return core.Lock{}, err
		}
		if req.Cmd == core.EditWindow {
			if !validIndex(req.Index, len(current.Windows)) {
				return core.Lock{}, core.ErrInvalid
			}
			current.Windows[*req.Index] = window
		} else {
			current.Windows = append(current.Windows, window)
		}
		return *current, core.ValidateWindows(current.Windows)
	case core.CreateRange:
		duration, err := core.Duration(req.Duration)
		if err != nil {
			return core.Lock{}, err
		}
		return core.Lock{Kind: core.Range, Zone: e.zone, StartAt: now.Unix(), EndAt: now.Add(duration).Unix()}, nil
	default:
		return core.Lock{}, core.ErrInvalid
	}
}

func (e *Engine) prepare(req core.Request, owner uint64, now time.Time) core.Response {
	const lifetime = 60

	const empty = ""
	lock, err := e.candidate(req, now)
	if err != nil {
		return failure(err, now)
	}
	targets, err := e.targets(req, now)
	if err != nil {
		return failure(err, now)
	}

	previous := empty
	if e.pending != nil {
		previous = e.pending.Code
	}
	code, err := replacementCode(previous)
	if err != nil {
		return failure(err, now)
	}
	e.pending = &Pending{Op: req.Cmd, Entry: *req.Entry, Domain: e.state.Entries[*req.Entry].Domain, Targets: targets, Lock: lock, Code: code, ExpireAt: now.Unix() + lifetime, Owner: owner}
	return e.pendingResponse(now)
}

func (e *Engine) pendingResponse(now time.Time) core.Response {
	preview := core.Entry{Domain: e.pending.Domain, Lock: e.pending.Lock.Clone()}
	response := core.Response{OK: true, Pending: true, Code: e.pending.Code, ExpiresIn: e.pending.ExpireAt - now.Unix(), Failures: e.pending.Failures, Preview: &preview, Now: now.Unix()}
	for index, item := range e.pending.Targets {
		const zero = 0
		if index == zero {
			response.Domains = append(response.Domains, e.pending.Domain)
		}
		response.Domains = append(response.Domains, item.Domain)
	}
	return response
}

func (e *Engine) confirm(code string, owner uint64, now time.Time) core.Response {
	const maxFailures, matched = 3, 1
	if e.pending == nil || e.pending.Owner != owner {
		return failure(core.ErrBadCode, now)
	}
	if now.Unix() >= e.pending.ExpireAt {
		e.pending = nil
		return failure(core.ErrCodeExpired, now)
	}
	if subtle.ConstantTimeCompare([]byte(code), []byte(e.pending.Code)) != matched {
		e.pending.Failures++
		if e.pending.Failures >= maxFailures {
			e.pending = nil
			return failure(core.ErrBadCode, now)
		}
		next, err := replacementCode(e.pending.Code)
		if err != nil {
			e.pending = nil
			return failure(err, now)
		}
		e.pending.Code = next
		response := e.pendingResponse(now)
		response.OK, response.Error = false, core.ErrBadCode
		return response
	}
	pending := e.pending
	e.pending = nil
	lock := pending.Lock.Clone()
	if lock.Kind == core.Range {
		span := lock.EndAt - lock.StartAt
		lock.StartAt, lock.EndAt = now.Unix(), now.Unix()+span
	}
	candidate := e.state.Clone()
	if !validIndex(&pending.Entry, len(candidate.Entries)) || candidate.Entries[pending.Entry].Domain != pending.Domain {
		return failure(core.ErrInvalid, now)
	}
	candidate.Entries[pending.Entry].Lock = lock
	if err := applyTargets(&candidate, pending.Targets, lock); err != nil {
		return failure(err, now)
	}
	if err := e.commit(candidate, now); err != nil {
		return failure(err, now)
	}
	return e.status(now)
}
