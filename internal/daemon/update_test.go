package daemon

import (
	"testing"

	"saveme/internal/core"
)

func TestUpdatePendingRejectsMutationsAndKeepsStatus(t *testing.T) {
	const owner, domain = 1, "other.test"
	f := newFixture(t)
	held := true
	f.engine.UpdatePending = func() bool { return held }
	request := core.Request{Cmd: core.AddDomain, Domain: domain}
	if response := f.engine.Handle(request, owner); response.Error != core.ErrLocked {
		t.Fatal(response)
	}
	if response := f.engine.Handle(core.Request{Cmd: core.Status}, owner); !response.OK {
		t.Fatal(response)
	}
	held = false
	if response := f.engine.Handle(request, owner); !response.OK {
		t.Fatal(response)
	}
}

func TestUpdatePendingRefusesConfirmationWithoutInterruptingEnforcement(t *testing.T) {
	const owner, duration, target = 1, "12h", 0
	f := newFixture(t)
	pending := f.engine.Handle(core.Request{Cmd: core.CreateRange, Duration: duration, Entry: at(target)}, owner)
	if !pending.Pending {
		t.Fatal(pending)
	}
	held := true
	f.engine.UpdatePending = func() bool { return held }
	confirmation := core.Request{Cmd: core.Confirm, Code: pending.Code}
	if response := f.engine.Handle(confirmation, owner); response.Error != core.ErrLocked || f.engine.state.Entries[target].Lock != nil {
		t.Fatal(response, f.repo.state)
	}
	held = false
	if response := f.engine.Handle(confirmation, owner); !response.OK || !response.Active {
		t.Fatal(response)
	}
	held = true
	f.enforcer.active = false
	if err := f.engine.Tick(); err != nil || !f.enforcer.active {
		t.Fatal(err, f.enforcer.active)
	}
	if response := f.engine.Handle(core.Request{Cmd: core.Status}, owner); !response.Active || len(response.Entries[target].Windows) != target {
		t.Fatal(response)
	}
}
