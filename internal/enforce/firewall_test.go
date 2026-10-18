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
	const failure, probe, empty = "powershell failed", "Write-Output", ""
	s.scripts = append(s.scripts, script)
	if s.fail {
		return empty, errors.New(failure)
	}
	if strings.Contains(script, probe) {
		return s.check, nil
	}
	return empty, nil
}

func newFirewall(shell *fakeShell) *Firewall {
	const upstream, doh = "1.1.1.1", "8.8.8.8"
	return &Firewall{Filter: &core.Filter{}, Upstreams: []netip.Addr{netip.MustParseAddr(upstream)}, DoH: []string{doh}, Run: shell.run}
}

func TestFirewallAppliesClearsAndSkipsRepeats(t *testing.T) {
	const domain, create, remove, nrpt = "example.com", "New-NetFirewallRule", "Remove-NetFirewallRule", "Add-DnsClientNrptRule"
	const healthy, upstream, rules, first, second, third, one, two = "6 1", "'1.1.1.1'", 6, 0, 1, 2, 1, 2
	shell := &fakeShell{check: healthy}
	firewall := newFirewall(shell)
	if err := firewall.Apply(nil); err != nil || len(shell.scripts) != one || !strings.Contains(shell.scripts[first], remove) {
		t.Fatal(err, shell.scripts)
	}
	if err := firewall.Apply(nil); err != nil || len(shell.scripts) != one {
		t.Fatal(err, shell.scripts)
	}
	if err := firewall.Apply([]string{domain}); err != nil || len(shell.scripts) != two {
		t.Fatal(err, shell.scripts)
	}
	script := shell.scripts[second]
	if strings.Count(script, create) != rules || !strings.Contains(script, nrpt) || strings.Contains(script, upstream) || !firewall.Filter.Blocked(domain) {
		t.Fatal(script)
	}
	if err := firewall.Clear(); err != nil || firewall.active || !strings.Contains(shell.scripts[third], remove) {
		t.Fatal(err, shell.scripts)
	}
}

func TestFirewallRepairsMissingRules(t *testing.T) {
	const domain, create = "example.com", "New-NetFirewallRule"
	const missing, healthy, third, three, four = "5 1", "6 1", 3, 4, 5
	shell := &fakeShell{check: missing}
	firewall := newFirewall(shell)
	if err := firewall.Apply([]string{domain}); err != nil {
		t.Fatal(err)
	}
	firewall.checked = firewall.checked.Add(-firewallRecheck())
	if err := firewall.Apply([]string{domain}); err != nil || len(shell.scripts) != three || !strings.Contains(shell.scripts[third], create) {
		t.Fatal(err, shell.scripts)
	}
	shell.check = healthy
	firewall.checked = firewall.checked.Add(-firewallRecheck())
	if err := firewall.Apply([]string{domain}); err != nil || len(shell.scripts) != four {
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

func TestFirewallPreservesVerifiedRulesOnStartup(t *testing.T) {
	const verified, domain, one, zero, create = "SAVEME_VERIFIED", "example.com", 1, 0, "New-NetFirewallRule"
	shell := &fakeShell{check: verified}
	firewall := newFirewall(shell)
	if err := firewall.Apply([]string{domain}); err != nil || !firewall.active || len(shell.scripts) != one || strings.Contains(shell.scripts[zero], create) {
		t.Fatal(err, shell.scripts)
	}
}
