package daemon

import (
	"testing"
	"time"

	"saveme/internal/core"
)

func TestShortDurationsConfirmPersistAndExpire(t *testing.T) {
	const owner, one, two, four, six = 1, "1h", "2h", "4h", "6h"
	const twoHours, fourHours, sixHours, zero, lastSecond = 2, 4, 6, 0, time.Second
	const target = zero
	cases := []struct {
		value string
		span  time.Duration
	}{{one, time.Hour}, {two, twoHours * time.Hour}, {four, fourHours * time.Hour}, {six, sixHours * time.Hour}}
	for _, scenario := range cases {
		t.Run(scenario.value, func(t *testing.T) {
			f := newFixture(t)
			pending := f.engine.Handle(core.Request{Cmd: core.CreateRange, Entry: at(target), Duration: scenario.value}, owner)
			if !pending.Pending {
				t.Fatal(pending)
			}
			confirmed := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: pending.Code}, owner)
			end := f.clock.now.Add(scenario.span)
			if !confirmed.OK || !confirmed.Active || confirmed.Entries[target].EndAt != end.Unix() || f.repo.state.Entries[target].Lock.EndAt != end.Unix() {
				t.Fatal(confirmed, f.repo.state)
			}
			f.clock.now = end.Add(-lastSecond)
			if err := f.engine.Tick(); err != nil || !f.enforcer.active {
				t.Fatal(err)
			}
			f.clock.now = end
			if err := f.engine.Tick(); err != nil || f.enforcer.active || f.repo.state.Entries[target].Lock != nil {
				t.Fatal(err, f.repo.state)
			}
			if response := f.engine.Handle(core.Request{Cmd: core.Status}, owner); !response.OK || response.Active || response.Entries[target].EndAt != zero {
				t.Fatal(response)
			}
		})
	}
}
