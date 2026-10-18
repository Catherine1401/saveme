package update

import (
	"os"
	"path/filepath"
	"strings"
)

func ApplyBundle(paths Layout, machine Machine, source string) error {
	const cache = "cache"
	const bin = "bin"
	const sessions = "sessions.lock"
	const name, mode = "VERSION", 0700
	data, err := os.ReadFile(filepath.Join(source, name))
	if err != nil {
		return err
	}
	version := strings.TrimSpace(string(data))
	if _, err := ParseVersion(version); err != nil {
		return err
	}

	unlock, err := Session(filepath.Join(paths.Root, sessions), true)
	if err != nil {
		return err
	}
	defer unlock()

	target := filepath.Join(paths.Root, cache, version)
	if err := os.MkdirAll(filepath.Join(target, bin), mode); err != nil {
		return err
	}
	for _, member := range payloadFiles() {
		const binaryMode = 0755
		if err := copyFile(filepath.Join(source, filepath.FromSlash(member)), filepath.Join(target, filepath.FromSlash(member)), binaryMode); err != nil {
			return err
		}
	}
	if err := sealStage(target, version); err != nil {
		return err
	}
	transaction := Transaction{Paths: paths, Machine: machine}
	if err := transaction.Recover(); err != nil {
		return err
	}
	if current, err := paths.ReadVersion(); err == nil && current == version {
		return nil
	}
	return transaction.Apply(version, target)
}
