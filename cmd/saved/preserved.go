package main

import (
	"reflect"
	"saveme/internal/core"
	"time"
)

func preservedUpdate(before, after core.State, status core.Response) error {
	const zero = 0
	if !status.OK && status.Error != core.ErrClockUnsync {
		return core.ErrEnforcement
	}
	if after.Version != before.Version || after.MaxSeen < before.MaxSeen || status.Now < after.MaxSeen || len(before.Entries) != len(after.Entries) || len(status.Entries) != len(after.Entries) {
		return core.ErrStorage
	}
	trusted, active := status.OK, false
	now := time.Unix(status.Now, zero)
	for index, prior := range before.Entries {
		entry, observed := after.Entries[index], status.Entries[index]
		if prior.Domain != entry.Domain || observed.Domain != entry.Domain {
			return core.ErrStorage
		}
		expired := trusted && prior.Lock != nil && prior.Lock.Kind == core.Range && status.Now >= prior.Lock.EndAt
		if !reflect.DeepEqual(prior.Lock.Clone(), entry.Lock.Clone()) && !(expired && entry.Lock == nil) {
			return core.ErrLocked
		}
		if err := preservedQuota(prior.Quota, entry.Quota, trusted); err != nil {
			return err
		}
		if err := preservedEntry(entry, observed, now, trusted); err != nil {
			return err
		}
		active = active || observed.Active
	}
	if status.Active != active {
		return core.ErrEnforcement
	}
	return nil
}

// Quota phải giữ nguyên giới hạn, và thời gian đã dùng không giảm trong cùng ngày; sang ngày mới chỉ khi đồng hồ tin cậy.
func preservedQuota(before, after *core.Quota, trusted bool) error {
	if before == nil || after == nil {
		if before != after {
			return core.ErrLocked
		}
		return nil
	}
	switch {
	case after.LimitSec != before.LimitSec, after.Day < before.Day:
		return core.ErrLocked
	case after.Day == before.Day && after.UsedSec < before.UsedSec:
		return core.ErrLocked
	case after.Day > before.Day && !trusted:
		return core.ErrLocked
	}
	return nil
}

// Hết quota theo status daemon báo: còn lại 0 giây.
func quotaExhausted(observed core.EntryStatus) bool {
	const none = 0
	return observed.Quota != nil && observed.Quota.RemainingSec == none
}

func preservedEntry(entry core.Entry, observed core.EntryStatus, now time.Time, trusted bool) error {
	expected := core.EntryStatus{Domain: entry.Domain}
	exhausted := quotaExhausted(observed)
	if (entry.Quota == nil) != (observed.Quota == nil) || (entry.Quota != nil && observed.Quota.LimitSec != entry.Quota.LimitSec) {
		return core.ErrLocked
	}
	if entry.Lock != nil {
		expected.Kind, expected.Zone = entry.Lock.Kind, entry.Lock.Zone
		expected.Active = !trusted || entry.Lock.Active(now)
		if expected.Active && trusted {
			expected.StartAt, expected.EndAt = entry.Lock.WindowStart(now), entry.Lock.Transition(now)
		} else if !expected.Active {
			expected.NextAt, expected.Windows = entry.Lock.Transition(now), entry.Lock.Windows
		}
	}
	if exhausted {
		// Hết quota thì entry phải active; các mốc thời gian còn phụ thuộc thời điểm reset nên không so sánh.
		expected.Active, expected.StartAt, expected.EndAt, expected.NextAt = true, 0, 0, 0
		observed.StartAt, observed.EndAt, observed.NextAt = 0, 0, 0
	}
	observed.Quota = nil
	expected.Windows = append([]core.Window{}, expected.Windows...)
	observed.Windows = append([]core.Window{}, observed.Windows...)
	if !reflect.DeepEqual(expected, observed) {
		return core.ErrLocked
	}
	return nil
}

func activeDomains(state core.State, status core.Response) []string {
	var domains []string
	for index, entry := range state.Entries {
		const zero = 0
		lockOn := entry.Lock != nil && (!status.OK || entry.Lock.Active(time.Unix(status.Now, zero)))
		if lockOn || (index < len(status.Entries) && quotaExhausted(status.Entries[index])) {
			domains = append(domains, entry.Domain)
		}
	}
	return domains
}
