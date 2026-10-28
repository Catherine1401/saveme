package core

import "testing"

func TestQuotaValidateBounds(t *testing.T) {
	const day, used, over, below, negative = "2026-10-28", 5, MaxQuotaSec + 1, MinQuotaSec - 1, -1
	cases := []struct {
		quota *Quota
		ok    bool
	}{
		{nil, true},
		{&Quota{LimitSec: MinQuotaSec, UsedSec: used, Day: day}, true},
		{&Quota{LimitSec: MaxQuotaSec, Day: day}, true},
		{&Quota{LimitSec: below, Day: day}, false},
		{&Quota{LimitSec: over, Day: day}, false},
		{&Quota{LimitSec: MinQuotaSec, UsedSec: negative, Day: day}, false},
	}
	for _, c := range cases {
		if got := c.quota.Validate() == nil; got != c.ok {
			t.Fatalf("quota %+v: valid=%v want %v", c.quota, got, c.ok)
		}
	}
}

func TestQuotaExhaustedAndClone(t *testing.T) {
	const limit, changed = MinQuotaSec, 0
	var none *Quota
	if none.Exhausted() || none.Clone() != nil {
		t.Fatal("nil quota must be inactive")
	}
	quota := &Quota{LimitSec: limit, UsedSec: limit - 1}
	if quota.Exhausted() {
		t.Fatal("exhausted before the limit")
	}
	copy := quota.Clone()
	quota.UsedSec = limit
	if !quota.Exhausted() || copy.UsedSec == limit || copy.UsedSec == changed {
		t.Fatal("clone shares state or exhaustion wrong")
	}
}

func TestStateValidateRejectsBadQuota(t *testing.T) {
	const domain = "a.com"
	state := EmptyState()
	state.Entries = []Entry{{Domain: domain, Quota: &Quota{LimitSec: MinQuotaSec - 1}}}
	if state.Validate() != ErrInvalid {
		t.Fatal("bad quota accepted")
	}
}
