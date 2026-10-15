//go:build !windows

package config

import (
	"os"
	"path/filepath"
	"strconv"
)

// Socket và file cấu hình mặc định của daemon production.
const DefaultSocket, DefaultConfig = "/run/saveme.sock", "/etc/saveme/config.json"

// Socket Unix phải là đường dẫn tuyệt đối.
func validSocket(path string) bool {
	return filepath.IsAbs(path)
}

// Linux yêu cầu uid người dùng khác root và tài khoản forwarder.
func (c Config) validIdentity() bool {
	const empty, zero = "", 0
	return c.AllowedUID != nil && *c.AllowedUID != zero && c.ForwardUser != empty
}

func Demo() Config {
	const prefix, socket, state, zone = "saveme-demo-", "saveme.sock", "state", "UTC"
	uid := uint32(os.Getuid())
	dir := filepath.Join(os.TempDir(), prefix+strconv.Itoa(int(uid)))
	return Config{AllowedUID: &uid, Socket: filepath.Join(dir, socket), StateDir: filepath.Join(dir, state), Zone: zone}
}
