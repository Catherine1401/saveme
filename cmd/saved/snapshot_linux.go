package main

import (
	"os"
	"saveme/internal/config"
	"syscall"
)

func stoppedUpdate(cfg config.Config, action func() error) error {
	const suffix, mode = ".lock", 0600
	file, err := os.OpenFile(cfg.Socket+suffix, os.O_RDWR, mode)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return err
	}
	return action()
}
