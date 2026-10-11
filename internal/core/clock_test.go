package core

import (
	"testing"
	"time"
)

func TestClockCannotRewind(t *testing.T) {
	const origin, floor, zero, step, forward, rewind = 1000, 1100, 0, 10, 10000, 100
	wall, mono := time.Unix(origin, zero), time.Unix(origin, zero)
	clock := NewClock(floor, func() time.Time { return wall }, func() time.Time { return mono }, func() bool { return true })
	if clock.Now().Unix() != floor {
		const (
			textPersistedFloor = "persisted floor"
		)

		t.Fatal(textPersistedFloor)
	}
	wall = wall.Add(-rewind * time.Hour)
	mono = mono.Add(step * time.Second)
	if clock.Now().Unix() != floor+step {
		const (
			textRewind = "rewind"
		)

		t.Fatal(textRewind)
	}
	wall = time.Unix(forward, zero)
	if clock.Now().Unix() != forward {
		const (
			textForward = "forward"
		)

		t.Fatal(textForward)
	}
	wall = time.Unix(origin, zero)
	mono = mono.Add(step * time.Second)
	if clock.Now().Unix() != forward+step {
		const (
			textForwardFloorNotRetained = "forward floor not retained"
		)

		t.Fatal(textForwardFloorNotRetained)
	}
}

func TestUntrustedWallIsIgnoredUntilNTPSync(t *testing.T) {
	const floor, wrong, synchronized, zero = 1000, 99999999, 2000, 0
	wall, mono, trusted := time.Unix(wrong, zero), time.Unix(floor, zero), false
	clock := NewClock(floor, func() time.Time { return wall }, func() time.Time { return mono }, func() bool { return trusted })
	if clock.Now().Unix() != floor {
		const (
			textUntrustedWallWasAdopted = "untrusted wall was adopted"
		)

		t.Fatal(textUntrustedWallWasAdopted)
	}
	wall, trusted = time.Unix(synchronized, zero), true
	if clock.Now().Unix() != synchronized {
		const (
			textSynchronizedWallNotAdopted = "synchronized wall not adopted"
		)

		t.Fatal(textSynchronizedWallNotAdopted)
	}
}
