package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Machine interface {
	Lock() (func(), error)
	Idle() error
	Stop() error
	Start() error
	Ready() error
	Protect() error
	Unprotect() error
	Check(checker string, arguments ...string) error
}

type Journal struct {
	Before   string `json:"before"`
	After    string `json:"after"`
	Phase    string `json:"phase"`
	Snapshot bool   `json:"snapshot"`
	Receipt  bool   `json:"receipt"`
}

type Transaction struct {
	Paths   Layout
	Machine Machine
}

func (t Transaction) folder() string {
	const name = "transaction"
	return filepath.Join(t.Paths.Root, name)
}

func (t Transaction) record(journal Journal) error {
	const name = "journal.json"
	return writeJSON(filepath.Join(t.folder(), name), journal)
}

func copyFile(source, target string, mode os.FileMode) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := atomicWrite(target, data); err != nil {
		return err
	}
	return os.Chmod(target, mode)
}

func (t Transaction) prepare(version, stage string) (Journal, error) {
	const mode = 0700
	const executableMode = 0755
	const name = "saved"
	const checkerName = "checker"
	const prepared = "prepared"
	const bin, checkFlag, configFlag = "bin", "-check-preserved-update", "-config"
	before, err := t.Paths.ReadVersion()
	journal := Journal{Before: before, After: version, Phase: prepared}
	if err != nil || !Newer(version, before) {
		return journal, os.ErrInvalid
	}
	if err := os.MkdirAll(t.folder(), mode); err != nil {
		return journal, err
	}
	checker := filepath.Join(t.folder(), executable(checkerName))

	if err := copyFile(filepath.Join(stage, bin, executable(name)), checker, executableMode); err != nil {
		return journal, err
	}
	if err := t.Machine.Check(checker, checkFlag, configFlag, t.Paths.Config); err != nil {
		return journal, err
	}
	if err := copyFile(t.Paths.Daemon, filepath.Join(t.folder(), executable(name)), executableMode); err != nil {
		return journal, err
	}
	if _, err := os.Stat(t.Paths.Receipt); err == nil {
		const receiptName = "VERSION"
		const dataMode = 0600
		if err := copyFile(t.Paths.Receipt, filepath.Join(t.folder(), receiptName), dataMode); err != nil {
			return journal, err
		}
		journal.Receipt = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return journal, err
	}
	return journal, t.record(journal)
}

func (t Transaction) Apply(version, stage string) error {
	unlock, err := t.Machine.Lock()
	if err != nil {
		return err
	}
	defer unlock()
	if err := t.Machine.Idle(); err != nil {
		return err
	}
	journal, err := t.prepare(version, stage)
	if err != nil {
		const journalName = "journal.json"
		if _, check := os.Stat(filepath.Join(t.folder(), journalName)); os.IsNotExist(check) {
			os.RemoveAll(t.folder())
		}
		return err
	}
	if err := t.install(&journal, stage); err != nil {
		if recovery := t.rollback(journal); recovery != nil {
			const format = "update failed: %v; recovery required: %w"
			return fmt.Errorf(format, err, recovery)
		}
		return err
	}
	return os.RemoveAll(t.folder())
}

