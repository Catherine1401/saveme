package core

import (
	"testing"
	"time"
)

func TestWindowStartFindsCurrentLockStart(t *testing.T) {
	const zone, hour, night, morning, noon, afternoon = "UTC", 60, 22, 6, 12, 13
	const rangeStart, rangeEnd, zero, day = 1000, 5000, 0, 1440
	const year, month, date, early, late, half, one = 2026, 10, 15, 2, 23, 30, 1
	const afterMidnight, beforeMidnight, inactive, sameDay, rangeCase, wholeDay = "after midnight", "before midnight", "inactive", "same day", "range", "whole day"
	base := time.Date(year, month, date, zero, zero, zero, zero, time.UTC)
	daily := &Lock{Kind: Daily, Zone: zone, Windows: []Window{{Start: night * hour, End: morning * hour}}}
	cases := []struct {
		name string
		lock *Lock
		now  time.Time
		want int64
	}{
		{afterMidnight, daily, base.Add(early * time.Hour), base.Add(-early * time.Hour).Unix()},
		{beforeMidnight, daily, base.Add(late * time.Hour), base.Add(night * time.Hour).Unix()},
		{inactive, daily, base.Add(noon * time.Hour), zero},
		{sameDay, &Lock{Kind: Daily, Zone: zone, Windows: []Window{{Start: noon * hour, End: afternoon * hour}}}, base.Add(noon*time.Hour + half*time.Minute), base.Add(noon * time.Hour).Unix()},
		{rangeCase, &Lock{Kind: Range, StartAt: rangeStart, EndAt: rangeEnd}, time.Unix(rangeStart+one, zero), rangeStart},
		{wholeDay, &Lock{Kind: Daily, Zone: zone, Windows: []Window{{Start: zero, End: day - one}, {Start: day - one, End: zero}}}, base, zero},
	}
	for _, tc := range cases {
		if got := tc.lock.WindowStart(tc.now); got != tc.want {
			t.Fatal(tc.name, got, tc.want)
		}
	}
}
