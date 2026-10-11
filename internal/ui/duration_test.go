package ui

import (
	"strings"
	"testing"

	"saveme/internal/core"
)

func TestDurationMenuSubmitsEveryChoice(t *testing.T) {
	const count, enter = 10, "enter"
	const missing = "duration choice missing from menu"
	choices := durationChoices()
	if len(choices) != count {
		t.Fatal(choices)
	}
	for index, choice := range choices {
		t.Run(choice.Value, func(t *testing.T) {
			model := readyModel()
			model.screen = rangeList
			if !strings.Contains(model.View(), choice.Label) {
				t.Fatal(missing, choice)
			}
			model.selected = index
			model, action := model.Update(control(enter))
			if action == nil {
				t.Fatal(choice)
			}
			action()
			client := model.client.(*recordingClient)
			if client.last.Cmd != core.CreateRange || client.last.Duration != choice.Value {
				t.Fatal(client.last)
			}
			if _, err := core.Duration(choice.Value); err != nil {
				t.Fatal(choice.Value, err)
			}
		})
	}
}
