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

type State struct {
	Version int      `json:"version"`
	Domains []string `json:"domains"`
	Lock    *Lock    `json:"lock,omitempty"`
	MaxSeen int64    `json:"max_seen"`
}

func EmptyState() State {
	const version = 1
	return State{Version: version, Domains: []string{}}
}

func (s State) Clone() State {
	copy := s
	copy.Domains = append([]string{}, s.Domains...)
	if s.Lock != nil {
		lock := *s.Lock
		lock.Windows = append([]Window{}, lock.Windows...)
		copy.Lock = &lock
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
