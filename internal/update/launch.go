package update

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"saveme/internal/build"
	"saveme/internal/config"
)

func Launch(paths Layout, executablePath string, args []string) (bool, int) {
	const versionFlag, shortVersion, command, zero, one, failure = "--version", "-version", "update", 0, 1, 1
	installed := filepath.Clean(executablePath) == filepath.Clean(paths.Launcher)
	if len(args) == one && (args[zero] == versionFlag || args[zero] == shortVersion) {
		version := build.Version
		if installed {
			if current, err := paths.ReadVersion(); err == nil {
				version = current
			}
		}
		fmt.Fprintln(os.Stdout, version)
		return true, zero
	}
	if len(args) > zero && args[zero] == command {
		if !installed {
			const literal1 = "Use the installed saveme launcher for updates."
			fmt.Fprintln(os.Stderr, literal1)
			return true, failure
		}
		return true, manual(paths, args[one:])
	}
	if !installed {
		return false, zero
	}
	if _, err := paths.ReadVersion(); err != nil {

		if !skipAutomatic(args) {
			const missing = "Auto updates are not configured; using this binary directly. Re-run the verified installer to enable them."
			fmt.Fprintln(os.Stderr, missing)
		}
		return false, zero
	}
	automatic := !skipAutomatic(args)
	if automatic {
		const literal2 = "open"
		const name = "sessions.lock"
		preferences := config.LoadPreferences()
		enabled := preferences.AutoUpdates == nil || *preferences.AutoUpdates
		if enabled {
			const operation = "status"
			if state, probe, probeErr := Connect(paths, Request{Operation: operation}); probeErr == nil {
				const ready = "ready"
				probe.Close()
				if state.Status.Phase == ready {
					const progress = "Installing the prepared saveme update; commitments are being preserved…"
					fmt.Fprintln(os.Stderr, progress)
				}
			}
		}
		response, lease, err := Connect(paths, Request{Operation: literal2, Automatic: enabled})
		if err != nil {
			const literal4 = "Auto updates unavailable. Re-run the verified installer once to set up the updater service."
			const pending = "transaction"
			if _, check := os.Stat(filepath.Join(paths.Root, pending)); check == nil {
				const literal3 = "Update recovery is pending. Restart the saveme updater service before opening the UI."
				fmt.Fprintln(os.Stderr, literal3)
				return true, failure
			}
			fmt.Fprintln(os.Stderr, literal4)
		} else {
			const literal5 = ""
			defer lease.Close()
			if response.Error != literal5 {
				fmt.Fprintln(os.Stderr, response.Error)
				return true, failure
			}
			if response.Status.LastError != literal5 {
				const literal6 = "Last update:"
				fmt.Fprintln(os.Stderr, literal6, response.Status.LastError)
			}
		}

		unlock, err := Session(filepath.Join(paths.Root, name), false)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return true, failure
		}
		defer unlock()
	}
	version, err := paths.ReadVersion()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, failure
	}
	child := exec.Command(paths.Payload(version), args...)
	child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := child.Run(); err != nil {
		if status, ok := err.(*exec.ExitError); ok {
			return true, status.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)
		return true, failure
	}
	return true, zero
}

func skipAutomatic(args []string) bool {
	for _, argument := range args {
		const equals = "="
		const version = "--version"
		const shortVersion = "-version"
		const help = "-help"
		const helpLong = "--help"
		const helpShort = "-h"
		const status = "-status"
		const statusLong = "--status"
		const demo = "-demo"
		const demoLong = "--demo"
		const socket = "-socket"
		const socketLong = "--socket"
		name, _, _ := strings.Cut(argument, equals)
		switch name {
		case version, shortVersion, help, helpLong, helpShort, status, statusLong, demo, demoLong, socket, socketLong:
			return true
		}
	}
	return false
}

func Observe() Status {
	const operation = "status"
	response, connection, err := Connect(Installed(), Request{Operation: operation})
	if err != nil {
		return Status{}
	}
	connection.Close()
	return response.Status
}

func manual(paths Layout, args []string) int {
	const checking = "checking"
	const attempts = 180
	const literal3 = "Applying any prepared update…"
	const statusFlag, statusShort, zero, one, failure, empty, check, status, open = "--status", "-status", 0, 1, 1, "", "check", "status", "open"
	if len(args) == one && (args[zero] == statusFlag || args[zero] == statusShort) {
		response, connection, err := Connect(paths, Request{Operation: status})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return failure
		}
		defer connection.Close()
		if err := json.NewEncoder(os.Stdout).Encode(response.Status); err != nil {
			return failure
		}
		if response.Error != empty {
			return failure
		}
		return zero
	}
	if len(args) != zero {
		const literal1 = "Usage: saveme update [--status]"
		fmt.Fprintln(os.Stderr, literal1)
		return failure
	}
	response, connection, err := Connect(paths, Request{Operation: check})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return failure
	}
	connection.Close()

	for count := zero; response.Status.Phase == checking && count < attempts; count++ {
		const interval = time.Second
		time.Sleep(interval)
		response, connection, err = Connect(paths, Request{Operation: status})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return failure
		}
		connection.Close()
	}
	if response.Status.Phase == checking {
		const literal2 = "Download still in progress. Run saveme update --status."
		fmt.Fprintln(os.Stderr, literal2)
		return failure
	}
	fmt.Fprintln(os.Stderr, literal3)
	response, connection, err = Connect(paths, Request{Operation: open, Automatic: true})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return failure
	}
	connection.Close()
	json.NewEncoder(os.Stdout).Encode(response.Status)
	if response.Error != empty || response.Status.LastError != empty || response.Status.Deferred != empty {
		return failure
	}
	return zero
}

func Managed(path string) bool {
	paths := Installed()
	version, err := paths.ReadVersion()
	return err == nil && filepath.Clean(path) == filepath.Clean(paths.Payload(version))
}
