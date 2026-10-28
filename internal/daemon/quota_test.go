package daemon

import (
	"errors"
	"strings"
	"testing"
	"time"

	"saveme/internal/core"
)

// Đặt quota qua chuẩn bị và mã xác nhận; trả phản hồi của lần xác nhận.
func setQuota(t *testing.T, f fixture, duration string) core.Response {
	t.Helper()
	const owner, target = 1, 0
	prepared := f.engine.Handle(core.Request{Cmd: core.SetQuota, Entry: at(target), Duration: duration}, owner)
	if !prepared.Pending || prepared.Preview == nil || prepared.Preview.Quota == nil || prepared.Preview.Lock != nil {
		t.Fatal(prepared)
	}
	return f.engine.Handle(core.Request{Cmd: core.Confirm, Code: prepared.Code}, owner)
}

// Dựng fixture với một entry mang quota và lock cho trước.
func quotaFixture(t *testing.T, quota *core.Quota, lock *core.Lock) fixture {
	t.Helper()
	f := newFixture(t)
	f.engine.state.Entries[0].Quota, f.engine.state.Entries[0].Lock = quota, lock
	return f
}

func TestEntryActiveCombinesLockAndQuota(t *testing.T) {
	const limit, today, yesterday = 600, "2026-10-09", "2026-10-08"
	ended := &core.Lock{Kind: core.Range, Zone: "UTC", StartAt: 1, EndAt: 2}
	cases := []struct {
		name    string
		quota   *core.Quota
		lock    *core.Lock
		trusted bool
		active  bool
	}{
		{"quota left, lock ended", &core.Quota{LimitSec: limit, UsedSec: limit - 1, Day: today}, ended, true, false},
		{"quota exhausted, lock ended", &core.Quota{LimitSec: limit, UsedSec: limit, Day: today}, ended, true, true},
		{"quota exhausted yesterday", &core.Quota{LimitSec: limit, UsedSec: limit, Day: yesterday}, nil, true, false},
		{"untrusted keeps exhausted", &core.Quota{LimitSec: limit, UsedSec: limit, Day: yesterday}, nil, false, true},
		{"no quota no lock", nil, nil, true, false},
	}
	for _, c := range cases {
		f := quotaFixture(t, c.quota, c.lock)
		f.clock.trusted = c.trusted
		if got := f.engine.entryActive(f.engine.state.Entries[0], f.clock.now); got != c.active {
			t.Fatalf("%s: active=%v want %v", c.name, got, c.active)
		}
	}
}

func TestStatusReportsQuotaAndResetAt(t *testing.T) {
	const limit, used, today = 1800, 754, "2026-10-09"
	const wantReset = int64(1791590400) // 2026-10-10T00:00:00Z
	f := quotaFixture(t, &core.Quota{LimitSec: limit, UsedSec: used, Day: today}, nil)
	item := f.engine.status(f.clock.now).Entries[0]
	if item.Active || item.Quota == nil || item.Quota.RemainingSec != limit-used || item.Quota.ResetAt != wantReset {
		t.Fatal(item)
	}
	f.engine.state.Entries[0].Quota.UsedSec = limit
	item = f.engine.status(f.clock.now).Entries[0]
	if !item.Active || item.Quota.RemainingSec != 0 || item.EndAt != wantReset || item.StartAt != 0 {
		t.Fatal(item)
	}
	f.engine.state.Entries[0].Quota = nil
	if f.engine.status(f.clock.now).Entries[0].Quota != nil {
		t.Fatal("quota reported without quota")
	}
}

func TestExhaustedQuotaEndsAtLaterOfLockAndReset(t *testing.T) {
	const limit, today = 600, "2026-10-09"
	f := quotaFixture(t, &core.Quota{LimitSec: limit, UsedSec: limit, Day: today}, nil)
	end := f.clock.now.Add(time.Hour).Unix()
	f.engine.state.Entries[0].Lock = &core.Lock{Kind: core.Range, Zone: "UTC", StartAt: f.clock.now.Add(-time.Hour).Unix(), EndAt: end}
	item := f.engine.status(f.clock.now).Entries[0]
	if !item.Active || item.EndAt != f.engine.resetAt(f.clock.now) || item.EndAt <= end {
		t.Fatal(item, end)
	}
}

