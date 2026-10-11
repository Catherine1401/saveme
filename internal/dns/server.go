package dns

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	miekg "github.com/miekg/dns"
)

type Blocker interface{ Blocked(string) bool }

type Resolver struct {
	Blocker   Blocker
	Upstreams []string
}

func (r *Resolver) ServeDNS(writer miekg.ResponseWriter, request *miekg.Msg) {
	const one, zero = 1, 0
	if len(request.Question) != one || request.Opcode != miekg.OpcodeQuery {
		response := new(miekg.Msg)
		response.SetRcode(request, miekg.RcodeFormatError)
		writer.WriteMsg(response)
		return
	}
	if r.Blocker != nil && r.Blocker.Blocked(request.Question[zero].Name) {
		response := new(miekg.Msg)
		response.SetReply(request)
		response.RecursionAvailable = true
		writer.WriteMsg(response)
		return
	}
	response, err := r.resolve(request, writer.RemoteAddr().Network())
	if err != nil {
		response = new(miekg.Msg)
		response.SetRcode(request, miekg.RcodeServerFailure)
	}
	writer.WriteMsg(response)
}

func (r *Resolver) resolve(request *miekg.Msg, network string) (*miekg.Msg, error) {
	const (
		number0 = 0
	)

	const udp, tcp, timeout, reason = "udp", "tcp", 2 * time.Second, "no upstream resolver"
	transport := udp
	if network == tcp {
		transport = tcp
	}
	var failures []error
	for _, upstream := range r.Upstreams {
		client := &miekg.Client{Net: transport, Timeout: timeout}
		response, _, err := client.Exchange(request, upstream)
		if err == nil && response.Truncated && transport == udp {
			client.Net = tcp
			response, _, err = client.Exchange(request, upstream)
		}
		if err == nil {
			return r.sanitize(request, response), nil
		}
		failures = append(failures, err)
	}
	if len(failures) == number0 {
		return nil, errors.New(reason)
	}
	return nil, errors.Join(failures...)
}

func (r *Resolver) sanitize(request, response *miekg.Msg) *miekg.Msg {
	if r.Blocker == nil {
		return response
	}
	for _, answer := range append(append([]miekg.RR{}, response.Answer...), response.Extra...) {
		if r.Blocker.Blocked(answer.Header().Name) {
			return emptyReply(request)
		}
		if cname, ok := answer.(*miekg.CNAME); ok && r.Blocker.Blocked(cname.Target) {
			return emptyReply(request)
		}
	}
	return response
}

func emptyReply(request *miekg.Msg) *miekg.Msg {
	response := new(miekg.Msg)
	response.SetReply(request)
	response.RecursionAvailable = true
	return response
}

type Service struct{ servers []*miekg.Server }

func Listen(addresses []string, resolver *Resolver) (*Service, error) {
	const udp, tcp = "udp", "tcp"
	service := &Service{}
	for _, address := range addresses {
		packet, err := net.ListenPacket(udp, address)
		if err != nil {
			service.Close()
			return nil, err
		}
		service.servers = append(service.servers, &miekg.Server{PacketConn: packet, Net: udp, Handler: resolver})
		listener, err := net.Listen(tcp, address)
		if err != nil {
			service.Close()
			return nil, err
		}
		service.servers = append(service.servers, &miekg.Server{Listener: listener, Net: tcp, Handler: resolver})
	}
	return service, nil
}

func (s *Service) Close() {
	const timeout = 2 * time.Second
	for _, server := range s.servers {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		server.ShutdownContext(ctx)
		cancel()
		if server.PacketConn != nil {
			server.PacketConn.Close()
		}
		if server.Listener != nil {
			server.Listener.Close()
		}
	}
}

func (s *Service) Run(ctx context.Context) error {
	const one = 1
	results := make(chan error, len(s.servers))
	var group sync.WaitGroup
	for _, server := range s.servers {
		group.Add(one)
		go func() { defer group.Done(); results <- server.ActivateAndServe() }()
	}
	select {
	case <-ctx.Done():
		s.Close()
		group.Wait()
		return nil
	case err := <-results:
		s.Close()
		group.Wait()
		return err
	}
}
