package dns

import (
	"context"
	"net"
	"testing"
	"time"

	miekg "github.com/miekg/dns"
	"saveme/internal/core"
)

type runningDNS struct {
	udp string
	tcp string
}

func runDNS(t *testing.T, service *Service) runningDNS {
	const udpIndex, tcpIndex, timeout = 0, 1, 3 * time.Second
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan error, tcpIndex)
	go func() { finished <- service.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-finished:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(timeout):
			t.Error(core.ErrEnforcement)
		}
	})
	return runningDNS{udp: service.servers[udpIndex].PacketConn.LocalAddr().String(), tcp: service.servers[tcpIndex].Listener.Addr().String()}
}

func upstream(t *testing.T, truncate bool) runningDNS {
	const tcp, udp, address, alias, blocked, answer, zero = "tcp", "udp", "127.0.0.1:0", "alias.test.", "example.com.", "192.0.2.10", 0
	t.Helper()
	listener, err := net.Listen(tcp, address)
	if err != nil {
		t.Fatal(err)
	}
	packet, err := net.ListenPacket(udp, listener.Addr().String())
	if err != nil {
		listener.Close()
		t.Fatal(err)
	}
	handler := miekg.HandlerFunc(func(writer miekg.ResponseWriter, request *miekg.Msg) {
		const ttl = 60
		response := new(miekg.Msg)
		response.SetReply(request)
		if truncate && writer.RemoteAddr().Network() == udp {
			response.Truncated = true
		} else {
			header := miekg.RR_Header{Name: request.Question[zero].Name, Rrtype: miekg.TypeA, Class: miekg.ClassINET, Ttl: ttl}
			response.Answer = []miekg.RR{&miekg.A{Hdr: header, A: net.ParseIP(answer)}}
			if header.Name == alias {
				header.Rrtype = miekg.TypeCNAME
				response.Answer = []miekg.RR{&miekg.CNAME{Hdr: header, Target: blocked}}
			}
		}
		writer.WriteMsg(response)
	})
	service := &Service{servers: []*miekg.Server{{Net: udp, PacketConn: packet, Handler: handler}, {Net: tcp, Listener: listener, Handler: handler}}}
	return runDNS(t, service)
}

func exchange(t *testing.T, address, network, name string, kind uint16) *miekg.Msg {
	const timeout, attempts = time.Second, 10
	t.Helper()
	request := new(miekg.Msg)
	request.SetQuestion(name, kind)
	client := &miekg.Client{Net: network, Timeout: timeout}
	var last error
	for range attempts {
		response, _, err := client.Exchange(request, address)
		if err == nil {
			return response
		}
		last = err
	}
	t.Fatal(last)
	return nil
}

func TestDNSNetworkBlockingForwardingAndRelease(t *testing.T) {
	const address, domain, blocked, subdomain, allowed, alias, zero, one = "127.0.0.1:0", "example.com", "example.com.", "a.example.com.", "allowed.test.", "alias.test.", 0, 1
	const udp, tcp = "udp", "tcp"
	filter := &core.Filter{}
	filter.Set([]string{domain})
	remote := upstream(t, false)
	service, err := Listen([]string{address}, &Resolver{Blocker: filter, Upstreams: []string{remote.udp}})
	if err != nil {
		t.Fatal(err)
	}
	server := runDNS(t, service)
	transports := []struct {
		network string
		address string
	}{{network: udp, address: server.udp}, {network: tcp, address: server.tcp}}
	for _, transport := range transports {
		for _, name := range []string{blocked, subdomain, alias} {
			response := exchange(t, transport.address, transport.network, name, miekg.TypeA)
			if response.Rcode != miekg.RcodeSuccess || len(response.Answer) != zero {
				t.Fatal(response)
			}
		}
		response := exchange(t, transport.address, transport.network, allowed, miekg.TypeA)
		if len(response.Answer) != one {
			t.Fatal(response)
		}
		response = exchange(t, transport.address, transport.network, blocked, miekg.TypeAAAA)
		if len(response.Answer) != zero {
			t.Fatal(response)
		}
	}
	filter.Set(nil)
	response := exchange(t, server.udp, udp, blocked, miekg.TypeA)
	if len(response.Answer) != one {
		t.Fatal(response)
	}
}

func TestDNSTruncationFallsBackToTCP(t *testing.T) {
	const address, udp, allowed, one = "127.0.0.1:0", "udp", "allowed.test.", 1
	remote := upstream(t, true)
	service, err := Listen([]string{address}, &Resolver{Upstreams: []string{remote.udp}})
	if err != nil {
		t.Fatal(err)
	}
	server := runDNS(t, service)
	response := exchange(t, server.udp, udp, allowed, miekg.TypeA)
	if response.Truncated || len(response.Answer) != one {
		t.Fatal(response)
	}
}

func TestDNSMissingUpstreamReturnsFailure(t *testing.T) {
	const address, udp, allowed = "127.0.0.1:0", "udp", "allowed.test."
	service, err := Listen([]string{address}, &Resolver{})
	if err != nil {
		t.Fatal(err)
	}
	server := runDNS(t, service)
	response := exchange(t, server.udp, udp, allowed, miekg.TypeA)
	if response.Rcode != miekg.RcodeServerFailure {
		t.Fatal(response)
	}
}

func TestDNSFacebookExactHosts(t *testing.T) {
	const address, facebook, apex, www, mobile, chat, messenger = "127.0.0.1:0", "facebook.com", "facebook.com.", "www.facebook.com.", "m.facebook.com.", "edge-chat.facebook.com.", "messenger.com."
	const udp, tcp, zero, one = "udp", "tcp", 0, 1
	const explicitChat = "edge-chat.facebook.com"
	filter := &core.Filter{}
	blocked := []string{facebook}
	filter.Set(blocked)
	remote := upstream(t, false)
	service, err := Listen([]string{address}, &Resolver{Blocker: filter, Upstreams: []string{remote.udp}})
	if err != nil {
		t.Fatal(err)
	}
	server := runDNS(t, service)
	transports := []struct {
		network string
		address string
	}{{network: udp, address: server.udp}, {network: tcp, address: server.tcp}}
	cases := []struct {
		name    string
		answers int
	}{{name: apex, answers: zero}, {name: www, answers: zero}, {name: mobile, answers: one}, {name: chat, answers: one}, {name: messenger, answers: one}}
	for _, transport := range transports {
		for _, tc := range cases {
			response := exchange(t, transport.address, transport.network, tc.name, miekg.TypeA)
			if response.Rcode != miekg.RcodeSuccess || len(response.Answer) != tc.answers {
				t.Fatal(response)
			}
		}
	}
	blocked = append(blocked, explicitChat)
	filter.Set(blocked)
	if response := exchange(t, server.udp, udp, chat, miekg.TypeA); len(response.Answer) != zero {
		t.Fatal(response)
	}
	filter.Set(nil)
	if response := exchange(t, server.udp, udp, www, miekg.TypeA); len(response.Answer) != one {
		t.Fatal(response)
	}
}