func TestSetEditClearQuota(t *testing.T) {
	const owner, target, thirty, sixty, used = 1, 0, "30m", "60m", 120
	f := newFixture(t)
	response := setQuota(t, f, thirty)
	quota := response.Entries[target].Quota
	if !response.OK || quota == nil || quota.LimitSec != 30*60 || f.repo.state.Entries[target].Quota == nil {
		t.Fatal(response)
	}
	f.engine.state.Entries[target].Quota.UsedSec = used
	response = setQuota(t, f, sixty)
	if quota = response.Entries[target].Quota; !response.OK || quota.LimitSec != 60*60 || quota.UsedSec != used {
		t.Fatal(response)
	}
	response = f.engine.Handle(core.Request{Cmd: core.ClearQuota, Entry: at(target)}, owner)
	if !response.OK || response.Entries[target].Quota != nil || f.repo.state.Entries[target].Quota != nil {
		t.Fatal(response)
	}
	if response = f.engine.Handle(core.Request{Cmd: core.ClearQuota, Entry: at(target)}, owner); response.Error != core.ErrInvalid {
		t.Fatal(response)
	}
}

func TestSetQuotaRejectsOutOfRangeAndKeepsLock(t *testing.T) {
	const owner, target, today = 1, 0, "2026-10-09"
	f := quotaFixture(t, nil, &core.Lock{Kind: core.Daily, Zone: "UTC", Windows: []core.Window{{Start: 60, End: 120}}})
	for _, value := range []string{"0m", "1441m", "30d", "abc", ""} {
		if response := f.engine.Handle(core.Request{Cmd: core.SetQuota, Entry: at(target), Duration: value}, owner); response.Error != core.ErrInvalid {
			t.Fatal(value, response)
		}
	}
	for _, value := range []string{"1m", "1440m"} {
		if response := setQuota(t, f, value); !response.OK {
			t.Fatal(value, response)
		}
	}
	if entry := f.repo.state.Entries[target]; entry.Lock == nil || entry.Quota == nil || entry.Quota.Day != today {
		t.Fatal(entry)
	}
}

func TestExhaustedQuotaRejectsEveryChange(t *testing.T) {
	const owner, target, limit, today, renamed = 1, 0, 600, "2026-10-09", "other.com"
	f := quotaFixture(t, &core.Quota{LimitSec: limit, UsedSec: limit, Day: today}, nil)
	requests := []core.Request{
		{Cmd: core.EditDomain, Index: at(target), Domain: renamed},
		{Cmd: core.DeleteDomain, Index: at(target)},
		{Cmd: core.SetQuota, Entry: at(target), Duration: "5m"},
		{Cmd: core.ClearQuota, Entry: at(target)},
	}
	for _, req := range requests {
		if response := f.engine.Handle(req, owner); response.Error != core.ErrLocked {
			t.Fatal(req.Cmd, response)
		}
	}
	if len(f.engine.state.Entries) != 1 || f.engine.state.Entries[target].Quota.LimitSec != limit {
		t.Fatal(f.engine.state)
	}
}

func TestSetQuotaStorageFailureDoesNotApply(t *testing.T) {
	const owner, target = 1, 0
	f := newFixture(t)
	prepared := f.engine.Handle(core.Request{Cmd: core.SetQuota, Entry: at(target), Duration: "5m"}, owner)
	f.repo.fail = true
	if response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: prepared.Code}, owner); response.Error != core.ErrStorage {
		t.Fatal(response)
	}
	if f.engine.state.Entries[target].Quota != nil {
		t.Fatal("quota applied despite storage failure")
	}
}

type fakeMeter struct {
	used []string
	err  error
	cut  []string
}

func (m *fakeMeter) InUse([]string) ([]string, error) { return m.used, m.err }
func (m *fakeMeter) Cut(blocked []string) error {
	m.cut = append([]string{}, blocked...)
	return m.err
}

func TestTickCountsOnlyDomainsInUse(t *testing.T) {
	const limit, today, domain = 600, "2026-10-09", "example.com"
	f := quotaFixture(t, &core.Quota{LimitSec: limit, Day: today}, nil)
	meter := &fakeMeter{used: []string{"other.com"}}
	f.engine.Meter = meter
	if err := f.engine.Tick(); err != nil || f.engine.state.Entries[0].Quota.UsedSec != 0 {
		t.Fatal(err, f.engine.state)
	}
	meter.used = []string{domain}
	for range 3 {
		if err := f.engine.Tick(); err != nil {
			t.Fatal(err)
		}
	}
	if f.engine.state.Entries[0].Quota.UsedSec != 3 {
		t.Fatal(f.engine.state.Entries[0].Quota)
	}
}

