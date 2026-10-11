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
	const socketFlag, socketPath, socketHelp = "socket", "/run/saveme.sock", "Daemon socket"
	const demoFlag, demoHelp, no, failure = "demo", "Connect to demo without network enforcement", false, 1
	const empty = ""
	const statusFlag, statusHelp = "status", "Print daemon status as JSON"
	const motionFlag, motionHelp, colorFlag, colorHelp, colorEnv = "no-motion", "Disable UI animation", "no-color", "Disable UI colors", "NO_COLOR"
	still := flag.Bool(motionFlag, no, motionHelp)
	plain := flag.Bool(colorFlag, no, colorHelp)
	socket := flag.String(socketFlag, socketPath, socketHelp)
	demo := flag.Bool(demoFlag, no, demoHelp)
	status := flag.Bool(statusFlag, no, statusHelp)
	flag.Parse()
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
	program := tea.NewProgram(terminal{model: ui.New(client, domain.Normalizer{}).Appearance(*plain || os.Getenv(colorEnv) != empty, *still)}, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(failure)
	}
}
