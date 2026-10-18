package main

import (
	"context"

	"saveme/internal/config"
	"saveme/internal/core"
	"saveme/internal/daemon"
	"saveme/internal/dns"
	"saveme/internal/enforce"
)

// Named pipe chỉ cho SID được phép kết nối.
func newServer(engine *daemon.Engine, cfg config.Config) *daemon.Server {
	return &daemon.Server{Engine: engine, SID: cfg.AllowedSID}
}

// Resolver chạy ngay trong service vì Firewall chỉ cho DNS ra tới các upstream đã cấu hình.
func runProtected(ctx context.Context, cfg config.Config) error {
	const v4, v6, taskCount, one = "127.0.0.1:53", "[::1]:53", 2, 1
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	filter := &core.Filter{}
	enforcer, err := enforce.ConfiguredFirewall(cfg.Upstreams, cfg.DoH, enforce.PowerShell)
	if err != nil {
		return err
	}
	enforcer.Filter = filter
	engine, err := protectedEngine(cfg, enforcer, enforcer.Clear)
	if err != nil {
		return err
	}
	engine.UpdatePending = updatePending
	resolver := &dns.Resolver{Blocker: filter, Upstreams: cfg.Upstreams}
	service, err := dns.Listen([]string{v4, v6}, resolver)
	if err != nil {
		return err
	}
	defer service.Close()
	results := make(chan error, taskCount)
	go func() { results <- service.Run(ctx) }()
	go func() { results <- serveEngine(ctx, cfg, engine) }()
	first := <-results
	cancel()
	for range taskCount - one {
		<-results
	}
	return first
}
