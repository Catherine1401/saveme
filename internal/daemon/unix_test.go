package daemon

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"saveme/internal/core"
)

func unixServer(t *testing.T, reject bool) string {
	const name, one, mode, timeout = "saveme.sock", 1, 0600, 3 * time.Second
	t.Helper()
	f := newFixture(t)
	path := filepath.Join(t.TempDir(), name)
	server := &Server{Engine: f.engine, UID: uint32(os.Getuid())}
	if err := server.Listen(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != mode {
		t.Fatal(info, err)
	}
	if reject {
		server.UID++
	}
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan error, one)
	go func() { finished <- server.Serve(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-finished:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(timeout):
			t.Error(core.ErrInvalid)
		}
		server.Close()
	})
	return path
}

func TestUnixSocketPeerAndConfirmation(t *testing.T) {
	const duration = "12h"
	path := unixServer(t, false)
	client, err := Dial(path)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	response, err := client.Call(core.Request{Cmd: core.CreateRange, Duration: duration})
	if err != nil || !response.Pending {
		t.Fatal(response, err)
	}
	response, err = client.Call(core.Request{Cmd: core.Confirm, Code: response.Code})
	if err != nil || !response.Active || response.State != nil {
		t.Fatal(response, err)
	}
}

func TestUnixSocketRejectsWrongPeerAndSecondDaemon(t *testing.T) {
	path := unixServer(t, true)
	client, err := Dial(path)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Call(core.Request{Cmd: core.Status}); err == nil {
		t.Fatal(core.ErrUnauthorized)
	}
	second := &Server{Engine: newFixture(t).engine, UID: uint32(os.Getuid())}
	if err := second.Listen(path); err == nil {
		second.Close()
		t.Fatal(core.ErrHasLock)
	}
}
