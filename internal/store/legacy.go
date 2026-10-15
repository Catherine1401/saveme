package store

import "saveme/internal/core"

// Cấu trúc state v1: mọi domain dùng chung một lock.
type legacyState struct {
	Version int        `json:"version"`
	Domains []string   `json:"domains"`
	Lock    *core.Lock `json:"lock,omitempty"`
	MaxSeen int64      `json:"max_seen"`
}

// Chuyển state v1 sang v2, mỗi domain giữ bản sao lock chung để không mất hiệu lực chặn.
func (l legacyState) migrate() core.State {
	state := core.EmptyState()
	state.MaxSeen = l.MaxSeen
	for _, domain := range l.Domains {
		state.Entries = append(state.Entries, core.Entry{Domain: domain, Lock: l.Lock.Clone()})
	}
	return state
}
