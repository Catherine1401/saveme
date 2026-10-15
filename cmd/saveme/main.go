package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"saveme/internal/config"
	"saveme/internal/daemon"
	"saveme/internal/domain"
	"saveme/internal/ui"
)

func main() {
	const socketFlag, socketHelp = "socket", "Daemon socket"
	const demoFlag, demoHelp, no, failure = "demo", "Connect to demo without network enforcement", false, 1
	const empty = ""
	const statusFlag, statusHelp = "status", "Print daemon status as JSON"
	const motionFlag, motionHelp, colorFlag, colorHelp, colorEnv = "no-motion", "Disable UI animation", "no-color", "Disable UI colors", "NO_COLOR"
	const themeFlag, themeHelp, clockFlag, clockHelp, effectFlag, effectHelp = "theme", "Color theme for this session; press s in the UI to save one", "clock", "Countdown clock style for this session", "clock-effects", "Comma-separated countdown clock effects for this session, or none"
	still := flag.Bool(motionFlag, no, motionHelp)
	plain := flag.Bool(colorFlag, no, colorHelp)
	socket := flag.String(socketFlag, config.DefaultSocket, socketHelp)
	demo := flag.Bool(demoFlag, no, demoHelp)
	status := flag.Bool(statusFlag, no, statusHelp)
	theme := flag.String(themeFlag, empty, themeHelp)
	clock := flag.String(clockFlag, empty, clockHelp)
	effect := flag.String(effectFlag, empty, effectHelp)
	flag.Parse()
	settings, err := sessionSettings(config.LoadPreferences(), *theme, *clock, *effect)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(failure)
	}
	if *demo {
		*socket = config.Demo().Socket
	}
	client, err := daemon.Dial(*socket)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(failure)
	}
	defer client.Close()
	if *status {
		if err := printStatus(client, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(failure)
		}
		return
	}
	program := tea.NewProgram(terminal{model: ui.New(client, domain.Normalizer{}).Appearance(*plain || os.Getenv(colorEnv) != empty, *still).Configure(settings, saveSettings)}, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(failure)
	}
}
