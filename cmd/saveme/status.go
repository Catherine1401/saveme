package main

import (
	"encoding/json"
	"io"

	"saveme/internal/core"
	"saveme/internal/daemon"
)

func printStatus(client *daemon.Client, output io.Writer) error {
	response, err := client.Call(core.Request{Cmd: core.Status})
	if err != nil {
		return err
	}
	if err := json.NewEncoder(output).Encode(response); err != nil {
		return err
	}
	if !response.OK && response.Error != core.ErrClockUnsync {
		return response.Error
	}
	return nil
}
