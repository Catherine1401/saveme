package enforce

import (
	"context"
	"net/netip"
	"os/exec"
	"strings"
	"sync"
	"time"

	"saveme/internal/core"
)

// Chạy script PowerShell và trả stdout.
type Runner func(script string) (string, error)

// Enforcer Windows: NRPT đưa DNS về resolver cục bộ, Firewall chặn DNS ra ngoài, DoT và DoH.
type Firewall struct {
	mu        sync.Mutex
	Filter    *core.Filter
	Upstreams []netip.Addr
	DoH       []string
	Run       Runner
	known     bool
	active    bool
	checked   time.Time
}

// Gọi powershell.exe với timeout; dùng làm Runner mặc định trên Windows.
func PowerShell(script string) (string, error) {
	const shell, noProfile, nonInteractive, policy, bypass, command, timeout = "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, shell, noProfile, nonInteractive, policy, bypass, command, script).Output()
	return string(output), err
}

func (f *Firewall) Apply(blocked []string) error {
	const zero = 0
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Filter.Set(blocked)
	want := len(blocked) > zero
	if f.known && want == f.active && time.Since(f.checked) < firewallRecheck() {
		return nil
	}
	return f.ensure(want)
}

// Áp hoặc gỡ cấu hình; khi đang khóa chỉ dựng lại nếu rule bị thiếu hay bị sửa.
func (f *Firewall) ensure(want bool) error {
	const expected = "6 1"
	if !want {
		if _, err := f.Run(clearScript()); err != nil {
			return err
		}
		f.known, f.active, f.checked = true, false, time.Now()
		return nil
	}
	if f.known && f.active {
		if output, err := f.Run(checkScript()); err == nil && strings.TrimSpace(output) == expected {
			f.checked = time.Now()
			return nil
		}
	}
	if _, err := f.Run(ensureScript(Complement(f.Upstreams), f.DoH)); err != nil {
		f.known = false
		return err
	}
	f.known, f.active, f.checked = true, true, time.Now()
	return nil
}

func (f *Firewall) Clear() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ensure(false)
}

// Chu kỳ kiểm lại rule khi trạng thái khóa không đổi.
func firewallRecheck() time.Duration {
	const interval = 30 * time.Second
	return interval
}
