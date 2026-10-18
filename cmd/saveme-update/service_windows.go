package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"saveme/internal/update"
)

func privileged() bool { return windows.GetCurrentProcessToken().IsElevated() }

type updaterService struct{ config string }

func serviceMode(config string) (bool, error) {
	const name, logName, mode = "saveme-updater", "updater.log", 0600
	inService, err := svc.IsWindowsService()
	if err != nil || !inService {
		return false, err
	}
	file, err := os.OpenFile(filepath.Join(filepath.Dir(config), logName), os.O_CREATE|os.O_APPEND|os.O_WRONLY, mode)
	if err != nil {
		return true, err
	}
	defer file.Close()
	log.SetOutput(file)
	return true, svc.Run(name, updaterService{config: config})
}

func (s updaterService) Execute(_ []string, requests <-chan svc.ChangeRequest, statuses chan<- svc.Status) (bool, uint32) {
	const interval = 100 * time.Millisecond
	const waitHint = 10000
	const buffer = 1
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pending := svc.Status{State: svc.StartPending, WaitHint: waitHint}
	statuses <- pending
	finished := make(chan error, buffer)
	go func() { finished <- run(ctx, s.config) }()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		const failure = 1
		select {
		case <-ticker.C:
			const operation = "status"
			const empty = ""
			const accepted = svc.AcceptStop | svc.AcceptShutdown
			response, connection, err := update.Connect(update.Installed(), update.Request{Operation: operation})
			if err != nil {
				pending.CheckPoint++
				statuses <- pending
				continue
			}
			connection.Close()
			if response.Error != empty {
				log.Print(response.Error)
				return false, failure
			}
			ticker.Stop()
			statuses <- svc.Status{State: svc.Running, Accepts: accepted}
		case err := <-finished:
			const success = 0
			if err != nil {
				log.Print(err)
				return false, failure
			}
			return false, success
		case request := <-requests:
			switch request.Cmd {
			case svc.Interrogate:
				statuses <- request.CurrentStatus
			case svc.Stop, svc.Shutdown:
				statuses <- svc.Status{State: svc.StopPending}
				cancel()
			}
		}
	}
}
