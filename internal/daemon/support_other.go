//go:build !windows

package daemon

// Linux có bộ đo kết nối nên hỗ trợ quota.
var quotaSupported = true
