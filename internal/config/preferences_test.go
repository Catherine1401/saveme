package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreferencesSaveLoadLegacyAndCorrupt(t *testing.T) {
	const home, appData, theme, clock, effect, legacy, broken, mode = "XDG_CONFIG_HOME", "AppData", "nord", "block", "slide", `{"theme":"dracula"}`, "{", 0600
	const empty, dirMode, legacyFile, legacyTheme, one, firstIndex = "", 0700, "theme.json", "dracula", 1, 0
	dir := t.TempDir()
	t.Setenv(home, dir)
	t.Setenv(appData, dir)
	if saved := LoadPreferences(); saved.Theme != empty || saved.Effects != nil {
		t.Fatal(saved)
	}
	path, err := PreferencesPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), legacyFile), []byte(legacy), mode); err != nil {
		t.Fatal(err)
	}
	if saved := LoadPreferences(); saved.Theme != legacyTheme {
		t.Fatal(saved)
	}
	off := false
	if err := SavePreferences(Preferences{Theme: theme, Clock: clock, Effects: []string{effect}, Warning: &off}); err != nil {
		t.Fatal(err)
	}
	saved := LoadPreferences()
	if saved.Theme != theme || saved.Clock != clock || len(saved.Effects) != one || saved.Effects[firstIndex] != effect || saved.Warning == nil || *saved.Warning {
		t.Fatal(saved)
	}
	if err := os.WriteFile(path, []byte(broken), mode); err != nil {
		t.Fatal(err)
	}
	if saved := LoadPreferences(); saved.Theme != empty || saved.Effects != nil {
		t.Fatal(saved)
	}
}
