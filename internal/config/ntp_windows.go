package config

import (
	"context"
	"os/exec"
)

// Dịch vụ w32time đã đồng bộ xong với nguồn mạng, không chỉ là có nguồn.
func synchronized(ctx context.Context) bool {
	const tool, query, status = "w32tm", "/query", "/status"
	output, err := exec.CommandContext(ctx, tool, query, status).Output()
	return err == nil && trustedWindowsStatus(string(output))
}
