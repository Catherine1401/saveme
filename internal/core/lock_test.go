package core

import (
	"testing"
	"time"
)

func instant(t *testing.T, text string) time.Time {
	t.Helper()
	value, err := time.Parse(time.RFC3339, text)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestOvernightBoundaries(t *testing.T) {
	const (
		textBeforeMidnight   = "before-midnight"
		text20261009T235900Z = "2026-10-09T23:59:00Z"
		textMidnight         = "midnight"
		text20261010T000000Z = "2026-10-10T00:00:00Z"
		textLastMinute       = "last-minute"
		text20261010T055959Z = "2026-10-10T05:59:59Z"
		textEndExclusive     = "end-exclusive"
		text20261010T060000Z = "2026-10-10T06:00:00Z"
	)

	const zone, start, end = "UTC", 22 * 60, 6 * 60
	lock := &Lock{Kind: Daily, Zone: zone, Windows: []Window{{Start: start, End: end}}}
	cases := []struct {
		name   string
		at     string
		active bool
	}{
		{name: textBeforeMidnight, at: text20261009T235900Z, active: true},
		{name: textMidnight, at: text20261010T000000Z, active: true},
		{name: textLastMinute, at: text20261010T055959Z, active: true},
		{name: textEndExclusive, at: text20261010T060000Z, active: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := lock.Active(instant(t, tc.at)); got != tc.active {
				const (
					textActiveVWantV = "active = %v, want %v"
				)

				t.Fatalf(textActiveVWantV, got, tc.active)
			}
		})
	}
}

func TestOverlap(t *testing.T) {
	const (
		textTouching  = "touching"
		number600     = 600
		number660     = 660
		number720     = 720
		textPartial   = "partial"
		number650     = 650
		textOvernight = "overnight"
		number1320    = 1320
		number360     = 360
		number300     = 300
		number420     = 420
		textEqual     = "equal"
		number60      = 60
	)

	cases := []struct {
		name    string
		windows []Window
		want    error
	}{
		{name: textTouching, windows: []Window{{Start: number600, End: number660}, {Start: number660, End: number720}}},
		{name: textPartial, windows: []Window{{Start: number600, End: number660}, {Start: number650, End: number720}}, want: ErrOverlap},
		{name: textOvernight, windows: []Window{{Start: number1320, End: number360}, {Start: number300, End: number420}}, want: ErrOverlap},
		{name: textEqual, windows: []Window{{Start: number60, End: number60}}, want: ErrInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidateWindows(tc.windows); got != tc.want {
				const (
					textGotVWantV = "got %v, want %v"
				)

				t.Fatalf(textGotVWantV, got, tc.want)
			}
		})
	}
}

func TestTransitions(t *testing.T) {
	const (
		textTouching         = "touching"
		textUTC              = "UTC"
		number600            = 600
		number660            = 660
		number720            = 720
		text20261009T103000Z = "2026-10-09T10:30:00Z"
		text20261009T120000Z = "2026-10-09T12:00:00Z"
		textSpring           = "spring"
		textAmericaNewYork   = "America/New_York"
		number60             = 60
		number180            = 180
		text20260308T063000Z = "2026-03-08T06:30:00Z"
		text20260308T070000Z = "2026-03-08T07:00:00Z"
		textAutumn           = "autumn"
		number120            = 120
		text20261101T053000Z = "2026-11-01T05:30:00Z"
		text20261101T070000Z = "2026-11-01T07:00:00Z"
	)

	cases := []struct {
		name string
		lock Lock
		now  string
		end  string
	}{
		{name: textTouching, lock: Lock{Kind: Daily, Zone: textUTC, Windows: []Window{{Start: number600, End: number660}, {Start: number660, End: number720}}}, now: text20261009T103000Z, end: text20261009T120000Z},
		{name: textSpring, lock: Lock{Kind: Daily, Zone: textAmericaNewYork, Windows: []Window{{Start: number60, End: number180}}}, now: text20260308T063000Z, end: text20260308T070000Z},
		{name: textAutumn, lock: Lock{Kind: Daily, Zone: textAmericaNewYork, Windows: []Window{{Start: number60, End: number120}}}, now: text20261101T053000Z, end: text20261101T070000Z},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got, want := tc.lock.Transition(instant(t, tc.now)), instant(t, tc.end).Unix(); got != want {
				const (
					textGotDWantD = "got %d, want %d"
				)

				t.Fatalf(textGotDWantD, got, want)
			}
		})
	}
}

func TestContinuousAndRange(t *testing.T) {
	const (
		number720 = 720
	)

	const zone, nowText, zero, rangeSeconds = "UTC", "2026-10-09T12:00:00Z", 0, 60
	now := instant(t, nowText)
	continuous := Lock{Kind: Daily, Zone: zone, Windows: []Window{{Start: zero, End: number720}, {Start: number720, End: zero}}}
	if !continuous.Active(now) || continuous.Transition(now) != zero {
		const (
			textFullDayLockMustHaveNoTransition = "full-day lock must have no transition"
		)

		t.Fatal(textFullDayLockMustHaveNoTransition)
	}
	rangeLock := Lock{Kind: Range, Zone: zone, StartAt: now.Unix(), EndAt: now.Unix() + rangeSeconds}
	if !rangeLock.Active(now) || rangeLock.Active(now.Add(rangeSeconds*time.Second)) {
		const (
			textRangeBoundaries = "range boundaries"
		)

		t.Fatal(textRangeBoundaries)
	}
}

func TestDurationsAndMinutes(t *testing.T) {
	const (
		text12h           = "12h"
		text1d            = "1d"
		text3d            = "3d"
		text7d            = "7d"
		text14d           = "14d"
		text30d           = "30d"
		text0d            = "0d"
		text31d           = "31d"
		text24h           = "24h"
		text1             = "1"
		text1dValue       = "-1d"
		text2400          = "24:00"
		text1260          = "12:60"
		text100           = "-1:00"
		text100Value      = "+1:00"
		text100ValueValue = "1:00"
		textAaBb          = "aa:bb"
	)

	const zero = 0
	valid := []string{text12h, text1d, text3d, text7d, text14d, text30d}
	for _, duration := range valid {
		if value, err := Duration(duration); err != nil || value <= zero {
			t.Fatal(duration, value, err)
		}
	}
	invalid := []string{text0d, text31d, text24h, text1, text1dValue}
	for _, duration := range invalid {
		if _, err := Duration(duration); err == nil {
			t.Fatal(duration)
		}
	}
	minutes := []string{text2400, text1260, text100, text100Value, text100ValueValue, textAaBb}
	for _, value := range minutes {
		if _, err := ParseMinute(value); err == nil {
			t.Fatal(value)
		}
	}
}
