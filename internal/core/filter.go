package core

import "sync"

type Filter struct {
	mu      sync.RWMutex
	domains []string
	active  bool
}

func (f *Filter) Set(state State, active bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.domains, f.active = append([]string{}, state.Domains...), active
}

func (f *Filter) Blocked(name string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.active && DomainBlocked(name, f.domains)
}
