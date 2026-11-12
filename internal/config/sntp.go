package config

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"time"
)

const (
	sntpPacketSize  = 48
	sntpEpochOffset = 2208988800 // seconds between 1900-01-01 and 1970-01-01
	sntpClientMode  = 0x23       // LI 0, version 4, mode 3 (client)
	sntpServerMode  = 4
	sntpMaxSkew     = 5 * time.Second
)

// defaultNTPServers mixes stable anycast addresses with names. Addresses keep
// the probe working when DNS itself is unavailable, and names survive address
// changes.
var defaultNTPServers = []string{
	"162.159.200.1:123",   // time.cloudflare.com
	"162.159.200.123:123", // time.cloudflare.com
	"216.239.35.0:123",    // time.google.com
	"216.239.35.4:123",    // time.google.com
	"time.windows.com:123",
	"pool.ntp.org:123",
}

// probeSNTP asks every server in parallel and reports whether any reply shows
// that the local clock is close to network time. It makes two rounds so one
// lost packet burst does not turn a synchronized clock into a warning, and
// stops at the first confirmation or when ctx expires.
func probeSNTP(ctx context.Context, servers []string, now func() time.Time) bool {
	if len(servers) == 0 {
		return false
	}
	deadline, limited := ctx.Deadline()
	for round := 0; round < 2; round++ {
		roundCtx := ctx
		cancel := func() {}
		if limited {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return false
			}
			roundCtx, cancel = context.WithTimeout(ctx, remaining/2)
		}
		ok := probeSNTPRound(roundCtx, servers, now)
		cancel()
		if ok {
			return true
		}
		if ctx.Err() != nil {
			return false
		}
	}
	return false
}

// probeSNTPRound queries all servers once and waits for the first confirmation
// or for every server to fail.
func probeSNTPRound(ctx context.Context, servers []string, now func() time.Time) bool {
	results := make(chan bool, len(servers))
	for _, server := range servers {
		go func(server string) {
			results <- sntpServerTrusted(ctx, server, now)
		}(server)
	}
	for range servers {
		select {
		case ok := <-results:
			if ok {
				return true
			}
		case <-ctx.Done():
			return false
		}
	}
	return false
}

// sntpServerTrusted sends one SNTP request and validates the reply. A negative
// or implausible round trip is rejected instead of widening the accepted
// offset.
func sntpServerTrusted(ctx context.Context, address string, now func() time.Time) bool {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "udp", address)
	if err != nil {
		return false
	}
	defer conn.Close()
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = now().Add(2 * time.Second)
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return false
	}
	request := make([]byte, sntpPacketSize)
	request[0] = sntpClientMode
	sent := now()
	putNTPTimestamp(request[40:48], sent)
	if _, err := conn.Write(request); err != nil {
		return false
	}
	reply := make([]byte, sntpPacketSize)
	if _, err := io.ReadFull(conn, reply); err != nil {
		return false
	}
	return trustedSNTPResponse(request, reply, sent, now())
}

// trustedSNTPResponse reports whether reply is a valid server response to
// request and its transmit time is within sntpMaxSkew of the local clock.
func trustedSNTPResponse(request, reply []byte, sent, received time.Time) bool {
	if len(request) < sntpPacketSize || len(reply) < sntpPacketSize {
		return false
	}
	leap := reply[0] >> 6
	version := (reply[0] >> 3) & 0x7
	mode := reply[0] & 0x7
	stratum := reply[1]
	if leap == 3 || version < 3 || version > 4 || mode != sntpServerMode || stratum == 0 || stratum > 15 {
		return false
	}
	if !bytes.Equal(reply[24:32], request[40:48]) {
		return false
	}
	server, ok := ntpTimestamp(reply[40:48])
	if !ok {
		return false
	}
	roundTrip := received.Sub(sent)
	if roundTrip < 0 || roundTrip > sntpMaxSkew {
		return false
	}
	offset := server.Sub(sent.Add(roundTrip / 2))
	return offset >= -sntpMaxSkew && offset <= sntpMaxSkew
}

// putNTPTimestamp writes at as a 64-bit NTP timestamp.
func putNTPTimestamp(field []byte, at time.Time) {
	const billion = 1000000000
	seconds := uint64(at.Unix() + sntpEpochOffset)
	fraction := (uint64(at.Nanosecond()) << 32) / billion
	binary.BigEndian.PutUint32(field[0:4], uint32(seconds))
	binary.BigEndian.PutUint32(field[4:8], uint32(fraction))
}

// ntpTimestamp decodes a 64-bit NTP timestamp, including the 2036 era wrap.
func ntpTimestamp(field []byte) (time.Time, bool) {
	if len(field) < 8 {
		return time.Time{}, false
	}
	seconds := binary.BigEndian.Uint32(field[0:4])
	fraction := binary.BigEndian.Uint32(field[4:8])
	if seconds == 0 && fraction == 0 {
		return time.Time{}, false
	}
	unix := int64(seconds) - sntpEpochOffset
	if unix < 0 {
		unix += 1 << 32
	}
	const billion = 1000000000
	nanos := (int64(fraction) * billion) >> 32
	return time.Unix(unix, nanos), true
}
