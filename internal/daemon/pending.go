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

func (e *Engine) candidate(req core.Request, now time.Time) (core.State, error) {
	const zero = 0
	state := e.state.Clone()
	if len(state.Domains) == zero {
		return core.State{}, core.ErrInvalid
	}
	if req.Cmd != core.AddWindow && state.Lock != nil {
		return core.State{}, core.ErrHasLock
	}
	switch req.Cmd {
	case core.CreateDaily:
		if len(req.Windows) == zero {
			return core.State{}, core.ErrInvalid
		}
		state.Lock = &core.Lock{Kind: core.Daily, Zone: e.zone}
		for _, input := range req.Windows {
			window, err := core.ParseWindow(input)
			if err != nil {
				return core.State{}, err
			}
			state.Lock.Windows = append(state.Lock.Windows, window)
		}
		if err := core.ValidateWindows(state.Lock.Windows); err != nil {
			return core.State{}, err
		}
	case core.AddWindow:
		if state.Lock == nil || state.Lock.Kind != core.Daily {
			return core.State{}, core.ErrInvalid
		}
		window, err := core.ParseWindow(core.InputWindow{Start: req.Start, End: req.End})
		if err != nil {
			return core.State{}, err
		}
		state.Lock.Windows = append(state.Lock.Windows, window)
		if err := core.ValidateWindows(state.Lock.Windows); err != nil {
			return core.State{}, err
		}
	case core.CreateRange:
		duration, err := core.Duration(req.Duration)
		if err != nil {
			return core.State{}, err
		}
		state.Lock = &core.Lock{Kind: core.Range, Zone: e.zone, StartAt: now.Unix(), EndAt: now.Add(duration).Unix()}
	default:
		return core.State{}, core.ErrInvalid
	}
	return state, nil
}

func (e *Engine) prepare(req core.Request, owner uint64, now time.Time) core.Response {
	const lifetime = 60

	const empty = ""
	candidate, err := e.candidate(req, now)
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
	e.pending = &Pending{Op: req.Cmd, Candidate: candidate, Code: code, ExpireAt: now.Unix() + lifetime, Owner: owner}
	return e.pendingResponse(now)
}

func (e *Engine) pendingResponse(now time.Time) core.Response {
	preview := e.pending.Candidate.Clone()
	return core.Response{OK: true, Pending: true, Code: e.pending.Code, ExpiresIn: e.pending.ExpireAt - now.Unix(), Failures: e.pending.Failures, Preview: &preview, Now: now.Unix()}
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
	candidate := e.pending.Candidate.Clone()
	if candidate.Lock.Kind == core.Range {
		span := candidate.Lock.EndAt - candidate.Lock.StartAt
		candidate.Lock.StartAt, candidate.Lock.EndAt = now.Unix(), now.Unix()+span
	}
	e.pending = nil
	if err := e.commit(candidate, now); err != nil {
		return failure(err, now)
	}
	return e.status(now)
}
