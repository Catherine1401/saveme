package main

import (
	"net"
	"time"

	"saveme/internal/config"
	"saveme/internal/core"
	"saveme/internal/enforce"
	"saveme/internal/store"
)

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
