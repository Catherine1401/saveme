package main

import (
	"context"
	"log"
	"net"

	"saveme/internal/config"
	"saveme/internal/core"
	"saveme/internal/daemon"
	"saveme/internal/dns"
	"saveme/internal/enforce"
	"saveme/internal/meter"
)

// Server Unix socket chỉ nhận peer có uid được phép.
func newServer(engine *daemon.Engine, cfg config.Config) *daemon.Server {
	return &daemon.Server{Engine: engine, UID: *cfg.AllowedUID}
}

func runProtected(ctx context.Context, cfg config.Config) error {
	const (
		number1 = 1
	)

	const proxyAddress, v4, v6 = "127.0.0.1:5353", "127.0.0.1:53", "[::1]:53"

	const taskCount = 3
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	uid, gid, err := forwardIdentity(cfg)
	if err != nil {
		return err
	}
	filter := &core.Filter{}
	enforcer := &enforce.Linux{ForwardUID: uid, Filter: filter}
	for _, address := range cfg.DoH {
		enforcer.DoH = append(enforcer.DoH, net.ParseIP(address))
	}
	engine, err := protectedEngine(cfg, enforcer, enforcer.Clear)
	if err != nil {
		return err
	}
	engine.UpdatePending = updatePending
	resolver := &dns.Resolver{Blocker: filter, Upstreams: []string{proxyAddress}}
	quotaFirewall := &enforce.QuotaFirewall{}
	defer quotaFirewall.Clear()
	startMeter(ctx, engine, resolver, quotaFirewall)
	service, err := dns.Listen([]string{v4, v6}, resolver)
	if err != nil {
		return err
	}
	defer service.Close()
	child, err := startForwarder(ctx, cfg, uid, gid)
	if err != nil {
		return err
	}

	results := make(chan error, taskCount)
	go func() { results <- child.Wait() }()
	go func() { results <- service.Run(ctx) }()
	go func() { results <- serveEngine(ctx, cfg, engine) }()
	first := <-results
	cancel()
	for range taskCount - number1 {
		<-results
	}
	return first
}

// Dựng bộ đo quota; không dựng được thì dùng bộ đo lỗi để giữ chặn đã có và báo ERR_ENFORCEMENT.
func startMeter(ctx context.Context, engine *daemon.Engine, resolver *dns.Resolver, firewall meter.Firewall) {
	capture := &meter.Capture{Table: meter.NewTable()}
	if err := capture.Open(); err != nil {
		log.Printf("quota meter unavailable: %v", err)
		engine.SetMeter(daemon.FailedMeter(err))
		return
	}
	measure := &meter.Meter{Table: capture.Table, Source: meter.Diag{}, Firewall: firewall}
	resolver.Observe = capture.Table.Observe
	engine.SetMeter(measure)
	go func() {
		if err := capture.Run(ctx); err != nil {
			log.Printf("quota meter stopped: %v", err)
			measure.Fail(err)
		}
	}()
}
