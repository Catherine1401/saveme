package main

import (
	"net"
	"reflect"
	"time"

	miekg "github.com/miekg/dns"
	"saveme/internal/config"
	"saveme/internal/core"
	"saveme/internal/enforce"
	"saveme/internal/store"
)

func preservedUpdate(before, after core.State, status core.Response) error {
	const zero = 0
	if !status.OK && status.Error != core.ErrClockUnsync {
		return core.ErrEnforcement
	}
	if after.Version != before.Version || after.MaxSeen < before.MaxSeen || status.Now < after.MaxSeen || len(before.Entries) != len(after.Entries) || len(status.Entries) != len(after.Entries) {
		return core.ErrStorage
	}
	trusted, active := status.OK, false
	now := time.Unix(status.Now, zero)
	for index, prior := range before.Entries {
		entry, observed := after.Entries[index], status.Entries[index]
		if prior.Domain != entry.Domain || observed.Domain != entry.Domain {
			return core.ErrStorage
		}
		expired := trusted && prior.Lock != nil && prior.Lock.Kind == core.Range && status.Now >= prior.Lock.EndAt
		if !reflect.DeepEqual(prior.Lock.Clone(), entry.Lock.Clone()) && !(expired && entry.Lock == nil) {
			return core.ErrLocked
		}
		if err := preservedEntry(entry, observed, now, trusted); err != nil {
			return err
		}
		active = active || observed.Active
	}
	if status.Active != active {
		return core.ErrEnforcement
	}
	return nil
}

func preservedEntry(entry core.Entry, observed core.EntryStatus, now time.Time, trusted bool) error {
	expected := core.EntryStatus{Domain: entry.Domain}
	if entry.Lock != nil {
		expected.Kind, expected.Zone = entry.Lock.Kind, entry.Lock.Zone
		expected.Active = !trusted || entry.Lock.Active(now)
		if expected.Active && trusted {
			expected.StartAt, expected.EndAt = entry.Lock.WindowStart(now), entry.Lock.Transition(now)
		} else if !expected.Active {
			expected.NextAt, expected.Windows = entry.Lock.Transition(now), entry.Lock.Windows
		}
	}
	expected.Windows = append([]core.Window{}, expected.Windows...)
	observed.Windows = append([]core.Window{}, observed.Windows...)
	if !reflect.DeepEqual(expected, observed) {
		return core.ErrLocked
	}
	return nil
}

func verifyUpdate(cfg config.Config, snapshotPath, statusPath string) error {
	var before core.State
	if err := readUpdateJSON(snapshotPath, &before); err != nil {
		return err
	}
	if err := before.Validate(); err != nil {
		return err
	}
	var status core.Response
	if err := readUpdateJSON(statusPath, &status); err != nil {
		return err
	}
	if err := checkPreservedUpdate(cfg); err != nil {
		return err
	}
	after, err := store.New(cfg.StateDir, nil).Load()
	if err != nil {
		return err
	}
	if err := preservedUpdate(before, after, status); err != nil {
		return err
	}
	uid, _, err := forwardIdentity(cfg)
	if err != nil {
		return err
	}
	enforcer := &enforce.Linux{ForwardUID: uid}
	for _, address := range cfg.DoH {
		enforcer.DoH = append(enforcer.DoH, net.ParseIP(address))
	}
	if err := enforcer.Verify(status.Active); err != nil {
		return err
	}
	if status.Active {
		const zero = 0
		var domains []string
		for _, entry := range after.Entries {
			if entry.Lock != nil && (!status.OK || entry.Lock.Active(time.Unix(status.Now, zero))) {
				domains = append(domains, entry.Domain)
			}
		}
		return verifyBlockedDNS(domains, cfg.Upstreams)
	}
	return nil
}

func verifyBlockedDNS(domains, addresses []string) error {
	const udp, tcp, suffix, zero, timeout = "udp", "tcp", ".", 0, time.Second
	for _, network := range []string{udp, tcp} {
		client := &miekg.Client{Net: network, Timeout: timeout}
		for _, address := range addresses {
			for _, domain := range domains {
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
