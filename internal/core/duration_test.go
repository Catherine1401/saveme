package core

import (
	"testing"
	"time"
)

func TestCustomDurationBounds(t *testing.T) {
	const one, mixed, last, maximum = "1m", "90m", "1439m", "1440m"
	const singleMinute, mixedMinutes, lastMinutes, maximumMinutes = 1, 90, 1439, 1440
	type scenario struct {
		text    string
		minutes int
	}
	for _, item := range []scenario{{one, singleMinute}, {mixed, mixedMinutes}, {last, lastMinutes}, {maximum, maximumMinutes}} {
		got, err := Duration(item.text)
		if err != nil || got != time.Duration(item.minutes)*time.Minute {
			t.Fatal(item, got, err)
		}
	}
	const empty, noNumber, zero, over, huge = "", "m", "0m", "1441m", "99999999999999999999999999999m"
	const negative, positive, fraction, space, trailing, seconds, mixedUnits = "-1m", "+1m", "1.5m", " 1m", "1m ", "60s", "1h30m"
	for _, text := range []string{empty, noNumber, zero, over, huge, negative, positive, fraction, space, trailing, seconds, mixedUnits} {
		if got, err := Duration(text); err != ErrInvalid {
			t.Fatal(text, got, err)
		}
	}
}
