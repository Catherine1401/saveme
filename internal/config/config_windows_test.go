package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWindowsTemplateValidatesWithUserSID(t *testing.T) {
	const template, user, system, socket = "../../deploy/windows/config.json", "S-1-5-21-1-2-3-1001", "S-1-5-18", `C:\saveme.sock`
	const name, mode, empty = "config.json", 0600, ""
	data, err := os.ReadFile(template)
	if err != nil {
		t.Fatal(err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Validate() == nil {
		t.Fatal(cfg)
	}
	cfg.AllowedSID = user
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), name)
	encoded, _ := json.Marshal(cfg)
	if err := os.WriteFile(path, encoded, mode); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err != nil {
		t.Fatal(err)
	}
	for _, broken := range []Config{{AllowedSID: system}, {AllowedSID: user, Socket: socket}} {
		candidate := cfg
		candidate.AllowedSID = broken.AllowedSID
		if broken.Socket != empty {
			candidate.Socket = broken.Socket
		}
		if candidate.Validate() == nil {
			t.Fatal(candidate)
		}
	}
}
