package config

import (
	"context"
	"os/exec"
	"strings"
)

// Dịch vụ w32time đang lấy giờ từ nguồn mạng, không phải đồng hồ máy.
func synchronized(ctx context.Context) bool {
	const tool, query, status, local, free = "w32tm", "/query", "/status", "Local CMOS Clock", "Free-running System Clock"
	output, err := exec.CommandContext(ctx, tool, query, status).Output()
	text := string(output)
	return err == nil && !strings.Contains(text, local) && !strings.Contains(text, free)
}
