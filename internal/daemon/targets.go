package daemon

import (
	"slices"
	"time"

	"saveme/internal/core"
)

// Domain bổ sung được khóa cùng lần xác nhận.
type target struct {
	Entry  int
	Domain string
}

// Kiểm các domain bổ sung: chỉ cho lệnh tạo lock, không trùng, không đang khóa, chưa có lock.
func (e *Engine) targets(req core.Request, now time.Time) ([]target, error) {
	const zero = 0
	if len(req.Targets) == zero {
		return nil, nil
	}
	if req.Cmd != core.CreateDaily && req.Cmd != core.CreateRange {
		return nil, core.ErrInvalid
	}
	seen := []int{*req.Entry}
	var result []target
	for _, index := range req.Targets {
		if !validIndex(&index, len(e.state.Entries)) || slices.Contains(seen, index) {
			return nil, core.ErrInvalid
		}
		entry := e.state.Entries[index]
		if e.entryActive(entry, now) {
			return nil, core.ErrLocked
		}
		if entry.Lock != nil {
			return nil, core.ErrHasLock
		}
		seen = append(seen, index)
		result = append(result, target{Entry: index, Domain: entry.Domain})
	}
	return result, nil
}

// Áp cùng một lock cho các domain bổ sung sau khi kiểm lại tên và trạng thái.
func applyTargets(state *core.State, targets []target, lock *core.Lock) error {
	for _, item := range targets {
		index := item.Entry
		if !validIndex(&index, len(state.Entries)) || state.Entries[index].Domain != item.Domain {
			return core.ErrInvalid
		}
		if state.Entries[index].Lock != nil {
			return core.ErrHasLock
		}
		state.Entries[index].Lock = lock.Clone()
	}
	return nil
}