func TestTickSavesAndBlocksWhenQuotaReached(t *testing.T) {
	const limit, today, domain = 600, "2026-10-09", "example.com"
	f := quotaFixture(t, &core.Quota{LimitSec: limit, UsedSec: limit - 1, Day: today}, nil)
	f.engine.Meter = &fakeMeter{used: []string{domain}}
	if err := f.engine.Tick(); err != nil || f.repo.state.Entries[0].Quota.UsedSec != limit || len(f.enforcer.blocked) != 1 {
		t.Fatal(err, f.repo.state, f.enforcer.blocked)
	}
	// Domain đã active thì không cộng thêm.
	if err := f.engine.Tick(); err != nil || f.engine.state.Entries[0].Quota.UsedSec != limit {
		t.Fatal(err, f.engine.state)
	}
	// Restart giữa ngày: engine mới từ state đã lưu vẫn chặn.
	clone := &fakeEnforcer{}
	restarted, err := New(f.repo.state, f.clock, f.repo, clone, asciiNormalizer{}, "UTC")
	if err != nil || restarted.Tick() != nil || len(clone.blocked) != 1 {
		t.Fatal(err, clone.blocked)
	}
}

func TestExhaustedQuotaStaysBlockedWhenSaveFails(t *testing.T) {
	const limit, today, domain = 600, "2026-10-09", "example.com"
	f := quotaFixture(t, &core.Quota{LimitSec: limit, UsedSec: limit - 1, Day: today}, nil)
	f.engine.Meter = &fakeMeter{used: []string{domain}}
	f.repo.fail = true
	if err := f.engine.Tick(); err == nil || len(f.enforcer.blocked) != 1 {
		t.Fatal(err, f.enforcer.blocked)
	}
}

func TestQuotaResetsAtMidnightOnlyWhenTrusted(t *testing.T) {
	const limit, yesterday = 600, "2026-10-08"
	f := quotaFixture(t, &core.Quota{LimitSec: limit, UsedSec: limit, Day: yesterday}, nil)
	f.clock.trusted = false
	if err := f.engine.Tick(); err != nil || f.engine.state.Entries[0].Quota.UsedSec != limit || len(f.enforcer.blocked) != 1 {
		t.Fatal(err, f.engine.state, f.enforcer.blocked)
	}
	f.clock.trusted = true
	if err := f.engine.Tick(); err != nil {
		t.Fatal(err)
	}
	quota := f.repo.state.Entries[0].Quota
	if quota.UsedSec != 0 || quota.Day != "2026-10-09" || len(f.enforcer.blocked) != 0 {
		t.Fatal(quota, f.enforcer.blocked)
	}
}

func TestQuotaResetsAcrossFakeMidnight(t *testing.T) {
	const limit, today = 600, "2026-10-09"
	f := quotaFixture(t, &core.Quota{LimitSec: limit, UsedSec: limit, Day: today}, nil)
	if err := f.engine.Tick(); err != nil || len(f.enforcer.blocked) != 1 {
		t.Fatal(err, f.enforcer.blocked)
	}
	f.clock.now = time.Date(2026, 10, 10, 0, 0, 1, 0, time.UTC)
	if err := f.engine.Tick(); err != nil || len(f.enforcer.blocked) != 0 || f.engine.state.Entries[0].Quota.UsedSec != 0 {
		t.Fatal(err, f.engine.state, f.enforcer.blocked)
	}
}

func TestQuotaCommandsRejectedWhenPlatformUnsupported(t *testing.T) {
	const owner, target, platform = 1, 0, "not supported"
	previous := quotaSupported
	quotaSupported = false
	defer func() { quotaSupported = previous }()
	f := newFixture(t)
	for _, cmd := range []core.Command{core.SetQuota, core.ClearQuota} {
		response := f.engine.Handle(core.Request{Cmd: cmd, Entry: at(target), Duration: "5m"}, owner)
		if response.Error != core.ErrInvalid || !strings.Contains(response.Message, platform) || f.engine.state.Entries[target].Quota != nil {
			t.Fatal(cmd, response)
		}
	}
}

func TestMeterFailureKeepsBlockAndReportsEnforcement(t *testing.T) {
	const limit, today = 600, "2026-10-09"
	f := quotaFixture(t, &core.Quota{LimitSec: limit, UsedSec: limit, Day: today}, nil)
	f.engine.Meter = &fakeMeter{err: errors.New("sock_diag down")}
	if err := f.engine.Tick(); err != nil || len(f.enforcer.blocked) != 1 {
		t.Fatal(err, f.enforcer.blocked)
	}
	response := f.engine.status(f.clock.now)
	if response.OK || response.Error != core.ErrEnforcement || !response.Entries[0].Active {
		t.Fatal(response)
	}
	f.engine.Meter = &fakeMeter{}
	f.engine.Tick()
	if f.engine.status(f.clock.now).Error == core.ErrEnforcement {
		t.Fatal("fault not cleared after meter recovered")
	}
}

