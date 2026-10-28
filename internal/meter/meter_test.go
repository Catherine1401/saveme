package meter

import (
	"errors"
	"net/netip"
	"slices"
	"testing"
)

type fakeSource struct {
	live      []Tuple
	err       error
	destroyed []Tuple
}

func (f *fakeSource) Established() ([]Tuple, error) { return f.live, f.err }
func (f *fakeSource) Destroy(t Tuple) error         { f.destroyed = append(f.destroyed, t); return nil }

type fakeFirewall struct {
	tuples []Tuple
	udp    []netip.Addr
	err    error
}

func (f *fakeFirewall) Drop(tuples []Tuple, udp []netip.Addr) error {
	f.tuples, f.udp = tuples, udp
	return f.err
}

func newMeter(source *fakeSource, firewall *fakeFirewall, domains ...string) *Meter {
	m := &Meter{Table: NewTable(), Source: source, Firewall: firewall}
	m.Table.SetDomains(domains)
	return m
}

func TestInUseCountsEachDomainOnItsOwnConnections(t *testing.T) {
	const shared = "157.240.197.35:443"
	insta, messenger := tuple("10.0.0.2:1", shared), tuple("10.0.0.2:2", shared)
	other := tuple("10.0.0.2:3", "8.8.8.8:443")
	source := &fakeSource{live: []Tuple{insta, messenger, other}}
	m := newMeter(source, &fakeFirewall{}, "instagram.com", "messenger.com")
	m.Table.Assign(insta, "www.instagram.com")
	m.Table.Assign(other, "dns.google")
	used, err := use(m)
	if err != nil || !slices.Equal(used, []string{"instagram.com"}) {
		t.Fatal(used, err)
	}
	m.Table.Assign(messenger, "www.messenger.com")
	if used, _ = use(m); !slices.Equal(used, []string{"instagram.com", "messenger.com"}) {
		t.Fatal(used)
	}
}

func TestInUseStopsWhenConnectionCloses(t *testing.T) {
	open := tuple("10.0.0.2:1", "1.1.1.1:443")
	source := &fakeSource{live: []Tuple{open}}
	m := newMeter(source, &fakeFirewall{}, "example.com")
	m.Table.Assign(open, "example.com")
	if used, _ := use(m); len(used) != 1 {
		t.Fatal(used)
	}
	source.live = nil
	if used, err := use(m); err != nil || len(used) != 0 {
		t.Fatal(used, err)
	}
}

func TestInUseReportsSourceFailure(t *testing.T) {
	failure := errors.New("sock_diag down")
	m := newMeter(&fakeSource{err: failure}, &fakeFirewall{}, "example.com")
	if used, err := use(m); !errors.Is(err, failure) || used != nil {
		t.Fatal(used, err)
	}
}

func TestCutDestroysAssignedConnectionsAndDropsUDP(t *testing.T) {
	blocked, free := tuple("10.0.0.2:1", "1.1.1.1:443"), tuple("10.0.0.2:2", "2.2.2.2:443")
	source, firewall := &fakeSource{live: []Tuple{blocked, free}}, &fakeFirewall{}
	m := newMeter(source, firewall, "a.com", "b.com")
	m.Table.Assign(blocked, "a.com")
	m.Table.Assign(free, "b.com")
	m.Table.Observe("cdn.b.com.", []netip.Addr{netip.MustParseAddr("3.3.3.3")})
	if err := m.Cut([]string{"a.com"}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(source.destroyed, []Tuple{blocked}) || !slices.Equal(firewall.tuples, []Tuple{blocked}) {
		t.Fatal(source.destroyed, firewall.tuples)
	}
	if len(firewall.udp) != 3 { // 1.1.1.1, 2.2.2.2 và 3.3.3.3 từ DNS
		t.Fatal(firewall.udp)
	}
	firewall.err = errors.New("nft failed")
	if m.Cut(nil) == nil {
		t.Fatal("firewall failure not reported")
	}
}

func TestFailMakesInUseFail(t *testing.T) {
	failure := errors.New("capture stopped")
	m := newMeter(&fakeSource{}, &fakeFirewall{}, "example.com")
	m.Fail(failure)
	if _, err := use(m); !errors.Is(err, failure) {
		t.Fatal(err)
	}
}

// Đọc InUse với đúng danh sách domain quota đã đặt trong bảng.
func use(m *Meter) ([]string, error) { return m.InUse(m.Table.domains) }
