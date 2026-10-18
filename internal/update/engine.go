package update

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Installer interface {
	Apply(version, stage string) error
	Recover() error
}

type Downloader interface {
	Latest(context.Context) (Release, error)
	Stage(context.Context, string, string) (string, error)
}

type Engine struct {
	Paths    Layout
	Source   Downloader
	Install  Installer
	mu       sync.Mutex
	workers  sync.WaitGroup
	state    Status
	sessions int
	checking bool
}

func (e *Engine) Initialize() error {
	const name = "update.json"
	const idle = "idle"
	data, err := os.ReadFile(filepath.Join(e.Paths.Root, name))
	if err == nil {
		err = json.Unmarshal(data, &e.state)
	}
	if err != nil && !os.IsNotExist(err) {
		e.state = Status{LastError: err.Error()}
	}
	e.state.Fatal, e.state.Phase = false, idle
	if err := e.Install.Recover(); err != nil {
		const recovery = "recovery_required"
		e.state.Fatal, e.state.Phase, e.state.LastError = true, recovery, err.Error()
	}
	version, err := e.Paths.ReadVersion()
	if err != nil {
		return err
	}
	e.state.Installed = version
	if Newer(e.state.Available, version) && !e.state.Fatal {
		const ready = "ready"
		e.state.Phase = ready
	} else {
		const empty = ""
		e.state.Available = empty
	}
	return e.save()
}

func (e *Engine) save() error {
	const name = "update.json"
	return writeJSON(filepath.Join(e.Paths.Root, name), e.state)
}

func (e *Engine) State() Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state
}

func (e *Engine) Open(ctx context.Context, automatic bool) (Status, func()) {
	const empty = ""
	const ready = "ready"
	e.mu.Lock()
	defer e.mu.Unlock()
	e.state.Deferred = empty
	if automatic && e.state.Phase == ready && !e.state.Fatal {
		const zero = 0
		const deferred = "other_sessions"
		if e.sessions == zero {
			const lockName = "sessions.lock"
			unlock, err := Session(filepath.Join(e.Paths.Root, lockName), true)
			if err == nil {
				e.apply()
				unlock()
			} else {
				e.state.Deferred = deferred
			}
		} else {
			e.state.Deferred = deferred
		}
	}
	if e.state.Fatal {
		return e.state, nil
	}
	e.sessions++
	release := func() { e.mu.Lock(); e.sessions--; e.mu.Unlock() }
	if automatic {
		e.check(ctx, false)
	}
	return e.state, release
}

func (e *Engine) apply() {
	const applying = "applying"
	const idle = "idle"
	const cache = "cache"
	const empty = ""
	version := e.state.Available
	stage := filepath.Join(e.Paths.Root, cache, version)
	if err := verifyStage(stage, version); err != nil {
		e.state.Phase, e.state.Available, e.state.LastError = idle, empty, err.Error()
		e.save()
		return
	}
	e.state.Phase = applying
	if err := e.save(); err != nil {
		e.state.LastError, e.state.Phase = err.Error(), idle
		return
	}
	err := e.Install.Apply(version, stage)
	if err == OtherSessions {
		const ready = "ready"
		e.state.Phase, e.state.Deferred = ready, OtherSessions.Error()
		e.save()
		return
	}
	e.state.Phase, e.state.Available = idle, empty
	if err != nil {
		const transaction = "transaction"
		e.state.Rejected, e.state.LastError = version, err.Error()

		if _, check := os.Stat(filepath.Join(e.Paths.Root, transaction)); check == nil {
			const recovery = "recovery_required"
			e.state.Fatal, e.state.Phase = true, recovery
		}
	} else {
		e.state.Installed, e.state.LastError, e.state.Rejected = version, empty, empty
	}
	if err := e.save(); err != nil {
		e.state.LastError = err.Error()
	}
}

func (e *Engine) Check(ctx context.Context, force bool) Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.check(ctx, force)
	return e.state
}

func (e *Engine) check(ctx context.Context, force bool) {
	const (
		literal1 = 1
	)

	const checking, ready, empty = "checking", "ready", ""
	if e.checking || e.state.Fatal || e.state.Phase == ready {
		return
	}
	if force {
		e.state.Rejected = empty
	}
	e.checking, e.state.Phase = true, checking
	e.workers.Add(literal1)
	go func() {
		const idle = "idle"
		defer e.workers.Done()
		release, err := e.Source.Latest(ctx)
		e.mu.Lock()
		installed, rejected := e.state.Installed, e.state.Rejected
		e.mu.Unlock()
		if err == nil && Newer(release.Tag, installed) && release.Tag != rejected {
			_, err = e.Source.Stage(ctx, e.Paths.Root, release.Tag)
		}
		e.mu.Lock()
		defer e.mu.Unlock()

		message := errorText(err)
		if ctx.Err() != nil {
			message = empty
		}
		if release.Tag == e.state.Rejected && err == nil {
			message = e.state.LastError
		}
		e.checking, e.state.Phase, e.state.LastError, e.state.Available = false, idle, message, empty
		if err == nil && Newer(release.Tag, e.state.Installed) && release.Tag != e.state.Rejected {
			e.state.Available, e.state.Phase = release.Tag, ready
		}
		if err := e.save(); err != nil {
			e.state.LastError = err.Error()
		}
	}()
}

func (e *Engine) Wait() { e.workers.Wait() }