func TestFailedMeterAtStartupKeepsStoredBlock(t *testing.T) {
	const limit, today = 600, "2026-10-09"
	f := quotaFixture(t, &core.Quota{LimitSec: limit, UsedSec: limit, Day: today}, nil)
	f.engine.Meter = FailedMeter(errors.New("no AF_PACKET"))
	if err := f.engine.Tick(); err != nil || len(f.enforcer.blocked) != 1 || f.engine.status(f.clock.now).Error != core.ErrEnforcement {
		t.Fatal(err, f.enforcer.blocked)
	}
}

func TestTickCutsBlockedDomainsAndListsQuotaDomains(t *testing.T) {
	const limit, today = 600, "2026-10-09"
	f := quotaFixture(t, &core.Quota{LimitSec: limit, UsedSec: limit, Day: today}, nil)
	meter := &fakeMeter{}
	f.engine.Meter = meter
	if err := f.engine.Tick(); err != nil || len(meter.cut) != 1 || meter.cut[0] != "example.com" {
		t.Fatal(err, meter.cut)
	}
	if domains := quotaDomains(f.engine.state); len(domains) != 1 || domains[0] != "example.com" {
		t.Fatal(domains)
	}
	meter.err = errors.New("nft failed")
	f.engine.Tick()
	if f.engine.status(f.clock.now).Error != core.ErrEnforcement {
		t.Fatal("cut failure not reported")
	}
}

// Tick giữ khóa Engine khi gọi Meter; Meter nhận danh sách quota qua tham số nên không cần khóa lại.
func TestMeterDoesNotNeedEngineLockDuringTick(t *testing.T) {
	const limit, today = 600, "2026-10-09"
	f := quotaFixture(t, &core.Quota{LimitSec: limit, Day: today}, nil)
	f.engine.Meter = FailedMeter(errors.New("x"))
	done := make(chan struct{})
	go func() { f.engine.Tick(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Tick deadlocked")
	}
}

func TestSetQuotaNeedsCorrectCodeAndStoresNothingBefore(t *testing.T) {
	const owner, other, target = 1, 2, 0
	f := newFixture(t)
	prepared := f.engine.Handle(core.Request{Cmd: core.SetQuota, Entry: at(target), Duration: "5m"}, owner)
	if !prepared.Pending || f.engine.state.Entries[target].Quota != nil || f.repo.state.Entries != nil {
		t.Fatal(prepared, f.engine.state)
	}
	if response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: prepared.Code}, other); response.Error != core.ErrBadCode {
		t.Fatal("other connection confirmed:", response)
	}
	wrong := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: "ZZZZ"}, owner)
	if wrong.Error != core.ErrBadCode || !wrong.Pending || f.engine.state.Entries[target].Quota != nil {
		t.Fatal(wrong)
	}
	if response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: wrong.Code}, owner); !response.OK || response.Entries[target].Quota == nil {
		t.Fatal(response)
	}
}

func TestSetQuotaCodeExpiresAndCancelDiscards(t *testing.T) {
	const owner, target = 1, 0
	f := newFixture(t)
	prepared := f.engine.Handle(core.Request{Cmd: core.SetQuota, Entry: at(target), Duration: "5m"}, owner)
	f.clock.now = f.clock.now.Add(61 * time.Second)
	if response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: prepared.Code}, owner); response.Error != core.ErrCodeExpired {
		t.Fatal(response)
	}
	prepared = f.engine.Handle(core.Request{Cmd: core.SetQuota, Entry: at(target), Duration: "5m"}, owner)
	f.engine.Handle(core.Request{Cmd: core.Cancel}, owner)
	if response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: prepared.Code}, owner); response.Error != core.ErrBadCode || f.engine.state.Entries[target].Quota != nil {
		t.Fatal(response)
	}
}

func TestSetQuotaRejectsTargetsAndBadValuesAtPrepare(t *testing.T) {
	const owner, target = 1, 0
	f := newFixture(t)
	cases := []core.Request{
		{Cmd: core.SetQuota, Entry: at(target), Duration: "5m", Targets: []int{1}},
		{Cmd: core.SetQuota, Entry: at(target), Duration: "1441m"},
		{Cmd: core.SetQuota, Entry: at(5), Duration: "5m"},
		{Cmd: core.SetQuota, Duration: "5m"},
	}
	for _, req := range cases {
		if response := f.engine.Handle(req, owner); response.Pending || response.Error != core.ErrInvalid {
			t.Fatal(req, response)
		}
	}
}
