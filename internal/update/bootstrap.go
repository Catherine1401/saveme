package update

import (
	"os"
	"path/filepath"
	"strings"
)

func Bootstrap(paths Layout, source string) error {
	const sessions = "sessions.lock"
	const client, daemon, helper = "saveme", "saved", "saveme-update"
	const helperName, sessionMode, pointerMode = "helper", 0644, 0644
	const (
		literal1 = ""
	)

	const versionFile, folder, bin, mode, binaryMode = "VERSION", "versions", "bin", 0755, 0755
	data, err := os.ReadFile(filepath.Join(source, versionFile))
	if err != nil {
		return err
	}
	version := strings.TrimSpace(string(data))
	if _, err := ParseVersion(version); err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Join(paths.Root, sessions)); err == nil {
		unlock, err := Session(filepath.Join(paths.Root, sessions), true)
		if err != nil {
			return err
		}
		defer unlock()
	}
	if source == literal1 {
		return os.ErrInvalid
	}
	target := filepath.Join(paths.Root, folder, version)
	if err := os.MkdirAll(target, mode); err != nil {
		return err
	}
	for _, directory := range []string{paths.Root, filepath.Join(paths.Root, folder), target} {
		if err := os.Chmod(directory, mode); err != nil {
			return err
		}
	}

	for _, name := range []string{client, daemon, helper} {
		if err := copyFile(filepath.Join(source, bin, executable(name)), filepath.Join(target, executable(name)), binaryMode); err != nil {
			return err
		}
	}

	if err := copyFile(filepath.Join(source, bin, executable(helper)), filepath.Join(paths.Root, executable(helperName)), binaryMode); err != nil {
		return err
	}
	if err := atomicWrite(paths.current(), data); err != nil {
		return err
	}
	if err := os.Chmod(paths.current(), pointerMode); err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(paths.Root, sessions), os.O_CREATE|os.O_RDONLY, sessionMode)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Chmod(filepath.Join(paths.Root, sessions), sessionMode)
}
