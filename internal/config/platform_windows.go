package config

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// Named pipe và file cấu hình mặc định của daemon production.
const DefaultSocket, DefaultConfig = `\\.\pipe\saveme`, `C:\ProgramData\saveme\config.json`

// Socket trên Windows phải là named pipe cục bộ.
func validSocket(path string) bool {
	const prefix, empty = `\\.\pipe\`, ""
	return strings.HasPrefix(path, prefix) && strings.TrimPrefix(path, prefix) != empty
}

// Windows yêu cầu SID hợp lệ của một tài khoản thường, không phải SYSTEM.
func (c Config) validIdentity() bool {
	const system = "S-1-5-18"
	sid, err := windows.StringToSid(c.AllowedSID)
	return err == nil && sid.IsValid() && c.AllowedSID != system
}

// Demo dùng pipe và thư mục tạm riêng theo SID của người dùng hiện tại.
func Demo() Config {
	const prefix, pipe, state, zone, empty = "saveme-demo-", `\\.\pipe\`, "state", "UTC", ""
	sid := empty
	if user, err := windows.GetCurrentProcessToken().GetTokenUser(); err == nil {
		sid = user.User.Sid.String()
	}
	dir := filepath.Join(os.TempDir(), prefix+sid)
	return Config{AllowedSID: sid, Socket: pipe + prefix + sid, StateDir: filepath.Join(dir, state), Zone: zone}
}
