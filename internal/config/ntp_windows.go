package config

import (
	"context"
	"os/exec"
	"time"
)

// Windows Time must report a completed network sync, or a direct SNTP query
// must confirm the local clock. w32time often stays unsynchronized even when
// NTP is reachable, which used to block every mutation on such hosts.
func synchronized(ctx context.Context) bool {
	if windowsTimeSynchronized(ctx) {
		return true
	}
	return probeSNTP(ctx, defaultNTPServers, time.Now)
}

// windowsTimeSynchronized checks w32time without consuming the whole NTP
// budget: a stuck service must not prevent the direct SNTP fallback.
func windowsTimeSynchronized(ctx context.Context) bool {
	const tool, query, status, budget = "w32tm", "/query", "/status", time.Second
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	output, err := exec.CommandContext(ctx, tool, query, status).Output()
	return err == nil && trustedWindowsStatus(string(output))
}
