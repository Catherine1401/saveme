package daemon

import (
	"errors"
	"sync"
	"testing"
	"time"

	"saveme/internal/core"
)

type fakeClock struct {
	now     time.Time
	trusted bool
}

func (c *fakeClock) Now() time.Time { return c.now }
func (c *fakeClock) Trusted() bool  { return c.trusted }

type memoryStore struct {
	state core.State
	fail  bool
}

func (r *memoryStore) Save(state core.State) error {
	const reason = "disk failure"
	if r.fail {
		return errors.New(reason)
	}
	r.state = state.Clone()
	return nil
}

type fakeEnforcer struct {
	active bool
	fail   bool
}

func (f *fakeEnforcer) Apply(_ core.State, active bool) error {
	const reason = "enforcement failure"
	if f.fail {
		return errors.New(reason)
	}
	f.active = active
	return nil
}

type asciiNormalizer struct{}

func (asciiNormalizer) Normalize(raw string) (string, error) {
	const empty = ""
	host, err := core.DomainHost(raw)
	if err != nil {
		return empty, err
	}
	return core.CanonicalDomain(host)
}

type fixture struct {
	engine   *Engine
	clock    *fakeClock
	repo     *memoryStore
	enforcer *fakeEnforcer
}

func newFixture(t *testing.T) fixture {

	const nowText, zone, domain = "2026-10-09T12:00:00Z", "UTC", "example.com"
	t.Helper()

	now, err := time.Parse(time.RFC3339, nowText)
	if err != nil {
		t.Fatal(err)
	}
	clock := &fakeClock{now: now, trusted: true}
	repo, enforce := &memoryStore{}, &fakeEnforcer{}
	state := core.EmptyState()
	state.Domains = []string{domain}
	engine, err := New(state, clock, repo, enforce, asciiNormalizer{}, zone)
	if err != nil {
		t.Fatal(err)
	}
	return fixture{engine: engine, clock: clock, repo: repo, enforcer: enforce}
}

func createRange(t *testing.T, f fixture) core.Response {

	const owner, duration = 1, "12h"
	t.Helper()

	response := f.engine.Handle(core.Request{Cmd: core.CreateRange, Duration: duration}, owner)
	if !response.Pending {
		t.Fatal(response)
	}
	response = f.engine.Handle(core.Request{Cmd: core.Confirm, Code: response.Code}, owner)
	if !response.OK || !response.Active {
		t.Fatal(response)
	}
	return response
}

func TestEveryMutationDeniedAndStatusRedacted(t *testing.T) {
	const owner = 1
	f := newFixture(t)
	createRange(t, f)
	f.enforcer.fail = true
	commands := []core.Command{core.CreateDaily, core.AddWindow, core.DeleteWindow, core.CreateRange, core.Confirm, core.Cancel, core.AddDomain, core.EditDomain, core.DeleteDomain}
	for _, command := range commands {
		response := f.engine.Handle(core.Request{Cmd: command}, owner)
		if response.Error != core.ErrLocked {
			t.Fatal(command, response)
		}
	}
	status := f.engine.Handle(core.Request{Cmd: core.Status}, owner)
	if status.State != nil || status.Preview != nil || !status.Active {
		t.Fatal(status)
	}
}

func TestPendingExpiresRotatesAndIsOwned(t *testing.T) {
	const owner, otherOwner, duration, wrong, ttl, attempts = 1, 2, "12h", "wrong", 60 * time.Second, 3
	f := newFixture(t)
	request := core.Request{Cmd: core.CreateRange, Duration: duration}
	initial := f.engine.Handle(request, owner)
	foreign := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: initial.Code}, otherOwner)
	if foreign.Error != core.ErrBadCode {
		t.Fatal(foreign)
	}
	for attempt := owner; attempt <= attempts; attempt++ {
		response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: wrong}, owner)
		if response.Error != core.ErrBadCode {
			t.Fatal(response)
		}
		if response.Pending != (attempt < attempts) {
			t.Fatal(response)
		}
		if response.Pending && response.Code == initial.Code {
			const (
				textCodeDidNotRotate = "code did not rotate"
			)

			t.Fatal(textCodeDidNotRotate)
		}
		initial = response
	}
	expiring := f.engine.Handle(request, owner)
	f.clock.now = f.clock.now.Add(ttl)
	response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: expiring.Code}, owner)
	if response.Error != core.ErrCodeExpired {
		t.Fatal(response)
	}
}

func TestPendingReplacementCancellationAndDomainChange(t *testing.T) {
	const (
		number24 = 24
	)

	const owner, firstDuration, secondDuration, domain = 1, "12h", "1d", "youtube.com"
	f := newFixture(t)
	f.engine.Handle(core.Request{Cmd: core.CreateRange, Duration: firstDuration}, owner)
	second := f.engine.Handle(core.Request{Cmd: core.CreateRange, Duration: secondDuration}, owner)
	if f.engine.pending.Candidate.Lock.EndAt-f.engine.pending.Candidate.Lock.StartAt != int64(number24*time.Hour/time.Second) {
		t.Fatal(second)
	}
	f.engine.Handle(core.Request{Cmd: core.Cancel}, owner)
	if response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: second.Code}, owner); response.Error != core.ErrBadCode {
		t.Fatal(response)
	}
	third := f.engine.Handle(core.Request{Cmd: core.CreateRange, Duration: firstDuration}, owner)
	f.engine.Handle(core.Request{Cmd: core.AddDomain, Domain: domain}, owner)
	if response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: third.Code}, owner); response.Error != core.ErrBadCode {
		t.Fatal(response)
	}
}

