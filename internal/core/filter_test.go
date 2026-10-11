package core

import (
	"sync"
	"testing"
)

func TestFilterActivatesAndReleases(t *testing.T) {
	const domain, subdomain, count = "example.com", "a.example.com", 16
	state := EmptyState()
	state.Domains = []string{domain}
	filter := &Filter{}
	filter.Set(state, true)
	if !filter.Blocked(subdomain) {
		const (
			textSubdomainNotBlocked = "subdomain not blocked"
		)

		t.Fatal(textSubdomainNotBlocked)
	}
	var group sync.WaitGroup
	for range count {
		const (
			number1 = 1
		)

		group.Add(number1)
		go func() { defer group.Done(); filter.Blocked(subdomain); filter.Set(state, true) }()
	}
	group.Wait()
	filter.Set(state, false)
	if filter.Blocked(subdomain) {
		const (
			textDomainStillBlockedAfterRelease = "domain still blocked after release"
		)

		t.Fatal(textDomainStillBlockedAfterRelease)
	}
}
