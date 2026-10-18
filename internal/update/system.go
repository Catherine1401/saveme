package update

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"saveme/internal/config"
	"saveme/internal/core"
)

type System struct {
	Paths  Layout
	Config config.Config
}

func outputError(format string, err error, output []byte) error {
	return fmt.Errorf(format, err, output)
}

func (s System) Check(checker string, arguments ...string) error {
	const timeout = 90 * time.Second
	const verify = "-verify-update"
	const zero = 0
	if len(arguments) > zero && arguments[zero] == verify {
		const literal1 = "transaction"
		const statusFlag = "-update-status"
		const statusName = "status.json"
		data, err := s.readStatus()
		if err != nil {
			return err
		}
		path := filepath.Join(s.Paths.Root, literal1, statusName)
		if err := atomicWrite(path, data); err != nil {
			return err
		}
		arguments = append(append([]string{}, arguments...), statusFlag, path)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, checker, arguments...)
	output := &limitedOutput{}
	command.Stdout, command.Stderr = output, output
	err := command.Run()
	if err != nil {
		const format = "%w: %s"
		return outputError(format, err, []byte(output.String()))
	}
	return nil
}

func (s System) Ready() error {
	const attempts = 20
	for range attempts {
		const interval = time.Second
		if data, err := s.readStatus(); err == nil {
			status, err := decodeStatus(data)
			if err == nil && (status.OK || status.Error == core.ErrClockUnsync) {
				return nil
			}
		}
		time.Sleep(interval)
	}
	return core.ErrEnforcement
}

func (s System) Idle() error {
	const ownConnection = 1
	data, err := s.readStatus()
	if err != nil {
		return err
	}
	status, err := decodeStatus(data)
	if err != nil {
		return err
	}
	if status.Clients > ownConnection {
		return OtherSessions
	}
	return nil
}
