package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

// Tiến trình đang chạy với token đã nâng quyền (Administrator hoặc SYSTEM).
func privileged() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

type service struct{ opts options }

// Chạy dưới Service Control Manager; trả false khi được gọi từ terminal.
func runService(opts options) (bool, error) {
	const name, logName, mode = "saved", "saved.log", 0600
	inService, err := svc.IsWindowsService()
	if err != nil || !inService {
		return false, err
	}
	logFile, err := os.OpenFile(filepath.Join(filepath.Dir(opts.config), logName), os.O_CREATE|os.O_APPEND|os.O_WRONLY, mode)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(logFile)
	}
	return true, svc.Run(name, service{opts: opts})
}

// Nhận lệnh dừng từ SCM và chờ daemon thoát.
func (s service) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	const accepted, failure, success = svc.AcceptStop | svc.AcceptShutdown, 1, 0
	const buffer = 1
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	status <- svc.Status{State: svc.StartPending}
	finished := make(chan error, buffer)
	go func() { finished <- run(ctx, s.opts) }()
	status <- svc.Status{State: svc.Running, Accepts: accepted}
	for {
		select {
		case err := <-finished:
			if err != nil {
				log.Print(err)
				return false, failure
			}
			return false, success
		case request := <-requests:
			switch request.Cmd {
			case svc.Interrogate:
				status <- request.CurrentStatus
			case svc.Stop, svc.Shutdown:
				status <- svc.Status{State: svc.StopPending}
				cancel()
			}
		}
	}
}
