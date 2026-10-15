package daemon

import "saveme/internal/core"

func validIndex(index *int, length int) bool {
	const zero = 0
	return index != nil && *index >= zero && *index < length
}

func (e *Engine) mutate(state *core.State, req core.Request) error {
	const one = 1
	switch req.Cmd {
	case core.AddDomain, core.EditDomain:
		return e.changeDomain(state, req)
	case core.DeleteDomain:
		if !validIndex(req.Index, len(state.Entries)) {
			return core.ErrInvalid
		}
		if state.Entries[*req.Index].Lock != nil && !req.Confirmed {
			return core.ErrInvalid
		}
		state.Entries = append(state.Entries[:*req.Index], state.Entries[*req.Index+one:]...)
		return nil
	case core.DeleteWindow:
		if !validIndex(req.Entry, len(state.Entries)) {
			return core.ErrInvalid
		}
		entry := &state.Entries[*req.Entry]
		if entry.Lock == nil || entry.Lock.Kind != core.Daily || !validIndex(req.Index, len(entry.Lock.Windows)) {
			return core.ErrInvalid
		}
		if len(entry.Lock.Windows) == one {
			if !req.Confirmed {
				return core.ErrInvalid
			}
			entry.Lock = nil
			return nil
		}
		entry.Lock.Windows = append(entry.Lock.Windows[:*req.Index], entry.Lock.Windows[*req.Index+one:]...)
		return nil
	default:
		return core.ErrInvalid
	}
}

func (e *Engine) changeDomain(state *core.State, req core.Request) error {
	domain, err := e.normalizer.Normalize(req.Domain)
	if err != nil {
		return err
	}
	if req.Cmd == core.EditDomain && !validIndex(req.Index, len(state.Entries)) {
		return core.ErrInvalid
	}
	for index, existing := range state.Entries {
		if existing.Domain == domain && (req.Cmd == core.AddDomain || index != *req.Index) {
			return core.ErrInvalid
		}
	}
	if req.Cmd == core.AddDomain {
		state.Entries = append(state.Entries, core.Entry{Domain: domain})
	} else {
		state.Entries[*req.Index].Domain = domain
	}
	return nil
}
