package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"saveme/internal/core"
)

type localNormalizer struct{}

func (localNormalizer) Normalize(raw string) (string, error) {
	const empty = ""
	host, err := core.DomainHost(raw)
	if err != nil {
		return empty, err
	}
	return core.CanonicalDomain(host)
}

type recordingClient struct {
	last     core.Request
	response core.Response
}

func (c *recordingClient) Call(request core.Request) (core.Response, error) {
	c.last = request
	return c.response, nil
}

func readyModel() Model {
	const domain, now = "example.com", 1000
	state := core.EmptyState()
	state.Domains = []string{domain}
	model := New(&recordingClient{}, localNormalizer{})
	model.status, model.ready = core.Response{OK: true, State: &state, Now: now}, true
	return model
}

func control(name string) KeyMsg { return KeyMsg{Name: name, Type: KeyControl} }
func runeKey(text string) KeyMsg { return KeyMsg{Type: KeyRunes, Runes: []rune(text)} }

func TestLockedViewHidesEveryMenu(t *testing.T) {
	const (
		number1000 = 1000
		number1060 = 1060
		textA      = "a"
	)

	const protected, menu, domains = "example.com", "Edit domains", "Daily schedule"
	model := readyModel()
	model.screen = domainList
	model, _ = model.Update(reply{request: core.Request{Cmd: core.Status}, response: core.Response{OK: true, Active: true, Now: number1000, EndAt: number1060}})
	view := model.View()
	if strings.Contains(view, protected) || strings.Contains(view, menu) || strings.Contains(view, domains) {
		t.Fatal(view)
	}
	model, action := model.Update(runeKey(textA))
	if action != nil || !model.status.Active {
		const (
			textLockedKeyMustDoNothing = "locked key must do nothing"
		)

		t.Fatal(textLockedKeyMustDoNothing)
	}
}

func TestConfirmationRejectsPasteAndRapidCharacters(t *testing.T) {
	const (
		textK   = "K"
		text7   = "7"
		number2 = 2
		textM   = "M"
	)

	const code, now, ttl, pause, zero = "K7MQ", 1000, 60, 35 * time.Millisecond, 0
	model := readyModel()
	model.screen, model.pending = confirmation, core.Response{Code: code, Now: now, ExpiresIn: ttl}
	paste := runeKey(code)
	paste.Paste = true
	model, action := model.Update(paste)
	if action != nil || len(model.codeInput) != zero {
		const (
			textPasteAccepted = "paste accepted"
		)

		t.Fatal(textPasteAccepted)
	}
	model, action = model.Update(runeKey(textK))
	if action == nil {
		const (
			textExpectedDeferredCharacter = "expected deferred character"
		)

		t.Fatal(textExpectedDeferredCharacter)
	}
	deferred := action()
	model.lastChar = time.Now()
	model, _ = model.Update(runeKey(text7))
	model, _ = model.Update(deferred)
	if len(model.codeInput) != zero {
		const (
			textBurstAccepted = "burst accepted"
		)

		t.Fatal(textBurstAccepted)
	}
	model.lastChar = time.Now().Add(-number2 * pause)
	model, action = model.Update(runeKey(textM))
	model, _ = model.Update(action())
	if model.codeInput != textM {
		const (
			textNormalTypingRejected = "normal typing rejected"
		)

		t.Fatal(textNormalTypingRejected)
	}
}

func TestCodeDrawingAndCancellation(t *testing.T) {
	const (
		textEsc   = "esc"
		textEmpty = ""
	)

	const code, border, alphabet = "K7MQ", "┏━━━┓", "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for _, char := range alphabet {
		const format = "┃ %s ┃"
		if !strings.Contains(drawCode(string(char)), fmt.Sprintf(format, string(char))) {
			t.Fatal(char)
		}
	}
	drawing := drawCode(code)
	if strings.Contains(drawing, code) || !strings.Contains(drawing, border) {
		t.Fatal(drawing)
	}
	model := readyModel()
	model.screen, model.previous, model.pending = confirmation, rangeList, core.Response{Code: code}
	model, _ = model.Update(control(textEsc))
	if model.pending.Code != textEmpty || model.screen != rangeList {
		const (
			textCodeNotClearedOnCancel = "code not cleared on cancel"
		)

		t.Fatal(textCodeNotClearedOnCancel)
	}
}

