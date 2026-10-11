package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"
	_ "time/tzdata"

	"saveme/internal/config"
	"saveme/internal/core"
	"saveme/internal/daemon"
	"saveme/internal/dns"
	"saveme/internal/domain"
	"saveme/internal/enforce"
	"saveme/internal/store"
)

type demoEnforcer struct{}

func (demoEnforcer) Apply(core.State, bool) error { return nil }

func runDemo(ctx context.Context, cfg config.Config) error {
	const mode, warning = 0700, "DEMO: no DNS or network blocking; socket %s"
	if err := os.MkdirAll(filepath.Dir(cfg.Socket), mode); err != nil {
		return err
	}
	repo := store.New(cfg.StateDir, nil)
	state, err := repo.Load()
	if err != nil {
		return err
	}
	clock := core.NewClock(state.MaxSeen, time.Now, time.Now, func() bool { return true })
	engine, err := daemon.New(state, clock, repo, demoEnforcer{}, domain.Normalizer{}, cfg.Zone)
	if err != nil {
		return err
	}
	log.Printf(warning, cfg.Socket)
	return serveEngine(ctx, cfg, engine)
}

func runProtected(ctx context.Context, cfg config.Config) error {
	const (
		number1 = 1
	)

	const proxyAddress, v4, v6, mode = "127.0.0.1:5353", "127.0.0.1:53", "[::1]:53", 0700

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
	repo := store.New(cfg.StateDir, store.Immutable{})
	state, err := repo.Load()
	if err != nil {
		cleanupErr := enforcer.Clear()
		return errors.Join(err, cleanupErr)
	}
	if err := os.MkdirAll(cfg.StateDir, mode); err != nil {
		return err
	}
	if err := repo.Save(state); err != nil {
		return err
	}
	ntp := &config.NTP{}
	clock := core.NewClock(state.MaxSeen, time.Now, time.Now, ntp.Trusted)
	engine, err := daemon.New(state, clock, repo, enforcer, domain.Normalizer{}, cfg.Zone)
	if err != nil {
		return err
	}
	if err := engine.Tick(); err != nil {
		return err
	}
	resolver := &dns.Resolver{Blocker: filter, Upstreams: []string{proxyAddress}}
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

func serveEngine(ctx context.Context, cfg config.Config, engine *daemon.Engine) error {
	const tickError = "state/enforcement sync failed: %v"
	server := &daemon.Server{Engine: engine, UID: *cfg.AllowedUID}
	if err := server.Listen(cfg.Socket); err != nil {
		return err
	}
	defer server.Close()
	if err := engine.Tick(); err != nil {
		return err
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := engine.Tick(); err != nil {
					log.Printf(tickError, err)
				}
			}
		}
	}()
	return server.Serve(ctx)
}
