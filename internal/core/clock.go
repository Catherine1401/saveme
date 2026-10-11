package core

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
	Trusted() bool
}

type WallClock struct {
	mu        sync.Mutex
	base      time.Time
	anchor    time.Time
	wall      func() time.Time
	monotonic func() time.Time
	trusted   func() bool
}

func NewClock(maxSeen int64, wall, monotonic func() time.Time, trusted func() bool) *WallClock {
	const zero = 0
	base := time.Unix(maxSeen, zero)
	if trusted() {
		base = wall().Round(zero)
	}
	if floor := time.Unix(maxSeen, zero); floor.After(base) {
		base = floor
	}
	return &WallClock{base: base, anchor: monotonic(), wall: wall, monotonic: monotonic, trusted: trusted}
}

func (c *WallClock) Now() time.Time {
	const stripMonotonic = 0
	c.mu.Lock()
	defer c.mu.Unlock()
	anchor := c.monotonic()
	now := c.base.Add(anchor.Sub(c.anchor))
	if wall := c.wall().Round(stripMonotonic); c.trusted() && wall.After(now) {
		now = wall
	}
	c.base, c.anchor = now, anchor
	return now
}

func (c *WallClock) Trusted() bool { return c.trusted() }
