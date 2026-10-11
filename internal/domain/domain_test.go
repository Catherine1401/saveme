package domain

import "testing"

func TestIDNAAndURLNormalization(t *testing.T) {
	const input, want = "https://www.BÜCHER.de:443/a", "xn--bcher-kva.de"
	got, err := (Normalizer{}).Normalize(input)
	if err != nil || got != want {
		t.Fatal(got, err)
	}
}
