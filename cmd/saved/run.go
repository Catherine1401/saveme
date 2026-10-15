package main

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"
	_ "time/tzdata"

	"saveme/internal/config"
	"saveme/internal/core"
	"saveme/internal/daemon"
	"saveme/internal/domain"
	"saveme/internal/store"
)

type demoEnforcer struct{}

func (demoEnforcer) Apply([]string) error { return nil }

func runDemo(ctx context.Context, cfg config.Config) error {
	const mode, warning = 0700, "DEMO: no DNS or network blocking; socket %s"
	if err := os.MkdirAll(filepath.Dir(cfg.StateDir), mode); err != nil {
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
	const codeVariable, invalidCode, unset = "SAVEME_DEMO_CODE", "DEMO: ignoring %s; use 4 confirmation-code characters", ""
	if code := os.Getenv(codeVariable); code != unset && !engine.UseFixedCode(code) {
		log.Printf(invalidCode, codeVariable)
	}
	log.Printf(warning, cfg.Socket)
	return serveEngine(ctx, cfg, engine)
}

// Nạp state đã lưu, dựng clock theo NTP và engine production; lỗi nạp state thì gỡ chặn trước khi thoát.
func protectedEngine(cfg config.Config, enforcer daemon.Enforcer, clear func() error) (*daemon.Engine, error) {
	const mode = 0700
	repo := store.New(cfg.StateDir, store.Immutable{})
	state, err := repo.Load()
	if err != nil {
		cleanupErr := clear()
		return nil, errors.Join(err, cleanupErr)
	}
	if err := os.MkdirAll(cfg.StateDir, mode); err != nil {
		return nil, err
	}
	if err := repo.Save(state); err != nil {
		return nil, err
	}
	ntp := &config.NTP{}
	clock := core.NewClock(state.MaxSeen, time.Now, time.Now, ntp.Trusted)
	engine, err := daemon.New(state, clock, repo, enforcer, domain.Normalizer{}, cfg.Zone)
	if err != nil {
		return nil, err
	}
	if err := engine.Tick(); err != nil {
		return nil, err
	}
	return engine, nil
}

func serveEngine(ctx context.Context, cfg config.Config, engine *daemon.Engine) error {
	const tickError = "state/enforcement sync failed: %v"
	server := newServer(engine, cfg)
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
