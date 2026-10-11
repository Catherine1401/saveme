package config

import (
	"testing"
)

func TestConfigurationValidation(t *testing.T) {
	const (
		text12700153 = "127.0.0.1:53"
	)

	const uid, forward, zone, socket, dir, upstream, doh = 1000, "saveme-dns", "Asia/Ho_Chi_Minh", "/run/saveme.sock", "/var/lib/saveme", "1.1.1.1:53", "1.1.1.1"
	user := uint32(uid)
	cfg := Config{AllowedUID: &user, ForwardUser: forward, Zone: zone, Socket: socket, StateDir: dir, Upstreams: []string{upstream}, DoH: []string{doh}}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	cfg.Upstreams = []string{text12700153}
	if err := cfg.Validate(); err == nil {
		const (
			textLoopbackResolverAccepted = "loopback resolver accepted"
		)

		t.Fatal(textLoopbackResolverAccepted)
	}
	cfg.Upstreams = []string{upstream}
	cfg.AllowedUID = nil
	if err := cfg.Validate(); err == nil {
		const (
			textMissingOwnerAccepted = "missing owner accepted"
		)

		t.Fatal(textMissingOwnerAccepted)
	}
}
