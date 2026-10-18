package update

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type testInstaller struct {
	calls             int
	fail, recoverFail bool
	busy              bool
}

func (i *testInstaller) Apply(string, string) error {
	i.calls++
	if i.busy {
		return OtherSessions
	}
	if i.fail {
		return os.ErrInvalid
	}
	return nil
}
func (i *testInstaller) Recover() error {
	if i.recoverFail {
		return os.ErrInvalid
	}
	return nil
}

func testLayout(t *testing.T) Layout {
	const current = "current"
	const receipt = "VERSION"
	const sessions = "sessions.lock"
	t.Helper()

	root := t.TempDir()
	paths := Layout{Root: root, Receipt: filepath.Join(root, receipt)}
	for _, name := range []string{current, receipt, sessions} {
		const old = "v0.1.4"
		const mode = 0644
		if err := os.WriteFile(filepath.Join(root, name), []byte(old), mode); err != nil {
			t.Fatal(err)
		}
	}
	return paths
}

func testEngine(t *testing.T) (*Engine, *testInstaller) {
	const tag, empty = "v0.2.0", ""
	t.Helper()

	installer := &testInstaller{}
	engine := &Engine{Paths: testLayout(t), Source: fixtureSource(t, tag, tag, empty, empty), Install: installer}
	if err := engine.Initialize(); err != nil {
		t.Fatal(err)
	}
	return engine, installer
}

func TestBackgroundStageAppliesOnlyOnNextOpen(t *testing.T) {
	const ready, version, zero, one = "ready", "v0.2.0", 0, 1
	engine, installer := testEngine(t)
	_, first := engine.Open(context.Background(), true)
	first()
	engine.Wait()
	if installer.calls != zero || engine.State().Phase != ready {
		t.Fatal(engine.State(), installer)
	}
	state, second := engine.Open(context.Background(), true)
	second()
	engine.Wait()
	if state.Installed != version || installer.calls != one {
		t.Fatal(state, installer)
	}
}

func TestSessionsDeferApplyIncludingHelperRestart(t *testing.T) {
	const deferred, sessions, zero, one = "other_sessions", "sessions.lock", 0, 1
	engine, installer := testEngine(t)
	_, release := engine.Open(context.Background(), true)
	engine.Wait()
	state, second := engine.Open(context.Background(), true)
	if state.Deferred != deferred || installer.calls != zero {
		t.Fatal(state, installer)
	}
	second()
	release()
	unlock, err := Session(filepath.Join(engine.Paths.Root, sessions), false)
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.Initialize(); err != nil {
		t.Fatal(err)
	}
	state, third := engine.Open(context.Background(), true)
	third()
	if state.Deferred != deferred || installer.calls != zero {
		t.Fatal(state, installer)
	}
	unlock()
	state, fourth := engine.Open(context.Background(), true)
	fourth()
	engine.Wait()
	if installer.calls != one {
		t.Fatal(state, installer)
	}
}

func TestDisabledAutomaticAndOfflineFailureDoNotApply(t *testing.T) {
	const zero, ready, tag, empty, missing = 0, "ready", "v0.2.0", "", "missing"
	engine, installer := testEngine(t)
	_, release := engine.Open(context.Background(), false)
	release()
	engine.Wait()
	if engine.State().Phase == ready || installer.calls != zero {
		t.Fatal(engine.State())
	}
	engine.Source = fixtureSource(t, tag, tag, empty, missing)
	_, release = engine.Open(context.Background(), true)
	release()
	engine.Wait()
	state := engine.State()
	if state.Fatal || state.LastError == empty || installer.calls != zero {
		t.Fatal(state)
	}
}

func TestRollbackFailureIsReportedAndBadVersionIsQuarantined(t *testing.T) {
	const tag, empty, zero, one = "v0.2.0", "", 0, 1
	engine, installer := testEngine(t)
	engine.Check(context.Background(), true)
	engine.Wait()
	installer.fail = true
	state, release := engine.Open(context.Background(), true)
	release()
	engine.Wait()
	if installer.calls != one || state.Rejected != tag || state.Fatal {
		t.Fatal(state, installer)
	}
	_, release = engine.Open(context.Background(), true)
	release()
	engine.Wait()
	if installer.calls != one || engine.State().LastError == empty {
		t.Fatal(installer, engine.State())
	}
	installer.recoverFail = true
	if err := engine.Initialize(); err != nil {
		t.Fatal(err)
	}
	state, release = engine.Open(context.Background(), true)
	if !state.Fatal || release != nil || state.LastError == empty {
		t.Fatal(state)
	}
	_ = zero
}

func TestDirectDaemonSessionDefersWithoutQuarantiningRelease(t *testing.T) {
	const ready, tag, empty, deferred = "ready", "v0.2.0", "", "other_sessions"
	engine, installer := testEngine(t)
	engine.Check(t.Context(), true)
	engine.Wait()
	installer.busy = true
	state, release := engine.Open(t.Context(), true)
	release()
	if state.Phase != ready || state.Available != tag || state.Rejected != empty || state.Deferred != deferred {
		t.Fatal(state)
	}
	installer.busy = false
	state, release = engine.Open(t.Context(), true)
	release()
	engine.Wait()
	if state.Installed != tag {
		t.Fatal(state)
	}
}
