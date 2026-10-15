//go:build !windows

package store

import "os"

// Flush thư mục để thao tác rename được ghi bền vững.
func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
