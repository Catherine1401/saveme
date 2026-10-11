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
		if !validIndex(req.Index, len(state.Domains)) {
			return core.ErrInvalid
		}
		if state.Lock != nil && len(state.Domains) == one {
			return core.ErrInvalid
		}
		state.Domains = append(state.Domains[:*req.Index], state.Domains[*req.Index+one:]...)
		return nil
	case core.DeleteWindow:
		if state.Lock == nil || state.Lock.Kind != core.Daily || !validIndex(req.Index, len(state.Lock.Windows)) {
			return core.ErrInvalid
		}
		if len(state.Lock.Windows) == one {
			if !req.Confirmed {
				return core.ErrInvalid
			}
			state.Lock = nil
			return nil
		}
		state.Lock.Windows = append(state.Lock.Windows[:*req.Index], state.Lock.Windows[*req.Index+one:]...)
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
	if req.Cmd == core.EditDomain && !validIndex(req.Index, len(state.Domains)) {
		return core.ErrInvalid
	}
	for index, existing := range state.Domains {
		if existing == domain && (req.Cmd == core.AddDomain || index != *req.Index) {
			return core.ErrInvalid
		}
	}
	if req.Cmd == core.AddDomain {
		state.Domains = append(state.Domains, domain)
	} else {
		state.Domains[*req.Index] = domain
	}
	return nil
}
