package main

import (
	"errors"
	"os"
	"syscall"
)

func updatePending() bool {
	const path = "/run/saveme-update.lock"
	return updateHeld(path)
}

func updateHeld(path string) bool {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	if err != nil {
		return true
	}
	defer file.Close()
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil
}
