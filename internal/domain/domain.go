package domain

import (
	"golang.org/x/net/idna"
	"saveme/internal/core"
)

type Normalizer struct{}

func (Normalizer) Normalize(raw string) (string, error) {
	const empty = ""
	host, err := core.DomainHost(raw)
	if err != nil {
		return empty, err
	}
	ascii, err := idna.Lookup.ToASCII(host)
	if err != nil {
		return empty, core.ErrInvalid
	}
	return core.CanonicalDomain(ascii)
}
