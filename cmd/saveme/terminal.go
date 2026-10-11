package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"saveme/internal/ui"
)

type terminal struct{ model ui.Model }

func (t terminal) Init() tea.Cmd { return command(t.model.Init()) }
func (t terminal) View() string  { return t.model.View() }

func (t terminal) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := message.(tea.WindowSizeMsg); ok {
		message = ui.Resize{Width: size.Width, Height: size.Height}
	}
	if key, ok := message.(tea.KeyMsg); ok {
		kind := ui.KeyControl
		if key.Type == tea.KeyRunes {
			kind = ui.KeyRunes
		}
		message = ui.KeyMsg{Name: key.String(), Type: kind, Runes: key.Runes, Paste: key.Paste}
	}
	model, action := t.model.Update(message)
	t.model = model
	return t, command(action)
}

func command(action ui.Cmd) tea.Cmd {
	if action == nil {
		return nil
	}
	return func() tea.Msg {
		message := action()
		switch value := message.(type) {
		case ui.BatchMsg:
			var actions []tea.Cmd
			for _, child := range value.Commands {
				actions = append(actions, command(child))
			}
			return tea.Batch(actions...)()
		case ui.SequenceMsg:
			var actions []tea.Cmd
			for _, child := range value.Commands {
				actions = append(actions, command(child))
			}
			return tea.Sequence(actions...)()
		case ui.ClearMsg:
			return tea.ClearScreen()
		case ui.QuitMsg:
			return tea.Quit()
		default:
			return value
		}
	}
}
