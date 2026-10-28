package meter

import (
	"net/netip"
	"testing"
)

func tuple(local, remote string) Tuple {
	return Tuple{Local: netip.MustParseAddrPort(local), Remote: netip.MustParseAddrPort(remote)}
}

func TestAssignBySNIIgnoresSharedIP(t *testing.T) {
	const shared, facebook, instagram = "157.240.197.35:443", "facebook.com", "instagram.com"
	table := NewTable()
	table.SetDomains([]string{facebook, instagram})
	insta := tuple("10.0.0.2:50000", shared)
	if got := table.Assign(insta, "www.instagram.com"); got != instagram {
		t.Fatal(got)
	}
	if got := table.Tuples([]string{facebook}); len(got) != 0 {
		t.Fatal("instagram connection counted for facebook", got)
	}
	if got := table.Tuples([]string{instagram}); len(got) != 1 || got[0] != insta {
		t.Fatal(got)
	}
}

func TestAssignFollowsExistingMatchingRules(t *testing.T) {
	table := NewTable()
	table.SetDomains([]string{"facebook.com", "example.com"})
	cases := []struct{ name, want string }{
		{"facebook.com", "facebook.com"},
		{"www.facebook.com", "facebook.com"},
		{"static.xx.fbcdn.net", ""},
		{"m.facebook.com", ""},             // subdomain không liệt kê của facebook.com
		{"cdn.example.com", "example.com"}, // subdomain thường khớp domain cha
		{"notexample.com", ""},
	}
	for index, c := range cases {
		local := netip.AddrPortFrom(netip.MustParseAddr("10.0.0.2"), uint16(40000+index))
		got := table.Assign(Tuple{Local: local, Remote: netip.MustParseAddrPort("1.1.1.1:443")}, c.name)
		if got != c.want {
			t.Fatalf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

func TestSetDomainsDropsRemovedEntriesAndRetainPrunes(t *testing.T) {
	table := NewTable()
	table.SetDomains([]string{"a.com", "b.com"})
	first, second := tuple("10.0.0.2:1", "1.1.1.1:443"), tuple("10.0.0.2:2", "1.1.1.1:443")
	table.Assign(first, "a.com")
	table.Assign(second, "b.com")
	table.SetDomains([]string{"a.com"})
	if _, ok := table.Domain(second); ok {
		t.Fatal("removed domain kept")
	}
	table.Retain(nil)
	if _, ok := table.Domain(first); ok {
		t.Fatal("closed connection kept")
	}
}
