package enforce

import (
	"net/netip"
	"testing"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"saveme/internal/meter"
)

func TestQuotaRulesDropTuplesAndUDP443(t *testing.T) {
	target := quotaTable()
	chain := &nftables.Chain{Name: quotaChainName, Table: target}
	tuples := []meter.Tuple{
		{Local: netip.MustParseAddrPort("10.0.0.2:50000"), Remote: netip.MustParseAddrPort("157.240.1.35:443")},
		{Local: netip.MustParseAddrPort("[2001:db8::2]:50001"), Remote: netip.MustParseAddrPort("[2a03:2880::1]:443")},
	}
	udp := []netip.Addr{netip.MustParseAddr("157.240.1.35"), netip.MustParseAddr("2a03:2880::1")}
	rules := quotaRules(target, chain, tuples, udp)
	if len(rules) != 4 {
		t.Fatal(len(rules))
	}
	markers := []string{tupleMarker, tupleMarker, quicMarker, quicMarker}
	for index, rule := range rules {
		last, ok := rule.Exprs[len(rule.Exprs)-1].(*expr.Verdict)
		if string(rule.UserData) != markers[index] || !ok || last.Kind != expr.VerdictDrop {
			t.Fatal(index, string(rule.UserData))
		}
	}
}

func TestQuotaRulesAreCapped(t *testing.T) {
	var udp []netip.Addr
	for index := range maxQuotaRules + 10 {
		udp = append(udp, netip.AddrFrom4([4]byte{10, 0, byte(index >> 8), byte(index)}))
	}
	if got := len(quotaRules(quotaTable(), &nftables.Chain{}, nil, udp)); got != maxQuotaRules {
		t.Fatal(got)
	}
}

// Cần netns riêng và root: dựng bảng thật rồi gỡ.
func TestQuotaFirewallInstallsAndClearsTable(t *testing.T) {
	isolated(t)
	firewall := &QuotaFirewall{}
	udp := []netip.Addr{netip.MustParseAddr("192.0.2.10")}
	if err := firewall.Drop(nil, udp); err != nil {
		t.Fatal(err)
	}
	found := func() bool {
		tables, err := (&nftables.Conn{}).ListTables()
		if err != nil {
			t.Fatal(err)
		}
		for _, table := range tables {
			if table.Name == quotaTableName {
				return true
			}
		}
		return false
	}
	if !found() {
		t.Fatal("quota table missing")
	}
	if err := firewall.Drop(nil, nil); err != nil || found() {
		t.Fatal("quota table not removed", err)
	}
}
