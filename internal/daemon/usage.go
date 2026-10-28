package daemon

import (
	"slices"
	"time"

	"saveme/internal/core"
)

// Meter báo các domain đang được dùng (có kết nối đang mở) trong danh sách domain có quota; lỗi nghĩa là mất số liệu.
type Meter interface {
	InUse(quota []string) ([]string, error)
}

// Cutter là Meter có thể cắt kết nối của domain đang bị chặn.
type Cutter interface{ Cut(blocked []string) error }

type failedMeter struct{ err error }

func (f failedMeter) InUse([]string) ([]string, error) { return nil, f.err }

// FailedMeter thay cho bộ đo không dựng được: không đếm gì và luôn báo lỗi, nên mọi chặn đang có được giữ nguyên.
func FailedMeter(err error) Meter { return failedMeter{err: err} }

type usageResult struct {
	changed  bool
	mustSave bool
}

// Reset quota sang ngày mới (chỉ khi đồng hồ tin cậy) và cộng một giây cho domain đang dùng; mustSave khi cần lưu trước khi chặn.
func (e *Engine) applyUsage(state *core.State, now time.Time, periodic bool) usageResult {
	const second = 1
	var result usageResult
	if !e.clock.Trusted() {
		return result
	}
	today := e.today(now)
	var inUse []string
	if periodic && e.Meter != nil {
		var err error
		inUse, err = e.Meter.InUse(quotaDomains(*state))
		e.meterFault = err
	}
	for index := range state.Entries {
		entry := &state.Entries[index]
		if entry.Quota == nil {
			continue
		}
		if entry.Quota.Day != today {
			entry.Quota.Day, entry.Quota.UsedSec = today, 0
			result.changed, result.mustSave = true, true
		}
		if e.entryActive(*entry, now) || !slices.Contains(inUse, entry.Domain) {
			continue
		}
		entry.Quota.UsedSec += second
		result.changed = true
		result.mustSave = result.mustSave || entry.Quota.Exhausted()
	}
	return result
}

// Cắt kết nối của domain bị chặn; lỗi bộ đo được ghi lại để status báo ERR_ENFORCEMENT.
func (e *Engine) cutConnections(blocked []string) {
	if cutter, ok := e.Meter.(Cutter); ok {
		if err := cutter.Cut(blocked); err != nil {
			e.meterFault = err
		}
	}
}

// SetMeter gắn bộ đo; gọi sau khi dựng Engine và trước khi phục vụ.
func (e *Engine) SetMeter(meter Meter) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Meter = meter
}

// Domain có quota trong state, dùng làm danh sách cho bộ đo.
func quotaDomains(state core.State) []string {
	var domains []string
	for _, entry := range state.Entries {
		if entry.Quota != nil {
			domains = append(domains, entry.Domain)
		}
	}
	return domains
}
