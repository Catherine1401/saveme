package enforce

import (
	"errors"
	"net/netip"
	"strings"
	"testing"

	"saveme/internal/core"
)

type fakeShell struct {
	scripts []string
	check   string
	fail    bool
}

func (s *fakeShell) run(script string) (string, error) {
	const failure = "powershell failed"
	s.scripts = append(s.scripts, script)
	if s.fail {
		return "", errors.New(failure)
	}
	if strings.Contains(script, "Write-Output") {
		return s.check, nil
	}
	return "", nil
}

func newFirewall(shell *fakeShell) *Firewall {
	const upstream, doh = "1.1.1.1", "8.8.8.8"
	return &Firewall{Filter: &core.Filter{}, Upstreams: []netip.Addr{netip.MustParseAddr(upstream)}, DoH: []string{doh}, Run: shell.run}
}

func TestFirewallAppliesClearsAndSkipsRepeats(t *testing.T) {
	const domain, create, remove, nrpt = "example.com", "New-NetFirewallRule", "Remove-NetFirewallRule", "Add-DnsClientNrptRule"
	shell := &fakeShell{check: "6 1"}
	firewall := newFirewall(shell)
	if err := firewall.Apply(nil); err != nil || len(shell.scripts) != 1 || !strings.Contains(shell.scripts[0], remove) {
		t.Fatal(err, shell.scripts)
	}
	if err := firewall.Apply(nil); err != nil || len(shell.scripts) != 1 {
		t.Fatal(err, shell.scripts)
	}
	if err := firewall.Apply([]string{domain}); err != nil || len(shell.scripts) != 2 {
		t.Fatal(err, shell.scripts)
	}
	script := shell.scripts[1]
	if strings.Count(script, create) != 6 || !strings.Contains(script, nrpt) || strings.Contains(script, "'1.1.1.1'") || !firewall.Filter.Blocked(domain) {
		t.Fatal(script)
	}
	if err := firewall.Clear(); err != nil || firewall.active || !strings.Contains(shell.scripts[2], remove) {
		t.Fatal(err, shell.scripts)
	}
}

func TestFirewallRepairsMissingRules(t *testing.T) {
	const domain, create = "example.com", "New-NetFirewallRule"
	shell := &fakeShell{check: "5 1"}
	firewall := newFirewall(shell)
	if err := firewall.Apply([]string{domain}); err != nil {
		t.Fatal(err)
	}
	firewall.checked = firewall.checked.Add(-firewallRecheck())
	if err := firewall.Apply([]string{domain}); err != nil || len(shell.scripts) != 3 || !strings.Contains(shell.scripts[2], create) {
		t.Fatal(err, shell.scripts)
	}
	shell.check = "6 1"
	firewall.checked = firewall.checked.Add(-firewallRecheck())
	if err := firewall.Apply([]string{domain}); err != nil || len(shell.scripts) != 4 {
		t.Fatal(err, shell.scripts)
	}
}

func TestFirewallFailureRetriesNextApply(t *testing.T) {
	const domain = "example.com"
	shell := &fakeShell{fail: true}
	firewall := newFirewall(shell)
	if err := firewall.Apply([]string{domain}); err == nil || firewall.known {
		t.Fatal(err)
	}
	shell.fail = false
	if err := firewall.Apply([]string{domain}); err != nil || !firewall.active {
		t.Fatal(err)
	}
}
