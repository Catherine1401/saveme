//go:build !windows

package main

import "os"

// Tiến trình đang chạy với quyền root.
func privileged() bool {
	const root = 0
	return os.Geteuid() == root
}

// Linux không có chế độ Windows service.
func runService(options) (bool, error) {
	return false, nil
}
