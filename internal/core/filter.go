package core

import "sync"

type Filter struct {
	mu      sync.RWMutex
	domains []string
}

func (f *Filter) Set(blocked []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.domains = append([]string{}, blocked...)
}

func (f *Filter) Blocked(name string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return DomainBlocked(name, f.domains)
}
