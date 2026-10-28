package daemon

import (
	"fmt"
	"time"

	"saveme/internal/core"
)

const dayLayout = "2006-01-02"

func (e *Engine) location() *time.Location {
	loc, err := time.LoadLocation(e.zone)
	if err != nil {
		return time.UTC
	}
	return loc
}

func (e *Engine) today(now time.Time) string { return now.In(e.location()).Format(dayLayout) }

// Thời điểm 00:00 kế tiếp theo zone của daemon.
func (e *Engine) resetAt(now time.Time) int64 {
	const nextDay = 1
	local := now.In(e.location())
	return time.Date(local.Year(), local.Month(), local.Day()+nextDay, 0, 0, 0, 0, local.Location()).Unix()
}

// Quota hết khi đã dùng đủ trong ngày hiện tại; đồng hồ chưa tin cậy thì giữ nguyên trạng thái đã lưu.
func (e *Engine) quotaExhausted(quota *core.Quota, now time.Time) bool {
	return quota.Exhausted() && (!e.clock.Trusted() || quota.Day == e.today(now))
}

func (e *Engine) quotaStatus(quota *core.Quota, now time.Time) *core.QuotaStatus {
	const zero = 0
	if quota == nil {
		return nil
	}
	used := quota.UsedSec
	if e.clock.Trusted() && quota.Day != e.today(now) {
		used = zero
	}
	return &core.QuotaStatus{LimitSec: quota.LimitSec, UsedSec: used, RemainingSec: max(quota.LimitSec-used, zero), ResetAt: e.resetAt(now)}
}

func (e *Engine) lockOn(lock *core.Lock, now time.Time) bool {
	return lock != nil && (!e.clock.Trusted() || lock.Active(now))
}

// Hợp nhất mốc kết thúc của lock với thời điểm reset khi quota đã hết; mốc 0 của lock nghĩa là không bao giờ kết thúc.
func (e *Engine) mergeQuotaEnd(item *core.EntryStatus, entry core.Entry, now time.Time) {
	const never = 0
	if !item.Active || !e.clock.Trusted() || !e.quotaExhausted(entry.Quota, now) {
		return
	}
	if e.lockOn(entry.Lock, now) && item.EndAt == never {
		return
	}
	item.EndAt = max(item.EndAt, e.resetAt(now))
}

// Xóa quota của entry; áp dụng trực tiếp, không qua mã xác nhận.
func (e *Engine) clearQuota(state *core.State, req core.Request) error {
	if !quotaSupported {
		return errQuotaUnsupported
	}
	if !validIndex(req.Entry, len(state.Entries)) || state.Entries[*req.Entry].Quota == nil {
		return core.ErrInvalid
	}
	state.Entries[*req.Entry].Quota = nil
	return nil
}

var errQuotaUnsupported = fmt.Errorf("%w: quota is not supported on this platform (Linux only)", core.ErrInvalid)

// Phân tích và kiểm giới hạn quota (giây) của lệnh set_quota.
func (e *Engine) quotaLimit(req core.Request) (int64, error) {
	if !quotaSupported {
		return 0, errQuotaUnsupported
	}
	if !validIndex(req.Entry, len(e.state.Entries)) || len(req.Targets) > 0 {
		return 0, core.ErrInvalid
	}
	duration, err := core.Duration(req.Duration)
	if err != nil {
		return 0, err
	}
	limit := int64(duration / time.Second)
	if limit < core.MinQuotaSec || limit > core.MaxQuotaSec {
		return 0, core.ErrInvalid
	}
	return limit, nil
}

// Đặt giới hạn mới; quota đã có thì giữ thời gian đã dùng.
func (e *Engine) applyQuota(entry *core.Entry, limit int64) {
	if entry.Quota == nil {
		entry.Quota = &core.Quota{Day: e.today(e.clock.Now())}
	}
	entry.Quota.LimitSec = limit
}

// Chuẩn bị đặt quota: cần mã xác nhận như lock.
func (e *Engine) prepareQuota(req core.Request, owner uint64, now time.Time) core.Response {
	limit, err := e.quotaLimit(req)
	if err != nil {
		return failure(err, now)
	}
	return e.startPending(&Pending{Op: core.SetQuota, Entry: *req.Entry, Domain: e.state.Entries[*req.Entry].Domain, Quota: limit, Owner: owner}, now)
}

// Xác nhận đặt quota sau khi mã đúng.
func (e *Engine) confirmQuota(pending *Pending, now time.Time) core.Response {
	candidate := e.state.Clone()
	if !validIndex(&pending.Entry, len(candidate.Entries)) || candidate.Entries[pending.Entry].Domain != pending.Domain {
		return failure(core.ErrInvalid, now)
	}
	e.applyQuota(&candidate.Entries[pending.Entry], pending.Quota)
	if err := e.commit(candidate, now); err != nil {
		return failure(err, now)
	}
	return e.status(now)
}
