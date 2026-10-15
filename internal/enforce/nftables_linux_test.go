package enforce

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/nftables"
	miekg "github.com/miekg/dns"
	"saveme/internal/core"
	localdns "saveme/internal/dns"
)

func isolated(t *testing.T) {
	const flag, enabled, current, host, empty, reason = "SAVEME_NETNS_TEST", "1", "/proc/self/ns/net", "SAVEME_HOST_NETNS", "", "requires a separate network namespace"
	t.Helper()
	if os.Getenv(flag) != enabled {
		t.Skip(reason)
	}
	self, err := os.Readlink(current)
	if err != nil {
		t.Fatal(err)
	}
	parent := os.Getenv(host)
	if parent == empty || self == parent {
		t.Fatal(reason)
	}
}

func startResolver(t *testing.T, addresses []string, resolver *localdns.Resolver) {
	const capacity, timeout = 1, 3 * time.Second
	t.Helper()
	service, err := localdns.Listen(addresses, resolver)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan error, capacity)
	go func() { finished <- service.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-finished:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(timeout):
			t.Error(context.DeadlineExceeded)
		}
	})
}

func query(t *testing.T, address, network string, expected int) {
	const name, timeout = "sub.example.com.", time.Second
	t.Helper()
	request := new(miekg.Msg)
	request.SetQuestion(name, miekg.TypeA)
	client := &miekg.Client{Net: network, Timeout: timeout}
	response, _, err := client.Exchange(request, address)
	if err != nil {
		t.Fatal(err)
	}
	if response.Rcode != miekg.RcodeSuccess || len(response.Answer) != expected {
		t.Fatal(response)
	}
}

func removeRules(t *testing.T, entireTable bool) {
	const first = 0
	t.Helper()
	connection := &nftables.Conn{}
	if entireTable {
		connection.DelTable(table())
	} else {
		rules, err := connection.GetRules(table(), chains(table())[first])
		if err != nil {
			t.Fatal(err)
		}
		if err := connection.DelRule(rules[first]); err != nil {
			t.Fatal(err)
		}
	}
	if err := connection.Flush(); err != nil {
		t.Fatal(err)
	}
}

func TestIsolatedDNSRedirectAndRepair(t *testing.T) {
	const domain, v4, v6, remote4, remote6, proxy = "example.com", "127.0.0.1:53", "[::1]:53", "127.0.0.2:53", "[::2]:53", "127.0.0.1:5353"
	const zero, forwardUID, udp, tcp = 0, 65534, "udp", "tcp"
	isolated(t)
	filter := &core.Filter{}
	blocked := []string{domain}
	startResolver(t, []string{v4, v6}, &localdns.Resolver{Blocker: filter, Upstreams: []string{proxy}})
	enforcer := &Linux{ForwardUID: forwardUID, Filter: filter}
	t.Cleanup(func() {
		if err := enforcer.Clear(); err != nil {
			t.Error(err)
		}
	})
	for _, remove := range []bool{false, true} {
		if err := enforcer.Apply(blocked); err != nil {
			t.Fatal(err)
		}
		for _, address := range []string{remote4, remote6} {
			for _, network := range []string{udp, tcp} {
				query(t, address, network, zero)
			}
		}
		removeRules(t, remove)
		if err := enforcer.Apply(blocked); err != nil {
			t.Fatal(err)
		}
		query(t, remote4, udp, zero)
	}
	if err := enforcer.Apply(nil); err != nil {
		t.Fatal(err)
	}
	connection := &nftables.Conn{}
	tables, err := connection.ListTables()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range tables {
		if item.Name == table().Name {
			t.Fatal(item)
		}
	}
}

func echoListener(t *testing.T, address, network string) {
	const byteCount, timeout, tcp = 1, 2 * time.Second, "tcp"
	t.Helper()
	if network == tcp {
		listener, err := net.Listen(network, address)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { listener.Close() })
		go func() {
			for {
				connection, err := listener.Accept()
				if err != nil {
					return
				}
				connection.Close()
			}
		}()
		return
	}
	packet, err := net.ListenPacket(network, address)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { packet.Close() })
	go func() {
		buffer := make([]byte, byteCount)
		for {
			_, peer, err := packet.ReadFrom(buffer)
			if err != nil {
				return
			}
			packet.SetWriteDeadline(time.Now().Add(timeout))
			packet.WriteTo(buffer, peer)
		}
	}()
}

func reachable(address, network string) bool {
	const timeout, value, size, tcp = 200 * time.Millisecond, 1, 1, "tcp"
	connection, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		return false
	}
	defer connection.Close()
	if network == tcp {
		return true
	}
	connection.SetDeadline(time.Now().Add(timeout))
	if _, err := connection.Write([]byte{value}); err != nil {
		return false
	}
	buffer := make([]byte, size)
	_, err = connection.Read(buffer)
	return err == nil
}

func TestIsolatedEncryptedDNSBlockingAndRelease(t *testing.T) {
	const v4, v6, dot4, dot6, doh4, doh6, uid = "127.0.0.2", "::2", "127.0.0.2:853", "[::2]:853", "127.0.0.2:443", "[::2]:443", 65534
	const udp, tcp, domain = "udp", "tcp", "example.com"
	const resultFormat = "address=%s network=%s locked=%t reachable=%t"
	isolated(t)
	enforcer := &Linux{ForwardUID: uid, Filter: &core.Filter{}, DoH: []net.IP{net.ParseIP(v4), net.ParseIP(v6)}}
	t.Cleanup(func() { enforcer.Clear() })
	addresses := []string{dot4, dot6, doh4, doh6}
	for _, address := range addresses {
		for _, network := range []string{udp, tcp} {
			echoListener(t, address, network)
		}
	}
	for _, active := range []bool{false, true, false} {
		var blocked []string
		if active {
			blocked = []string{domain}
		}
		if err := enforcer.Apply(blocked); err != nil {
			t.Fatal(err)
		}
		for _, address := range addresses {
			for _, network := range []string{udp, tcp} {
				if actual := reachable(address, network); actual == active {
					t.Fatalf(resultFormat, address, network, active, actual)
				}
			}
		}
	}
}
