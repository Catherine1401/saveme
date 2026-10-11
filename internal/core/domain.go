package core

import (
	"net"
	"net/url"
	"strings"
	"unicode"
)

func DomainHost(raw string) (string, error) {
	const schemeSeparator, defaultScheme, slash, dot, empty = "://", "https://", "/", ".", ""
	value := strings.Map(func(char rune) rune {
		const removed = -1
		if unicode.IsSpace(char) {
			return removed
		}
		return char
	}, raw)
	if value == empty {
		return empty, ErrInvalid
	}
	if !strings.Contains(value, schemeSeparator) {
		value = defaultScheme + value
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.User != nil || parsed.Host == empty {
		return empty, ErrInvalid
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), dot))
	if strings.ContainsAny(host, slash) || net.ParseIP(host) != nil {
		return empty, ErrInvalid
	}
	return host, nil
}

func CanonicalDomain(host string) (string, error) {
	const prefix, empty = "www.", ""
	value := strings.TrimPrefix(strings.ToLower(host), prefix)
	if err := ValidateDomain(value); err != nil {
		return empty, err
	}
	return value, nil
}

func ValidateDomain(domain string) error {
	const dot, hyphen = ".", "-"
	const zero, minimumLabels, maxLabel, maxDomain, one = 0, 2, 63, 253, 1
	const lowerLetter, upperLetter, lowerDigit, upperDigit = 'a', 'z', '0', '9'
	if len(domain) > maxDomain {
		return ErrInvalid
	}
	labels := strings.Split(domain, dot)
	if len(labels) < minimumLabels {
		return ErrInvalid
	}
	for _, label := range labels {
		if len(label) == zero || len(label) > maxLabel || strings.HasPrefix(label, hyphen) || strings.HasSuffix(label, hyphen) {
			return ErrInvalid
		}
		for _, char := range label {
			letter := char >= lowerLetter && char <= upperLetter
			digit := char >= lowerDigit && char <= upperDigit
			if !letter && !digit && string(char) != hyphen {
				return ErrInvalid
			}
		}
	}
	numeric := true
	for _, char := range labels[len(labels)-one] {
		numeric = numeric && char >= lowerDigit && char <= upperDigit
	}
	if numeric {
		return ErrInvalid
	}
	return nil
}

func DomainBlocked(name string, domains []string) bool {
	const dot, facebook, facebookWWW = ".", "facebook.com", "www.facebook.com"
	name = strings.ToLower(strings.TrimSuffix(name, dot))
	for _, domain := range domains {
		if domain == facebook {
			if name == facebook || name == facebookWWW {
				return true
			}
			continue
		}
		if name == domain || strings.HasSuffix(name, dot+domain) {
			return true
		}
	}
	return false
}
