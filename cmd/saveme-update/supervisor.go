package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"saveme/internal/update"
)

func supervise(ctx context.Context, configPath string) error {
	paths := update.Installed()
	for {
		const helperName = "saveme-update"
		const workerFlag = "-worker"
		const configFlag = "-config"
		const interval = time.Second
		const suffix = ".exe"
		const buffer = 1
		version, err := paths.ReadVersion()
		if err != nil {
			return err
		}
		name := helperName

		if filepath.Ext(paths.Launcher) == suffix {
			name += suffix
		}
		command := exec.Command(filepath.Join(filepath.Dir(paths.Payload(version)), name), workerFlag, configFlag, configPath)
		command.Stdout, command.Stderr = log.Writer(), log.Writer()
		input, err := command.StdinPipe()
		if err != nil {
			return err
		}
		if err := command.Start(); err != nil {
			input.Close()
			return err
		}

		finished := make(chan error, buffer)
		go func() { finished <- command.Wait() }()
		ticker := time.NewTicker(interval)
		restart := false
		for !restart {
			select {
			case err := <-finished:
				ticker.Stop()
				input.Close()
				return err
			case <-ctx.Done():
				ticker.Stop()
				input.Close()
				<-finished
				return nil
			case <-ticker.C:
				if workerChanged(paths, version) {
					restart = true
				}
			}
		}
		ticker.Stop()
		input.Close()
		if err := <-finished; err != nil {
			return err
		}
	}
}

func workerChanged(paths update.Layout, version string) bool {
	const transaction = "transaction"
	if _, err := os.Stat(filepath.Join(paths.Root, transaction)); !os.IsNotExist(err) {
		return false
	}
	current, err := paths.ReadVersion()
	return err == nil && current != version
}

func watchSupervisor(cancel context.CancelFunc) {
	const bufferSize = 1
	buffer := make([]byte, bufferSize)
	_, _ = io.ReadFull(os.Stdin, buffer)
	cancel()
}
