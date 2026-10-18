package update

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func runningHelper(t *testing.T) (*Engine, Layout, context.CancelFunc) {
	const attempts = 100
	const socket = "updater.sock"
	const (
		literal1 = 1
	)

	t.Helper()

	engine, _ := testEngine(t)
	paths := engine.Paths
	paths.Endpoint = testEndpoint(filepath.Join(paths.Root, socket))
	engine.Paths = paths
	cfg := testIdentity(t)
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan error, literal1)
	go func() { finished <- Serve(ctx, cfg, engine) }()
	t.Cleanup(func() { cancel(); <-finished; engine.Wait() })

	for range attempts {
		const interval = 10 * time.Millisecond
		connection, err := dial(paths.Endpoint)
		if err == nil {
			connection.Close()
			return engine, paths, cancel
		}
		time.Sleep(interval)
	}
	t.Fatal(paths.Endpoint)
	return nil, paths, cancel
}

func TestRPCLeasesAndStrictRequests(t *testing.T) {
	const literal1 = ""
	const open = "open"
	const status = "status"
	const ready = "ready"
	const attempts = 100
	const deferred = "other_sessions"
	const unknownPath = "{\"operation\":\"check\",\"path\":\"/tmp/untrusted\"}\n"
	engine, paths, _ := runningHelper(t)
	response, lease, err := Connect(paths, Request{Operation: open, Automatic: true})
	if err != nil || response.Error != literal1 {
		t.Fatal(response, err)
	}
	engine.Wait()
	response, observer, err := Connect(paths, Request{Operation: status})
	if err != nil || response.Status.Phase != ready {
		t.Fatal(response, err)
	}
	observer.Close()
	response, second, err := Connect(paths, Request{Operation: open, Automatic: true})

	if err != nil || response.Status.Deferred != deferred {
		t.Fatal(response, err)
	}
	second.Close()
	lease.Close()
	for range attempts {
		const literal2 = 0
		const interval = 10 * time.Millisecond
		engine.mu.Lock()
		count := engine.sessions
		engine.mu.Unlock()
		if count == literal2 {
			break
		}
		time.Sleep(interval)
	}

	connection, err := dial(paths.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	connection.SetDeadline(time.Now().Add(time.Second))
	if _, err := connection.Write([]byte(unknownPath)); err != nil {
		t.Fatal(err)
	}
	var result Response
	if err := json.NewDecoder(connection).Decode(&result); err == nil {
		t.Fatal(result)
	}
	connection.Close()
}

func TestRPCShutdownClosesIdleLeases(t *testing.T) {
	const size = 1
	const open, deadline = "open", 3 * time.Second
	_, paths, cancel := runningHelper(t)
	_, lease, err := Connect(paths, Request{Operation: open})
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	cancel()
	lease.SetReadDeadline(time.Now().Add(deadline))

	buffer := make([]byte, size)
	if _, err := lease.Read(buffer); err == nil {
		t.Fatal(buffer)
	}
}

type blockedInstaller struct {
	Installer
	started, proceed chan struct{}
}

func (i blockedInstaller) Apply(version, stage string) error {
	close(i.started)
	<-i.proceed
	return i.Installer.Apply(version, stage)
}

func TestRPCShutdownPreservesApplyingReply(t *testing.T) {
	const shortWait = 30 * time.Millisecond
	const timeout = 3 * time.Second
	const buffer = 1
	engine, paths, cancel := runningHelper(t)
	engine.Check(t.Context(), false)
	engine.Wait()
	installer := blockedInstaller{Installer: engine.Install, started: make(chan struct{}), proceed: make(chan struct{})}
	engine.Install = installer
	finished := make(chan error, buffer)
	go func() {
		const open = "open"
		response, connection, err := Connect(paths, Request{Operation: open, Automatic: true})
		if err == nil {
			const installed = "v0.2.0"
			connection.Close()
			if response.Status.Installed != installed {
				err = os.ErrInvalid
			}
		}
		finished <- err
	}()
	<-installer.started
	cancel()
	select {
	case err := <-finished:
		const earlyReply = "reply closed before apply finished"
		close(installer.proceed)
		t.Fatal(earlyReply, err)
	case <-time.After(shortWait):
	}
	close(installer.proceed)
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(timeout):
		const missingReply = "apply reply did not finish"
		t.Fatal(missingReply)
	}
}

func TestSkipAutomaticModes(t *testing.T) {
	const noMotion = "--no-motion"
	const help, versionStatus, demo, socket, attached = "--help", "-status", "--demo", "-socket", "--socket=/tmp/test"
	for _, argument := range []string{help, versionStatus, demo, socket, attached} {
		if !skipAutomatic([]string{argument}) {
			t.Fatal(argument)
		}
	}

	if skipAutomatic([]string{noMotion}) {
		t.Fatal(noMotion)
	}
}
