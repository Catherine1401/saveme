package config

import "testing"

func TestTrustedWindowsStatus(t *testing.T) {
	const (
		synced                = "Leap Indicator: 0(no warning)\r\nStratum: 2 (secondary reference - syncd by (S)NTP)\r\nSource: time.windows.com,0x9\r\n"
		notSynced             = "Leap Indicator: 3(not synchronized)\r\nStratum: 0 (unspecified)\r\nSource: time.windows.com,0x9\r\n"
		localClock            = "Leap Indicator: 3(not synchronized)\r\nStratum: 0 (unspecified)\r\nSource: Local CMOS Clock\r\n"
		freeRunning           = "Leap Indicator: 3(not synchronized)\r\nStratum: 0 (unspecified)\r\nSource: Free-running System Clock\r\n"
		localWithCleanLeap    = "Leap Indicator: 0(no warning)\r\nStratum: 1 (primary reference)\r\nSource: Local CMOS Clock\r\n"
		leapSecond            = "Leap Indicator: 1(positive leap second)\r\nStratum: 1 (primary reference)\r\nSource: time.windows.com,0x9\r\n"
		unsynchronizedStratum = "Leap Indicator: 0(no warning)\r\nStratum: 16 (unsynchronized)\r\nSource: time.windows.com,0x9\r\n"
		missingStratum        = "Leap Indicator: 0(no warning)\r\n"
	)
	cases := []struct {
		name    string
		text    string
		trusted bool
	}{
		{"synced", synced, true},
		{"not synchronized source", notSynced, false},
		{"local clock", localClock, false},
		{"free running", freeRunning, false},
		{"local clock with clean leap", localWithCleanLeap, false},
		{"leap second", leapSecond, true},
		{"unsynchronized stratum", unsynchronizedStratum, false},
		{"missing stratum", missingStratum, false},
		{"empty", "", false},
		{"garbage", "not w32tm output", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := trustedWindowsStatus(tc.text); got != tc.trusted {
				t.Fatalf("trustedWindowsStatus() = %v, want %v", got, tc.trusted)
			}
		})
	}
}