func (t Transaction) install(journal *Journal, stage string) error {
	const folder = "versions"
	const mode = 0755
	const daemon = "saved"
	const client = "saveme"
	const helper = "saveme-update"
	const binaryMode = 0755
	const stopping, replacing, committed = "stopping", "replacing", "committed"
	const checkerName, snapshot, snapshotFlag, configFlag = "checker", "commitment.json", "-update-snapshot", "-config"
	checker := filepath.Join(t.folder(), executable(checkerName))
	if err := t.Machine.Protect(); err != nil {
		return err
	}
	journal.Phase = stopping
	if err := t.record(*journal); err != nil {
		return err
	}
	if err := t.Machine.Stop(); err != nil {
		return err
	}
	if err := t.Machine.Check(checker, snapshotFlag, filepath.Join(t.folder(), snapshot), configFlag, t.Paths.Config); err != nil {
		return err
	}
	journal.Snapshot, journal.Phase = true, replacing
	if err := t.record(*journal); err != nil {
		return err
	}

	payload := filepath.Join(t.Paths.Root, folder, journal.After)
	if err := os.MkdirAll(payload, mode); err != nil {
		return err
	}
	if err := os.Chmod(payload, mode); err != nil {
		return err
	}
	for _, member := range []string{client, daemon, helper} {
		const bin = "bin"
		if err := copyFile(filepath.Join(stage, bin, executable(member)), filepath.Join(payload, executable(member)), binaryMode); err != nil {
			return err
		}
	}
	if err := copyFile(filepath.Join(payload, executable(daemon)), t.Paths.Daemon, binaryMode); err != nil {
		return err
	}
	if err := t.Machine.Start(); err != nil {
		return err
	}
	if err := t.verify(checker); err != nil {
		return err
	}
	if err := t.switchVersion(journal.After); err != nil {
		return err
	}
	journal.Phase = committed
	if err := t.record(*journal); err != nil {
		return err
	}
	return t.Machine.Unprotect()
}

func (t Transaction) switchVersion(version string) error {
	const mode, newline = 0644, "\n"
	data := []byte(version + newline)
	if err := atomicWrite(t.Paths.Receipt, data); err != nil {
		return err
	}
	if err := os.Chmod(t.Paths.Receipt, mode); err != nil {
		return err
	}
	if err := atomicWrite(t.Paths.current(), data); err != nil {
		return err
	}
	return os.Chmod(t.Paths.current(), mode)
}

func (t Transaction) verify(checker string) error {
	const verify, snapshot, config = "-verify-update", "commitment.json", "-config"
	if err := t.Machine.Ready(); err != nil {
		return err
	}
	return t.Machine.Check(checker, verify, filepath.Join(t.folder(), snapshot), config, t.Paths.Config)
}

func (t Transaction) rollback(journal Journal) error {
	const prepared = "prepared"
	const daemon = "saved"
	const checkerName = "checker"
	const binaryMode = 0755
	if journal.Phase == prepared {
		if err := t.Machine.Unprotect(); err != nil {
			return err
		}
		return os.RemoveAll(t.folder())
	}
	if err := t.Machine.Protect(); err != nil {
		return err
	}
	if err := t.Machine.Stop(); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(t.folder(), executable(daemon)), t.Paths.Daemon, binaryMode); err != nil {
		return err
	}
	checker := filepath.Join(t.folder(), executable(checkerName))
	if journal.Snapshot {
		const snapshot = "commitment.json"
		const restore = "-restore-update"
		const configFlag = "-config"
		if err := t.Machine.Check(checker, restore, filepath.Join(t.folder(), snapshot), configFlag, t.Paths.Config); err != nil {
			return err
		}
	}
	if err := t.switchVersion(journal.Before); err != nil {
		return err
	}
	if !journal.Receipt {
		if err := os.Remove(t.Paths.Receipt); err != nil {
			return err
		}
	}
	if err := t.Machine.Start(); err != nil {
		return err
	}
	if journal.Snapshot {
		if err := t.verify(checker); err != nil {
			return err
		}
	} else if err := t.Machine.Ready(); err != nil {
		return err
	}
	if err := t.Machine.Unprotect(); err != nil {
		return err
	}
	return os.RemoveAll(t.folder())
}

func (t Transaction) Recover() error {
	const prepared, stopping, replacing, committed = "prepared", "stopping", "replacing", "committed"
	const name = "journal.json"
	data, err := os.ReadFile(filepath.Join(t.folder(), name))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var journal Journal
	if err := json.Unmarshal(data, &journal); err != nil {
		return err
	}
	if _, err := ParseVersion(journal.Before); err != nil {
		return err
	}
	if _, err := ParseVersion(journal.After); err != nil {
		return err
	}

	if !Newer(journal.After, journal.Before) {
		return os.ErrInvalid
	}
	switch journal.Phase {
	case prepared, stopping, replacing, committed:
	default:
		return os.ErrInvalid
	}
	unlock, err := t.Machine.Lock()
	if err != nil {
		return err
	}
	defer unlock()
	return t.rollback(journal)
}
