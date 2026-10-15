package enforce

import (
	"testing"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"saveme/internal/core"
)

func TestIsolatedVerifyRejectsChangedRules(t *testing.T) {
	const uid, first, domain = 65534, 0, "example.com"
	isolated(t)
	enforcer := &Linux{ForwardUID: uid, Filter: &core.Filter{}}
	t.Cleanup(func() {
		if err := enforcer.Clear(); err != nil {
			t.Error(err)
		}
	})
	if err := enforcer.Apply([]string{domain}); err != nil {
		t.Fatal(err)
	}
	if err := enforcer.Verify(true); err != nil {
		t.Fatal(err)
	}
	connection := &nftables.Conn{}
	chain := chains(table())[first]
	rules, err := connection.GetRules(table(), chain)
	if err != nil {
		t.Fatal(err)
	}
	rule := rules[first]
	rule.Exprs = []expr.Any{&expr.Verdict{Kind: expr.VerdictAccept}}
	connection.ReplaceRule(rule)
	if err := connection.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := enforcer.Verify(true); err == nil {
		const message = "changed expressions accepted despite unchanged markers"
		t.Fatal(message)
	}
	if err := enforcer.Apply(nil); err != nil {
		t.Fatal(err)
	}
	if err := enforcer.Verify(false); err != nil {
		t.Fatal(err)
	}
	if err := enforcer.Verify(true); err == nil {
		const message = "missing active rules accepted"
		t.Fatal(message)
	}
}
