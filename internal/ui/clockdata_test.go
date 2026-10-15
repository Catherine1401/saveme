package ui

import "testing"

func TestEmbeddedClockDataHasExpectedShape(t *testing.T) {
	const symbols, pixelRows, brailleRows, brailleColumns, blank, empty, first = 11, 5, 4, 2, " ", "", 0
	data := loadClockData(clockDataFile)
	if len(data.Pixel) != symbols || len(data.Segments) != symbols || len(data.Braille) != brailleRows {
		t.Fatal(len(data.Pixel), len(data.Segments), len(data.Braille))
	}
	for _, glyph := range data.Pixel {
		if len(glyph) != pixelRows {
			t.Fatal(glyph)
		}
	}
	for _, row := range data.Braille {
		if len(row) != brailleColumns {
			t.Fatal(row)
		}
	}
	for _, set := range []lineSet{data.Lines.Rounded, data.Lines.Double, data.Lines.Light, data.Lines.Heavy} {
		for index, char := range set {
			if char == empty || (index == first) != (char == blank) {
				t.Fatal(index, char)
			}
		}
	}
}

func TestBrokenClockDataStopsLoading(t *testing.T) {
	const broken, expected = "{", "expected panic"
	defer func() {
		if recover() == nil {
			t.Fatal(expected)
		}
	}()
	loadClockData([]byte(broken))
}
