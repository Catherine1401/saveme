//go:build !windows

package config

import (
	"context"
	"os/exec"
	"strings"
)

// systemd báo đồng hồ đã đồng bộ NTP.
func synchronized(ctx context.Context) bool {
	const tool, command, property, flag, yes = "timedatectl", "show", "--property=NTPSynchronized", "--value", "yes"
	output, err := exec.CommandContext(ctx, tool, command, property, flag).Output()
	return err == nil && strings.TrimSpace(string(output)) == yes
}
