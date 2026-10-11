package core

import (
	"strings"
	"testing"
)

func TestDomainNormalization(t *testing.T) {
	const (
		textHttpsWwwExampleCOM443AB1 = " https://www.Example.COM:443/a?b=1 "
		textExampleCom               = "example.com"
		textWWWYOUTUBECOM            = "WWW.YOUTUBE.COM."
		textYoutubeCom               = "youtube.com"
		textXnBcherKvaDe             = "xn--bcher-kva.de"
		spaced                       = " EXAM PLE.COM\t"
	)

	cases := []struct {
		raw  string
		want string
	}{
		{raw: textHttpsWwwExampleCOM443AB1, want: textExampleCom},
		{raw: textWWWYOUTUBECOM, want: textYoutubeCom},
		{raw: textXnBcherKvaDe, want: textXnBcherKvaDe},
		{raw: spaced, want: textExampleCom},
	}
	for _, tc := range cases {
		host, err := DomainHost(tc.raw)
		if err != nil {
			t.Fatal(err)
		}
		got, err := CanonicalDomain(host)
		if err != nil || got != tc.want {
			const (
				textSSV = "%s: %s %v"
			)

			t.Fatalf(textSSV, tc.raw, got, err)
		}
	}
}

func TestInvalidDomains(t *testing.T) {
	const (
		textLocalhost           = "localhost"
		textExample123          = "example.123"
		textACom                = "-a.com"
		textAComValue           = "a-.com"
		textAComValueValue      = "a..com"
		textABCom               = "a_b.com"
		textHttpsUserExampleCom = "https://user@example.com"
		text127001              = "127.0.0.1"
		text1                   = "[::1]"
		textHttpsA              = "https:///a"
		textExampleComBad       = "example.com:bad"
	)

	const oversized, letter, suffix = 64, "a", ".com"
	invalid := []string{textLocalhost, textExample123, textACom, textAComValue, textAComValueValue, textABCom, strings.Repeat(letter, oversized) + suffix}
	for _, name := range invalid {
		if ValidateDomain(name) == nil {
			t.Fatal(name)
		}
	}
	hosts := []string{textHttpsUserExampleCom, text127001, text1, textHttpsA, textExampleComBad}
	for _, name := range hosts {
		if _, err := DomainHost(name); err == nil {
			t.Fatal(name)
		}
	}
}

func TestDomainBoundary(t *testing.T) {
	const (
		textEXAMPLECOM         = "EXAMPLE.COM."
		textABExampleCom       = "a.b.example.com"
		textNotexampleCom      = "notexample.com"
		textExampleComEvilTest = "example.com.evil.test"
	)

	const blocked = "example.com"
	cases := []struct {
		name    string
		blocked bool
	}{
		{name: textEXAMPLECOM, blocked: true}, {name: textABExampleCom, blocked: true},
		{name: textNotexampleCom}, {name: textExampleComEvilTest},
	}
	for _, tc := range cases {
		if DomainBlocked(tc.name, []string{blocked}) != tc.blocked {
			t.Fatal(tc.name)
		}
	}
}

func TestFacebookDomainBoundary(t *testing.T) {
	const facebook, www, mobile, nested, deceptive, suffix, messenger = "facebook.com", "WWW.FACEBOOK.COM.", "m.facebook.com", "a.b.facebook.com", "notfacebook.com", "facebook.com.evil.test", "messenger.com"
	cases := []struct {
		name    string
		blocked bool
	}{
		{name: facebook, blocked: true}, {name: www, blocked: true},
		{name: mobile}, {name: nested}, {name: deceptive}, {name: suffix}, {name: messenger},
	}
	for _, tc := range cases {
		if DomainBlocked(tc.name, []string{facebook}) != tc.blocked {
			t.Fatal(tc.name)
		}
	}
	if !DomainBlocked(mobile, []string{facebook, mobile}) {
		t.Fatal(mobile)
	}
	if DomainBlocked(facebook, nil) {
		t.Fatal(facebook)
	}
}
