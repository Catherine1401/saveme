package config

import (
	"context"
	"sync"
	"time"
)

type NTP struct {
	mu      sync.Mutex
	checked time.Time
	trusted bool
}

func (n *NTP) Trusted() bool {
	const interval, timeout = 10 * time.Second, 2 * time.Second
	n.mu.Lock()
	defer n.mu.Unlock()
	if !n.checked.IsZero() && time.Since(n.checked) < interval {
		return n.trusted
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	n.checked, n.trusted = time.Now(), synchronized(ctx)
	return n.trusted
}
