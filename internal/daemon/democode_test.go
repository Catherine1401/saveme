package daemon

import (
	"testing"

	"saveme/internal/core"
)

func TestFixedCodeIsUsedForPreparationAndRetries(t *testing.T) {
	const owner, first, duration, code, wrong, invalid, lower, validation = 1, 0, "1h", "K7MQ", "AAAA", "K7M", "k7mq", "fixed code validation"
	f := newFixture(t)
	if f.engine.UseFixedCode(invalid) || f.engine.UseFixedCode(lower) || !f.engine.UseFixedCode(code) {
		t.Fatal(validation)
	}
	preview := f.engine.Handle(core.Request{Cmd: core.CreateRange, Entry: at(first), Duration: duration}, owner)
	if preview.Code != code {
		t.Fatal(preview)
	}
	retry := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: wrong}, owner)
	if retry.Error != core.ErrBadCode || retry.Code != code {
		t.Fatal(retry)
	}
	if response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: code}, owner); !response.OK || !response.Entries[first].Active {
		t.Fatal(response)
	}
}
