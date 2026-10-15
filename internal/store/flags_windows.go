package store

import "os"

type Immutable struct{}

// Bật/tắt thuộc tính read-only của file; Windows ánh xạ quyền ghi sang thuộc tính này.
func (Immutable) Set(path string, enabled bool) error {
	const readOnly, writable = 0444, 0600
	if enabled {
		return os.Chmod(path, readOnly)
	}
	return os.Chmod(path, writable)
}
