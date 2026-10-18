package enforce

import (
	"net"
	"net/netip"
	"strings"
)

func ConfiguredFirewall(upstreams, doh []string, run Runner) (*Firewall, error) {
	firewall := &Firewall{DoH: doh, Run: run}
	for _, value := range upstreams {
		host, _, err := net.SplitHostPort(value)
		if err != nil {
			return nil, err
		}
		address, err := netip.ParseAddr(host)
		if err != nil {
			return nil, err
		}
		firewall.Upstreams = append(firewall.Upstreams, address)
	}
	return firewall, nil
}

// Nhóm độc lập giữ DNS đóng khi daemon cũ hoặc mới dựng lại nhóm rule chính.
func (f *Firewall) Guard(enabled bool) error {
	const main, guard = "saveme", "saveme-update"
	script := clearScript()
	if enabled {
		script = ensureScript(Complement(f.Upstreams), f.DoH)
	}
	_, err := f.Run(strings.ReplaceAll(script, main, guard))
	return err
}
