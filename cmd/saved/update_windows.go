package main

import (
	"saveme/internal/config"
	"saveme/internal/core"
)

func snapshotUpdate(config.Config, string) error       { return core.ErrInvalid }
func restoreUpdate(config.Config, string) error        { return core.ErrInvalid }
func verifyUpdate(config.Config, string, string) error { return core.ErrInvalid }