func TestTimeEntryBoundsAndNavigation(t *testing.T) {
	const (
		text2   = "2"
		text3   = "3"
		number1 = 1
		text5   = "5"
		text9   = "9"
		number2 = 2
		textUp  = "up"
	)

	const zero, hour, minute, invalidDigit = 0, 23, 59, "4"
	model := readyModel()
	model.beginWindow()
	model, _ = model.Update(runeKey(text2))
	model, _ = model.Update(runeKey(invalidDigit))
	if model.field != zero {
		const (
			textInvalidHourAdvanced = "invalid hour advanced"
		)

		t.Fatal(textInvalidHourAdvanced)
	}
	model, _ = model.Update(runeKey(text3))
	if model.fields[zero] != hour || model.field != number1 {
		t.Fatal(model.fields, model.field)
	}
	model, _ = model.Update(runeKey(text5))
	model, _ = model.Update(runeKey(text9))
	if model.fields[number1] != minute || model.field != number2 {
		t.Fatal(model.fields, model.field)
	}
	model.field = zero
	model, _ = model.Update(control(textUp))
	if model.fields[zero] != zero {
		const (
			textHourDidNotWrap = "hour did not wrap"
		)

		t.Fatal(textHourDidNotWrap)
	}
}

func TestWindowsOverlapAndLastWindowConfirmation(t *testing.T) {
	const (
		number1320 = 1320
		number360  = 360
		textD      = "d"
		number5    = 5
		number0    = 0
		number7    = 7
		textEnter  = "enter"
	)

	const zone = "UTC"
	model := readyModel()
	model.status.State.Lock = &core.Lock{Kind: core.Daily, Zone: zone, Windows: []core.Window{{Start: number1320, End: number360}}}
	model.screen = windowList
	model, action := model.Update(runeKey(textD))
	if model.screen != lastWindow || action != nil {
		const (
			textLastWindowDeletedWithoutConfirmation = "last window deleted without confirmation"
		)

		t.Fatal(textLastWindowDeletedWithoutConfirmation)
	}
	model.beginWindow()
	model.fields = [timeFieldCount]int{number5, number0, number7, number0}
	model, action = model.Update(control(textEnter))
	if action != nil || model.error != string(core.ErrOverlap) {
		const (
			textOverlapAccepted = "overlap accepted"
		)

		t.Fatal(textOverlapAccepted, model.error)
	}
}

func TestCountdownDomainValidationAndEmptyViews(t *testing.T) {
	const (
		number90061             = 90061
		text1d010101            = "1d 01:01:01"
		number59                = 59
		text000059              = "00:00:59"
		number1                 = 1
		text000000              = "00:00:00"
		textHttpsWwwEXAMPLEComA = "https://www.EXAMPLE.com/a"
		textSLUExampleCom       = "Will save: example.com"
		textExample123          = "example.123"
		textEnter               = "enter"
		textChACDomain          = "No domains"
	)

	cases := []struct {
		seconds int64
		want    string
	}{{seconds: number90061, want: text1d010101}, {seconds: number59, want: text000059}, {seconds: -number1, want: text000000}}
	for _, tc := range cases {
		if got := countdown(tc.seconds); got != tc.want {
			t.Fatal(got, tc.want)
		}
	}
	model := readyModel()
	model.screen, model.domainText = domainEdit, textHttpsWwwEXAMPLEComA
	if !strings.Contains(model.View(), textSLUExampleCom) {
		t.Fatal(model.View())
	}
	model.domainText = textExample123
	_, action := model.Update(control(textEnter))
	if action != nil {
		const (
			textInvalidDomainSubmitted = "invalid domain submitted"
		)

		t.Fatal(textInvalidDomainSubmitted)
	}
	model.status.State.Domains = nil
	model.screen = domainList
	if !strings.Contains(model.View(), textChACDomain) {
		t.Fatal(model.View())
	}
}
