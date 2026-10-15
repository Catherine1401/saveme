package ui

import (
	"strings"
	"testing"
	"time"
)

func TestKeycastShowsPressedKeyWithMeaningAndFades(t *testing.T) {
	const space, add, typed, marked, adding, typing, later, width, height = " ", "a", "y", "space  mark", "a  add domain", "typing", 2 * time.Second, 80, 24
	model := readyModel().Appearance(true, true).Keycast(true)
	model, _ = model.Update(Resize{Width: width, Height: height})
	model, _ = model.Update(runeKey(space))
	model.appearance.now = model.castAt
	if view := model.View(); !strings.Contains(view, marked) {
		t.Fatal(view)
	}
	model, _ = model.Update(runeKey(add))
	model.appearance.now = model.castAt
	if !strings.Contains(model.View(), adding) {
		t.Fatal(model.View())
	}
	model, _ = model.Update(runeKey(typed))
	model.appearance.now = model.castAt
	if !strings.Contains(model.View(), typing) {
		t.Fatal(model.View())
	}
	model.appearance.now = model.castAt.Add(later)
	if strings.Contains(model.View(), typing) {
		t.Fatal(model.View())
	}
}

func TestKeycastIsOffByDefault(t *testing.T) {
	const space, marked = " ", "space  mark"
	model := readyModel().Appearance(true, true)
	model, _ = model.Update(runeKey(space))
	if strings.Contains(model.View(), marked) {
		t.Fatal(model.View())
	}
}

func TestKeycastNamesNumberedChoices(t *testing.T) {
	const twelveHours, work, five, two = "5  12 hours", "2  Work hours", "5", "2"
	model := readyModel().Keycast(true)
	model.screen = rangeList
	if got := model.keyMeaning(runeKey(five)); got != twelveHours {
		t.Fatal(got)
	}
	model.screen = presetList
	if got := model.keyMeaning(runeKey(two)); got != work {
		t.Fatal(got)
	}
}
