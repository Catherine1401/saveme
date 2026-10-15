package core

type Kind string

const (
	Daily Kind = "daily"
	Range Kind = "range"
)

type Window struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type Lock struct {
	Kind    Kind     `json:"kind"`
	Zone    string   `json:"zone"`
	Windows []Window `json:"windows,omitempty"`
	StartAt int64    `json:"start_at,omitempty"`
	EndAt   int64    `json:"end_at,omitempty"`
}

type Entry struct {
	Domain string `json:"domain"`
	Lock   *Lock  `json:"lock,omitempty"`
}

type State struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`
	MaxSeen int64   `json:"max_seen"`
}

const StateVersion = 2

func EmptyState() State {
	return State{Version: StateVersion, Entries: []Entry{}}
}

func (l *Lock) Clone() *Lock {
	if l == nil {
		return nil
	}
	lock := *l
	lock.Windows = append([]Window{}, l.Windows...)
	return &lock
}

func (e Entry) Clone() Entry {
	return Entry{Domain: e.Domain, Lock: e.Lock.Clone()}
}

func (s State) Clone() State {
	copy := s
	copy.Entries = make([]Entry, len(s.Entries))
	for index, entry := range s.Entries {
		copy.Entries[index] = entry.Clone()
	}
	return copy
}

type ErrorCode string

const (
	ErrLocked       ErrorCode = "ERR_LOCKED"
	ErrOverlap      ErrorCode = "ERR_OVERLAP"
	ErrInvalid      ErrorCode = "ERR_INVALID"
	ErrHasLock      ErrorCode = "ERR_HAS_LOCK"
	ErrClockUnsync  ErrorCode = "ERR_CLOCK_UNSYNC"
	ErrBadCode      ErrorCode = "ERR_BAD_CODE"
	ErrCodeExpired  ErrorCode = "ERR_CODE_EXPIRED"
	ErrUnauthorized ErrorCode = "ERR_UNAUTHORIZED"
	ErrStorage      ErrorCode = "ERR_STORAGE"
	ErrEnforcement  ErrorCode = "ERR_ENFORCEMENT"
)

func (e ErrorCode) Error() string { return string(e) }
