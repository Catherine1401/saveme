package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata"

	"saveme/internal/config"
	"saveme/internal/update"
)

func main() {
	const applyFlag, applyHelp = "apply-bundle", "Apply a verified bundle during an elevated manual upgrade"
	const workerFlag, workerHelp = "worker", "Run the versioned update worker under the service supervisor"
	const configFlag, configHelp, installFlag, installHelp, sourceFlag, sourceHelp, empty = "config", "Daemon configuration file", "install-runtime", "Install the protected runtime from a verified bundle", "source", "Verified bundle directory", ""
	configPath := flag.String(configFlag, config.DefaultConfig, configHelp)
	install := flag.Bool(installFlag, false, installHelp)
	source := flag.String(sourceFlag, empty, sourceHelp)

	apply := flag.String(applyFlag, empty, applyHelp)

	worker := flag.Bool(workerFlag, false, workerHelp)
	flag.Parse()
	if !privileged() {
		log.Fatal(os.ErrPermission)
	}
	if *apply != empty {
		cfg, err := config.Load(*configPath)
		if err != nil {
			log.Fatal(err)
		}
		paths := update.Installed()
		if err := update.ApplyBundle(paths, update.System{Paths: paths, Config: cfg}, *apply); err != nil {
			log.Fatal(err)
		}
		return
	}
	if *install {
		if err := update.Bootstrap(update.Installed(), *source); err != nil {
			log.Fatal(err)
		}
		return
	}
	if *worker {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go watchSupervisor(cancel)
		if err := runWorker(ctx, *configPath); err != nil {
			log.Fatal(err)
		}
		return
	}
	if handled, err := serviceMode(*configPath); handled || err != nil {
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, *configPath); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, path string) error { return supervise(ctx, path) }
func runWorker(ctx context.Context, path string) error {
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	paths := update.Installed()
	engine := &update.Engine{Paths: paths, Source: update.GitHub(), Install: update.Transaction{Paths: paths, Machine: update.System{Paths: paths, Config: cfg}}}
	if err := engine.Initialize(); err != nil {
		return err
	}
	defer engine.Wait()
	return update.Serve(ctx, cfg, engine)
}