func TestConfirmationUsesCurrentTimeAndRechecksDaily(t *testing.T) {
	const (
		textUTC  = "UTC"
		number12 = 12
		number60 = 60
		number13 = 13
		text1400 = "14:00"
		text1500 = "15:00"
	)

	const owner, duration, wait, span = 1, "12h", 30 * time.Second, 12 * time.Hour
	f := newFixture(t)
	preview := f.engine.Handle(core.Request{Cmd: core.CreateRange, Duration: duration}, owner)
	f.clock.now = f.clock.now.Add(wait)
	response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner)
	if response.EndAt != f.clock.now.Add(span).Unix() {
		t.Fatal(response)
	}
	f = newFixture(t)
	f.engine.state.Lock = &core.Lock{Kind: core.Daily, Zone: textUTC, Windows: []core.Window{{Start: number12*number60 + owner, End: number13 * number60}}}
	preview = f.engine.Handle(core.Request{Cmd: core.AddWindow, Start: text1400, End: text1500}, owner)
	if !preview.Pending {
		t.Fatal(preview)
	}
	f.clock.now = f.clock.now.Add(time.Minute)
	response = f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner)
	if response.Error != core.ErrLocked {
		t.Fatal(response)
	}
}

func TestDailyStartsImmediatelyAndFinalDomainIsProtected(t *testing.T) {
	const (
		text1200 = "12:00"
		text1300 = "13:00"
	)

	const owner, index, nextDay = 1, 0, 24 * time.Hour
	f := newFixture(t)
	preview := f.engine.Handle(core.Request{Cmd: core.CreateDaily, Windows: []core.InputWindow{{Start: text1200, End: text1300}}}, owner)
	response := f.engine.Handle(core.Request{Cmd: core.Confirm, Code: preview.Code}, owner)
	if !response.Active {
		t.Fatal(response)
	}
	f.clock.now = f.clock.now.Add(nextDay - time.Hour)
	response = f.engine.Handle(core.Request{Cmd: core.DeleteDomain, Index: func() *int { value := index; return &value }()}, owner)
	if response.Error != core.ErrInvalid {
		t.Fatal(response)
	}
	value := index
	response = f.engine.Handle(core.Request{Cmd: core.DeleteWindow, Index: &value}, owner)
	if response.Error != core.ErrInvalid {
		const (
			textLastWindowNeedsConfirmation = "last window needs confirmation"
		)

		t.Fatal(textLastWindowNeedsConfirmation, response)
	}
	response = f.engine.Handle(core.Request{Cmd: core.DeleteWindow, Index: &value, Confirmed: true}, owner)
	if !response.OK || response.State.Lock != nil {
		t.Fatal(response)
	}
}

func TestExpiryRestartAndClockTrust(t *testing.T) {
	const (
		textUTC = "UTC"
	)

	const owner, span, domain = 1, 12 * time.Hour, "youtube.com"
	f := newFixture(t)
	createRange(t, f)
	restarted, err := New(f.repo.state, f.clock, f.repo, f.enforcer, asciiNormalizer{}, textUTC)
	if err != nil {
		t.Fatal(err)
	}
	f.clock.now = f.clock.now.Add(span)
	response := restarted.Handle(core.Request{Cmd: core.Status}, owner)
	if response.Active || response.State.Lock != nil || f.enforcer.active {
		t.Fatal(response)
	}
	f.clock.trusted = false
	response = restarted.Handle(core.Request{Cmd: core.AddDomain, Domain: domain}, owner)
	if response.Error != core.ErrClockUnsync {
		t.Fatal(response)
	}
}

func TestConcurrentChangesAndStorageFailure(t *testing.T) {
	const (
		number2       = 2
		textRedditCom = "reddit.com"
	)

	const owner, domain, count = 1, "youtube.com", 8
	f := newFixture(t)
	var group sync.WaitGroup
	for range count {
		group.Add(owner)
		go func() { defer group.Done(); f.engine.Handle(core.Request{Cmd: core.AddDomain, Domain: domain}, owner) }()
	}
	group.Wait()
	if len(f.engine.state.Domains) != number2 {
		t.Fatal(f.engine.state.Domains)
	}
	f.repo.fail = true
	response := f.engine.Handle(core.Request{Cmd: core.AddDomain, Domain: textRedditCom}, owner)
	if response.Error != core.ErrStorage {
		t.Fatal(response)
	}
	f.repo.fail = false
	response = f.engine.Handle(core.Request{Cmd: core.DeleteDomain}, owner)
	if response.Error != core.ErrStorage {
		const (
			textStorageFaultMustStayLatched = "storage fault must stay latched"
		)

		t.Fatal(textStorageFaultMustStayLatched, response)
	}
}

func TestEnforcementFailureRetainsDurableLock(t *testing.T) {
	const owner, duration = 1, "12h"
	f := newFixture(t)
	preview := f.engine.Handle(core.Request{Cmd: core.CreateRange, Duration: duration}, owner)
	f.enforcer.fail = true
	response := f.engine.confirm(preview.Code, owner, f.clock.now)
	if response.Error != core.ErrEnforcement || f.repo.state.Lock == nil || f.engine.state.Lock == nil {
		t.Fatal(response)
	}
	f.enforcer.fail = false
	if err := f.engine.Tick(); err != nil || !f.enforcer.active {
		t.Fatal(err)
	}
}

func TestUntrustedClockKeepsExistingScheduleClosed(t *testing.T) {
	const owner, zone, start, end = 1, "UTC", 22 * 60, 6 * 60
	f := newFixture(t)
	f.engine.state.Lock = &core.Lock{Kind: core.Daily, Zone: zone, Windows: []core.Window{{Start: start, End: end}}}
	f.clock.trusted = false
	response := f.engine.Handle(core.Request{Cmd: core.Status}, owner)
	if !response.Active || response.State != nil || response.Error != core.ErrClockUnsync || !f.enforcer.active {
		t.Fatal(response)
	}
}
