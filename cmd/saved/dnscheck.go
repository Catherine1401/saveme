package main

import (
	miekg "github.com/miekg/dns"
	"saveme/internal/core"
	"time"
)

func verifyBlockedDNS(domains, addresses []string) error {
	const udp = "udp"
	const tcp = "tcp"
	for _, network := range []string{udp, tcp} {
		const timeout = time.Second
		client := &miekg.Client{Net: network, Timeout: timeout}
		for _, address := range addresses {
			for _, domain := range domains {
				const suffix = "."
				const zero = 0
				request := new(miekg.Msg)
				request.SetQuestion(domain+suffix, miekg.TypeA)
				response, _, err := client.Exchange(request, address)
				if err != nil {
					return err
				}
				if response.Rcode != miekg.RcodeSuccess || len(response.Answer) != zero {
					return core.ErrEnforcement
				}
			}
		}
	}
	return nil
}
