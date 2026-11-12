package config

import (
	"context"
	"net"
	"testing"
	"time"
)

// testSNTPReply builds a valid server response that echoes request's transmit
// timestamp and reports serverTime.
func testSNTPReply(request []byte, serverTime time.Time) []byte {
	reply := make([]byte, sntpPacketSize)
	reply[0] = 0x24 // LI 0, version 4, mode 4 (server)
	reply[1] = 2
	copy(reply[24:32], request[40:48])
	putNTPTimestamp(reply[40:48], serverTime)
	return reply
}

func TestTrustedSNTPResponse(t *testing.T) {
	const tolerance = 10 * time.Second
	sent := time.Unix(1794457276, 0)
	received := sent.Add(100 * time.Millisecond)
	request := make([]byte, sntpPacketSize)
	request[0] = sntpClientMode
	putNTPTimestamp(request[40:48], sent)
	valid := testSNTPReply(request, sent.Add(50*time.Millisecond))
	cases := []struct {
		name    string
		reply   []byte
		sent    time.Time
		recv    time.Time
		trusted bool
	}{
		{"synced", valid, sent, received, true},
		{"not synchronized", mutate(valid, func(reply []byte) { reply[0] = 0xE4 }), sent, received, false},
		{"unspecified stratum", mutate(valid, func(reply []byte) { reply[1] = 0 }), sent, received, false},
		{"unsynchronized stratum", mutate(valid, func(reply []byte) { reply[1] = 16 }), sent, received, false},
		{"client mode", mutate(valid, func(reply []byte) { reply[0] = sntpClientMode }), sent, received, false},
		{"old version", mutate(valid, func(reply []byte) { reply[0] = (reply[0] &^ 0x38) | 2<<3 }), sent, received, false},
		{"mismatched originate", mutate(valid, func(reply []byte) { reply[24] ^= 0xFF }), sent, received, false},
		{"missing transmit", mutate(valid, func(reply []byte) { clear(reply[40:48]) }), sent, received, false},
		{"clock ahead", testSNTPReply(request, sent.Add(tolerance)), sent, received, false},
		{"clock behind", testSNTPReply(request, sent.Add(-tolerance)), sent, received, false},
		{"clock close", testSNTPReply(request, sent.Add(3*time.Second)), sent, received, true},
		{"zero round trip", valid, sent, sent, true},
		{"short reply", valid[:sntpPacketSize-1], sent, received, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := trustedSNTPResponse(request, tc.reply, tc.sent, tc.recv); got != tc.trusted {
				t.Fatalf("trustedSNTPResponse() = %v, want %v", got, tc.trusted)
			}
		})
	}
}

func mutate(packet []byte, change func([]byte)) []byte {
	clone := append([]byte{}, packet...)
	change(clone)
	return clone
}

func TestProbeSNTPAcceptsLocalServer(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	go func() {
		request := make([]byte, sntpPacketSize)
		n, addr, err := conn.ReadFrom(request)
		if err != nil || n < sntpPacketSize {
			return
		}
		reply := testSNTPReply(request[:n], time.Now().Add(time.Second))
		_, _ = conn.WriteTo(reply, addr)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if !probeSNTP(ctx, []string{conn.LocalAddr().String()}, time.Now) {
		t.Fatal("probeSNTP rejected a valid local reply")
	}
}

func TestProbeSNTPRejectsSilentServer(t *testing.T) {
	const timeout = 50 * time.Millisecond
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if probeSNTP(ctx, []string{conn.LocalAddr().String()}, time.Now) {
		t.Fatal("probeSNTP accepted a silent server")
	}
}

func TestProbeSNTPWithoutServers(t *testing.T) {
	if probeSNTP(context.Background(), nil, time.Now) {
		t.Fatal("probeSNTP accepted an empty server list")
	}
}
