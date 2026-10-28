package meter

import (
	"encoding/binary"
	"golang.org/x/net/bpf"
	"net/netip"
	"testing"
	"time"
)

// Dựng gói IPv4 hoặc IPv6 chứa một segment TCP tới cổng dport.
func buildPacket(src, dst netip.AddrPort, dport uint16, seq uint32, payload []byte) []byte {
	tcp := make([]byte, tcpMinimum)
	binary.BigEndian.PutUint16(tcp[0:], src.Port())
	binary.BigEndian.PutUint16(tcp[2:], dport)
	binary.BigEndian.PutUint32(tcp[tcpSeqOffset:], seq)
	tcp[tcpOffsetByte] = (tcpMinimum / tcpOffsetUnit) << tcpOffsetShift
	tcp = append(tcp, payload...)
	if src.Addr().Is4() {
		ip := make([]byte, ipv4Minimum)
		ip[0], ip[ipv4Protocol] = versionIPv4<<4|ipv4Minimum/tcpOffsetUnit, protocolTCP
		copy(ip[ipv4Source:], src.Addr().AsSlice())
		copy(ip[ipv4Source+4:], dst.Addr().AsSlice())
		return append(ip, tcp...)
	}
	ip := make([]byte, ipv6Header)
	ip[0], ip[ipv6Next] = versionIPv6<<4, protocolTCP
	copy(ip[ipv6Source:], src.Addr().AsSlice())
	copy(ip[ipv6Source+16:], dst.Addr().AsSlice())
	return append(ip, tcp...)
}

func TestParsePacketIPv4AndIPv6(t *testing.T) {
	const seq = 1000
	payload := []byte{recordHandshake, 3, 1}
	for _, pair := range [][2]string{{"10.0.0.2:50000", "157.240.1.35:443"}, {"[2001:db8::2]:50000", "[2a03:2880::1]:443"}} {
		src, dst := netip.MustParseAddrPort(pair[0]), netip.MustParseAddrPort(pair[1])
		segment, ok := ParsePacket(buildPacket(src, dst, httpsPort, seq, payload))
		if !ok || segment.Tuple.Local != src || segment.Tuple.Remote != dst || segment.Seq != seq || len(segment.Payload) != len(payload) {
			t.Fatal(pair, segment, ok)
		}
	}
}

func TestParsePacketRejectsOtherTraffic(t *testing.T) {
	src, dst := netip.MustParseAddrPort("10.0.0.2:50000"), netip.MustParseAddrPort("1.1.1.1:80")
	cases := map[string][]byte{
		"wrong port":  buildPacket(src, dst, 80, 1, []byte{1}),
		"no payload":  buildPacket(src, dst, httpsPort, 1, nil),
		"empty":       nil,
		"truncated":   buildPacket(src, dst, httpsPort, 1, []byte{1})[:ipv4Minimum+3],
		"garbage ver": append([]byte{0x15}, make([]byte, 60)...),
	}
	for name, packet := range cases {
		if _, ok := ParsePacket(packet); ok {
			t.Fatal(name, "accepted")
		}
	}
}

func TestFlowsAssembleHelloSplitAcrossSegments(t *testing.T) {
	const seq = 7000
	hello := realHello(t, "www.instagram.com")
	tuple := tuple("10.0.0.2:50000", "157.240.197.35:443")
	flows := NewFlows(time.Now)
	cut := len(hello) / 2
	if name, done := flows.Feed(Segment{Tuple: tuple, Seq: seq, Payload: hello[:cut]}); done {
		t.Fatal("done on half hello", name)
	}
	// Gửi lại đoạn đầu (retransmit) không làm hỏng việc gom.
	flows.Feed(Segment{Tuple: tuple, Seq: seq, Payload: hello[:cut]})
	name, done := flows.Feed(Segment{Tuple: tuple, Seq: seq + uint32(cut), Payload: hello[cut:]})
	if !done || name != "www.instagram.com" {
		t.Fatal(name, done)
	}
}

func TestFlowsIgnoreNonTLSGapsAndExpiry(t *testing.T) {
	const seq = 7000
	hello := realHello(t, "example.com")
	tuple := tuple("10.0.0.2:50000", "1.1.1.1:443")
	now := time.Now()
	flows := NewFlows(func() time.Time { return now })
	if _, done := flows.Feed(Segment{Tuple: tuple, Seq: seq, Payload: []byte("GET /")}); done || len(flows.flows) != 0 {
		t.Fatal("tracked non-TLS flow")
	}
	flows.Feed(Segment{Tuple: tuple, Seq: seq, Payload: hello[:10]})
	if _, done := flows.Feed(Segment{Tuple: tuple, Seq: seq + 500, Payload: hello[10:]}); done || len(flows.flows) != 0 {
		t.Fatal("gap accepted")
	}
	flows.Feed(Segment{Tuple: tuple, Seq: seq, Payload: hello[:10]})
	now = now.Add(flowLifetime + time.Second)
	if _, done := flows.Feed(Segment{Tuple: tuple, Seq: seq + 10, Payload: hello[10:]}); done {
		t.Fatal("expired flow completed")
	}
}

func TestCaptureFilterKeepsOnlyHTTPSWithPayload(t *testing.T) {
	v4src, v4dst := netip.MustParseAddrPort("10.0.0.2:50000"), netip.MustParseAddrPort("1.1.1.1:443")
	v6src, v6dst := netip.MustParseAddrPort("[2001:db8::2]:50000"), netip.MustParseAddrPort("[2a03::1]:443")
	cases := []struct {
		name   string
		ipv6   bool
		packet []byte
		keep   bool
	}{
		{"v4 https payload", false, buildPacket(v4src, v4dst, httpsPort, 1, []byte{recordHandshake, 3}), true},
		{"v4 other port", false, buildPacket(v4src, v4dst, 80, 1, []byte{1}), false},
		{"v4 ack only", false, buildPacket(v4src, v4dst, httpsPort, 1, nil), false},
		{"v6 https payload", true, buildPacket(v6src, v6dst, httpsPort, 1, []byte{recordHandshake, 3}), true},
		{"v6 other port", true, buildPacket(v6src, v6dst, 80, 1, []byte{1}), false},
		{"v6 ack only", true, buildPacket(v6src, v6dst, httpsPort, 1, nil), false},
		{"v6 packet on v4 filter", false, buildPacket(v6src, v6dst, httpsPort, 1, []byte{recordHandshake}), false},
		{"v4 packet on v6 filter", true, buildPacket(v4src, v4dst, httpsPort, 1, []byte{recordHandshake}), false},
	}
	for _, c := range cases {
		vm, err := bpf.NewVM(CaptureFilter(c.ipv6))
		if err != nil {
			t.Fatal(err)
		}
		size, err := vm.Run(c.packet)
		if err != nil || (size > 0) != c.keep {
			t.Fatalf("%s: size=%d err=%v", c.name, size, err)
		}
	}
}
