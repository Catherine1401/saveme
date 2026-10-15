package enforce

import (
	"net/netip"
	"sort"
)

type span struct{ low, high netip.Addr }

// Trả các dải địa chỉ phủ toàn bộ không gian IPv4/IPv6 trừ loopback và các địa chỉ được miễn.
func Complement(exempt []netip.Addr) []string {
	const v4Loopback, v4First, v4Last, v6Loopback, v6First, v6Last = "127.0.0.0", "0.0.0.0", "255.255.255.255", "::1", "::", "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"
	const v4LoopbackEnd = "127.255.255.255"
	v4 := []span{{netip.MustParseAddr(v4Loopback), netip.MustParseAddr(v4LoopbackEnd)}}
	v6 := []span{{netip.MustParseAddr(v6Loopback), netip.MustParseAddr(v6Loopback)}}
	for _, address := range exempt {
		address = address.Unmap()
		if address.Is4() {
			v4 = append(v4, span{address, address})
		} else {
			v6 = append(v6, span{address, address})
		}
	}
	result := gaps(v4, netip.MustParseAddr(v4First), netip.MustParseAddr(v4Last))
	return append(result, gaps(v6, netip.MustParseAddr(v6First), netip.MustParseAddr(v6Last))...)
}

// Liệt kê các khoảng trống giữa các dải bị loại trong [first, last].
func gaps(excluded []span, first, last netip.Addr) []string {
	const separator = "-"
	sort.Slice(excluded, func(i, j int) bool { return excluded[i].low.Less(excluded[j].low) })
	var result []string
	next, done := first, false
	for _, item := range excluded {
		if done || item.high.Less(next) {
			continue
		}
		if next.Less(item.low) {
			result = append(result, rangeText(next, item.low.Prev(), separator))
		}
		if item.high == last {
			done = true
			continue
		}
		next = item.high.Next()
	}
	if !done {
		result = append(result, rangeText(next, last, separator))
	}
	return result
}

func rangeText(low, high netip.Addr, separator string) string {
	if low == high {
		return low.String()
	}
	return low.String() + separator + high.String()
}
