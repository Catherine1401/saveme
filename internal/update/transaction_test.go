package update

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testMachine struct {
	guard                                        bool
	paths                                        Layout
	fail                                         string
	failed                                       bool
	restoreFail                                  bool
	stops, starts, snapshots, restores, verifies int
}

func (m *testMachine) Lock() (func(), error) { return func() {}, nil }
func (m *testMachine) Ready() error          { return nil }
func (m *testMachine) Stop() error           { m.stops++; return nil }
func (m *testMachine) Start() error {
	const start = "start"
	m.starts++

	if m.fail == start && !m.failed {
		m.failed = true
		return os.ErrInvalid
	}
	return nil
}
func (m *testMachine) Check(_ string, arguments ...string) error {
	const check = "-check-preserved-update"
	const snapshot = "-update-snapshot"
	const restore = "-restore-update"
	const verify = "-verify-update"
	const zero = 0
	switch arguments[zero] {
	case check:
		if m.fail == check {
			return os.ErrInvalid
		}
	case snapshot:
		const pathIndex = 1
		const state = "state"
		const mode = 0600
		m.snapshots++
		return os.WriteFile(arguments[pathIndex], []byte(state), mode)
	case restore:
		m.restores++
		if m.restoreFail {
			return os.ErrInvalid
		}
	case verify:
		m.verifies++
		if m.fail == verify && !m.failed {
			m.failed = true
			return os.ErrInvalid
		}
	}
	return nil
}

func transactionFixture(t *testing.T) (Transaction, *testMachine, string) {
	const name, old, mode, tag, empty = "saved", "old daemon", 0755, "v0.2.0", ""
	t.Helper()

	paths := testLayout(t)
	paths.Daemon = filepath.Join(paths.Root, executable(name))
	if err := os.WriteFile(paths.Daemon, []byte(old), mode); err != nil {
		t.Fatal(err)
	}
	stage, err := fixtureSource(t, tag, tag, empty, empty).Stage(t.Context(), paths.Root, tag)
	if err != nil {
		t.Fatal(err)
	}
	machine := &testMachine{paths: paths}
	return Transaction{Paths: paths, Machine: machine}, machine, stage
}

func TestTransactionCommitsTogetherAndCleansJournal(t *testing.T) {
	const version, zero, one = "v0.2.0", 0, 1
	transaction, machine, stage := transactionFixture(t)
	if err := transaction.Apply(version, stage); err != nil {
		t.Fatal(err)
	}
	current, err := transaction.Paths.ReadVersion()
	if err != nil || current != version || machine.snapshots != one || machine.verifies != one || machine.restores != zero {
		t.Fatal(current, err, machine)
	}
	data, err := os.ReadFile(transaction.Paths.Daemon)
	if err != nil || !bytes.Equal(data, testBinary()) {
		t.Fatal(string(data), err)
	}
	if _, err := os.Stat(transaction.folder()); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestTransactionFailureRestoresBinariesPointerAndCommitment(t *testing.T) {
	const check = "-check-preserved-update"
	const start = "start"
	const verify = "-verify-update"
	for _, failure := range []string{check, start, verify} {
		t.Run(failure, func(t *testing.T) {
			const version = "v0.2.0"
			const before = "v0.1.4"
			const original = "old daemon"
			const zero, one = 0, 1
			transaction, machine, stage := transactionFixture(t)
			machine.fail = failure
			if err := transaction.Apply(version, stage); err == nil {
				t.Fatal(machine)
			}
			current, err := transaction.Paths.ReadVersion()
			if err != nil || current != before {
				t.Fatal(current, err)
			}
			data, err := os.ReadFile(transaction.Paths.Daemon)
			if err != nil || string(data) != original {
				t.Fatal(string(data), err)
			}

			if failure == check && machine.stops != zero {
				t.Fatal(machine)
			}
			if failure != check && machine.restores != one {
				t.Fatal(machine)
			}
		})
	}
}

func TestRecoveryRetainsBackupOnRestoreFailureAndCanRetry(t *testing.T) {
	const (
		literal1 = "recovery required"
	)

	const version, failing, original, before = "v0.2.0", "-verify-update", "old daemon", "v0.1.4"
	transaction, machine, stage := transactionFixture(t)
	machine.fail, machine.restoreFail = failing, true
	if err := transaction.Apply(version, stage); err == nil || !strings.Contains(err.Error(), literal1) {
		t.Fatal(err)
	}
	if _, err := os.Stat(transaction.folder()); err != nil {
		t.Fatal(err)
	}
	machine.restoreFail = false
	if err := transaction.Recover(); err != nil {
		t.Fatal(err)
	}
	current, err := transaction.Paths.ReadVersion()
	if err != nil || current != before {
		t.Fatal(current, err)
	}
	data, err := os.ReadFile(transaction.Paths.Daemon)
	if err != nil || string(data) != original {
		t.Fatal(string(data), err)
	}
	if _, err := os.Stat(transaction.folder()); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestRecoveryAfterInterruptedReplacement(t *testing.T) {
	const original, one = "old daemon", 1
	const version, phase, daemon, mode, checker, snapshot, snapshotFlag = "v0.2.0", "replacing", "interrupted replacement", 0755, "checker", "commitment.json", "-update-snapshot"
	transaction, machine, stage := transactionFixture(t)
	journal, err := transaction.prepare(version, stage)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Check(filepath.Join(transaction.folder(), executable(checker)), snapshotFlag, filepath.Join(transaction.folder(), snapshot)); err != nil {
		t.Fatal(err)
	}
	journal.Phase, journal.Snapshot = phase, true
	if err := transaction.record(journal); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(transaction.Paths.Daemon, []byte(daemon), mode); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Recover(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(transaction.Paths.Daemon)
	if err != nil || string(data) != original || machine.restores != one {
		t.Fatal(string(data), err, machine)
	}
}

func (m *testMachine) Protect() error   { m.guard = true; return nil }
func (m *testMachine) Unprotect() error { m.guard = false; return nil }
func (m *testMachine) Idle() error      { return nil }
