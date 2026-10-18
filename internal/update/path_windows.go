package update

import (
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
)

func Installed() Layout {
	const flags, folder, runtimeFolder = 0, "saveme", "runtime"
	const pipe, launcher, daemon, config, receipt = `\\.\pipe\saveme-updater`, "saveme.exe", "saved.exe", "config.json", "VERSION"
	program, _ := windows.KnownFolderPath(windows.FOLDERID_ProgramFiles, flags)
	data, _ := windows.KnownFolderPath(windows.FOLDERID_ProgramData, flags)
	base := filepath.Join(program, folder)
	return Layout{filepath.Join(base, runtimeFolder), pipe, filepath.Join(base, launcher), filepath.Join(base, daemon), filepath.Join(data, folder, config), filepath.Join(base, receipt)}
}

func executable(name string) string {
	const suffix = ".exe"
	return name + suffix
}

func replace(source, target string) error {
	const attempts = 50
	from, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}

	for range attempts {
		const flags = windows.MOVEFILE_REPLACE_EXISTING | windows.MOVEFILE_WRITE_THROUGH
		const delay = 100 * time.Millisecond
		err = windows.MoveFileEx(from, to, flags)
		if err == nil {
			return nil
		}
		if err != windows.ERROR_SHARING_VIOLATION && err != windows.ERROR_ACCESS_DENIED {
			return err
		}
		time.Sleep(delay)
	}
	return err
}
