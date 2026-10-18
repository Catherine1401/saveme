package update

import (
	"os"
	"path/filepath"
)

func Installed() Layout {
	const root, endpoint = "/usr/local/libexec/saveme", "/run/saveme-updater.sock"
	const launcher, daemon, config, receipt = "/usr/local/bin/saveme", "/usr/local/libexec/saved", "/etc/saveme/config.json", "/usr/local/share/saveme/VERSION"
	return Layout{root, endpoint, launcher, daemon, config, receipt}
}

func executable(name string) string { return name }
func replace(source, target string) error {
	if err := os.Rename(source, target); err != nil {
		return err
	}
	directory, err := os.Open(filepath.Dir(target))
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
