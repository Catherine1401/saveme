package ui

import (
	"saveme/internal/core"
	"strings"
)

func codePattern(char rune) string {
	const a = "0111010001111111000110001"
	const b = "1111010001111101000111110"
	const c = "0111110000100001000001111"
	const d = "1111010001100011000111110"
	const e = "1111110000111101000011111"
	const f = "1111110000111101000010000"
	const g = "0111110000101111000101111"
	const h = "1000110001111111000110001"
	const j = "0011100010000101001001100"
	const k = "1000110010111001001010001"
	const l = "1000010000100001000011111"
	const m = "1000111011101011000110001"
	const n = "1000111001101011001110001"
	const p = "1111010001111101000010000"
	const q = "0111010001100011001101111"
	const r = "1111010001111101001010001"
	const s = "0111110000011100000111110"
	const t = "1111100100001000010000100"
	const u = "1000110001100011000101110"
	const v = "1000110001100010101000100"
	const w = "1000110001101011101110001"
	const x = "1000101010001000101010001"
	const y = "1000101010001000010000100"
	const z = "1111100010001000100011111"
	const two = "0111010001000100010011111"
	const three = "1111000001011100000111110"
	const four = "1001010010111110001000010"
	const five = "1111110000111100000111110"
	const six = "0111010000111101000101110"
	const seven = "1111100010001000100001000"
	const eight = "0111010001011101000101110"
	const nine = "0111010001011110000101110"
	const empty = ""
	alphabet := core.ConfirmationAlphabet()
	patterns := []string{a, b, c, d, e, f, g, h, j, k, l, m, n, p, q, r, s, t, u, v, w, x, y, z, two, three, four, five, six, seven, eight, nine}
	for index, letter := range alphabet {
		if letter == char {
			return patterns[index]
		}
	}
	return empty
}

func drawCode(code string) string {
	const width, height, zero, on, block, blank, gap, newline = 5, 5, 0, '1', "██", "  ", "   ", "\n"
	var output strings.Builder
	for row := zero; row < height; row++ {
		output.WriteString(gap)
		for _, char := range code {
			pattern := codePattern(char)
			if len(pattern) != width*height {
				continue
			}
			for column := zero; column < width; column++ {
				pixel := blank
				if pattern[row*width+column] == on {
					pixel = block
				}
				output.WriteString(pixel)
			}
			output.WriteString(gap)
		}
		output.WriteString(newline)
	}
	return output.String()
}
