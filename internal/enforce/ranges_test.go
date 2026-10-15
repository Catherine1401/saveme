package enforce

import (
	"net/netip"
	"reflect"
	"testing"
)

func TestComplementExcludesUpstreamsAndLoopback(t *testing.T) {
	const first, second, v6 = "1.1.1.1", "8.8.8.8", "2606:4700:4700::1111"
	want := []string{"0.0.0.0-1.1.1.0", "1.1.1.2-8.8.8.7", "8.8.8.9-126.255.255.255", "128.0.0.0-255.255.255.255", "::", "::2-2606:4700:4700::1110", "2606:4700:4700::1112-ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"}
	got := Complement([]netip.Addr{netip.MustParseAddr(second), netip.MustParseAddr(first), netip.MustParseAddr(v6), netip.MustParseAddr(first)})
	if !reflect.DeepEqual(got, want) {
		t.Fatal(got)
	}
}

func TestComplementHandlesBoundaries(t *testing.T) {
	const zero, last = "0.0.0.0", "255.255.255.255"
	want := []string{"0.0.0.1-126.255.255.255", "128.0.0.0-255.255.255.254", "::", "::2-ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"}
	got := Complement([]netip.Addr{netip.MustParseAddr(last), netip.MustParseAddr(zero)})
	if !reflect.DeepEqual(got, want) {
		t.Fatal(got)
	}
}
