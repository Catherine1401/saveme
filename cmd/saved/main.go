package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"saveme/internal/config"
	"saveme/internal/dns"
)

type options struct {
	config         string
	demo           bool
	forwarder      bool
	upstreams      string
	checkConfig    bool
	checkUpdate    bool
	checkPreserved bool
	snapshot       string
	verify         string
	status         string
	restore        string
}

func main() {
	const configFlag, configHelp = "config", "Daemon configuration file"
	const demoFlag, demoHelp, forwardFlag, forwardHelp = "demo", "Run demo without network enforcement", "forwarder", "Run the DNS forwarder"
	const upstreamFlag, defaultUpstreams, upstreamHelp = "upstreams", "1.1.1.1:53,8.8.8.8:53", "Comma-separated resolver IP:53 addresses"
	const no = false
	const checkConfigFlag, checkConfigHelp = "check-config", "Validate configuration without starting the daemon"
	const checkUpdateFlag, checkUpdateHelp = "check-update", "Validate configuration and require stored state without a lock"
	const preservedFlag, preservedHelp = "check-preserved-update", "Require valid state before a verified Linux update"
	const empty = ""
	const snapshotFlag, snapshotHelp = "update-snapshot", "Save an update commitment snapshot while the service is stopped"
	const verifyFlag, verifyHelp = "verify-update", "Verify preserved commitment and enforcement against an update snapshot"
	const statusFlag, statusHelp = "update-status", "Owner status response for update verification"
	const restoreFlag, restoreHelp = "restore-update", "Restore an update snapshot while the service is stopped"
	var opts options
	flag.StringVar(&opts.config, configFlag, config.DefaultConfig, configHelp)
	flag.BoolVar(&opts.demo, demoFlag, no, demoHelp)
	flag.BoolVar(&opts.forwarder, forwardFlag, no, forwardHelp)
	flag.StringVar(&opts.upstreams, upstreamFlag, defaultUpstreams, upstreamHelp)
	flag.BoolVar(&opts.checkConfig, checkConfigFlag, no, checkConfigHelp)
	flag.BoolVar(&opts.checkUpdate, checkUpdateFlag, no, checkUpdateHelp)
	flag.StringVar(&opts.snapshot, snapshotFlag, empty, snapshotHelp)
	flag.StringVar(&opts.verify, verifyFlag, empty, verifyHelp)
	flag.StringVar(&opts.status, statusFlag, empty, statusHelp)
	flag.StringVar(&opts.restore, restoreFlag, empty, restoreHelp)
	flag.BoolVar(&opts.checkPreserved, preservedFlag, no, preservedHelp)
	flag.Parse()
	if handled, err := runService(opts); handled || err != nil {
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, opts); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, opts options) error {
	const separator, address, empty = ",", "127.0.0.1:5353", ""
	if opts.forwarder {
		if privileged() {
			return os.ErrPermission
		}
		resolver := &dns.Resolver{Upstreams: strings.Split(opts.upstreams, separator)}
		service, err := dns.Listen([]string{address}, resolver)
		if err != nil {
			return err
		}
		defer service.Close()
		return service.Run(ctx)
	}
	if opts.demo {
		if privileged() {
			return os.ErrPermission
		}
		return runDemo(ctx, config.Demo())
	}
	if !privileged() {
		return os.ErrPermission
	}
	cfg, err := config.Load(opts.config)
	if err != nil {
		return err
	}
	if opts.snapshot != empty {
		return snapshotUpdate(cfg, opts.snapshot)
	}
	if opts.restore != empty {
		return restoreUpdate(cfg, opts.restore)
	}
	if opts.verify != empty {
		return verifyUpdate(cfg, opts.verify, opts.status)
	}
	if opts.checkPreserved {
		return checkPreservedUpdate(cfg)
	}
	if opts.checkUpdate {
		return checkUpdate(cfg)
	}
	if opts.checkConfig {
		return nil
	}
	return runProtected(ctx, cfg)
}
