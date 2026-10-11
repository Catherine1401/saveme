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
	config      string
	demo        bool
	forwarder   bool
	upstreams   string
	checkConfig bool
	checkUpdate bool
}

func main() {
	const configFlag, configPath, configHelp = "config", "/etc/saveme/config.json", "Daemon configuration file"
	const demoFlag, demoHelp, forwardFlag, forwardHelp = "demo", "Run demo without network enforcement", "forwarder", "Run the DNS forwarder"
	const upstreamFlag, defaultUpstreams, upstreamHelp = "upstreams", "1.1.1.1:53,8.8.8.8:53", "Comma-separated resolver IP:53 addresses"
	const no = false
	const checkConfigFlag, checkConfigHelp = "check-config", "Validate configuration without starting the daemon"
	const checkUpdateFlag, checkUpdateHelp = "check-update", "Validate configuration and require stored state without a lock"
	var opts options
	flag.StringVar(&opts.config, configFlag, configPath, configHelp)
	flag.BoolVar(&opts.demo, demoFlag, no, demoHelp)
	flag.BoolVar(&opts.forwarder, forwardFlag, no, forwardHelp)
	flag.StringVar(&opts.upstreams, upstreamFlag, defaultUpstreams, upstreamHelp)
	flag.BoolVar(&opts.checkConfig, checkConfigFlag, no, checkConfigHelp)
	flag.BoolVar(&opts.checkUpdate, checkUpdateFlag, no, checkUpdateHelp)
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, opts); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, opts options) error {
	const separator, address, root = ",", "127.0.0.1:5353", 0
	if opts.forwarder {
		if os.Geteuid() == root {
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
		if os.Geteuid() == root {
			return os.ErrPermission
		}
		return runDemo(ctx, config.Demo())
	}
	if os.Geteuid() != root {
		return os.ErrPermission
	}
	cfg, err := config.Load(opts.config)
	if err != nil {
		return err
	}
	if opts.checkUpdate {
		return checkUpdate(cfg)
	}
	if opts.checkConfig {
		return nil
	}
	return runProtected(ctx, cfg)
}
