package config

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type NTP struct {
	mu      sync.Mutex
	checked time.Time
	trusted bool
}

func (n *NTP) Trusted() bool {
	const interval, timeout, tool, command, property, flag, yes = 10 * time.Second, 2 * time.Second, "timedatectl", "show", "--property=NTPSynchronized", "--value", "yes"
	n.mu.Lock()
	defer n.mu.Unlock()
	if !n.checked.IsZero() && time.Since(n.checked) < interval {
		return n.trusted
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, tool, command, property, flag).Output()
	n.checked, n.trusted = time.Now(), err == nil && strings.TrimSpace(string(output)) == yes
	return n.trusted
}
