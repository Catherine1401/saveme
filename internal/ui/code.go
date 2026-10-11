package ui

import "strings"

func drawCode(code string) string {
	const top, bottom, space, gap, newline = "┏━━━┓", "┗━━━┛", " ", "  ", "\n"
	const zero, rows = 0, 3
	var output strings.Builder
	for row := zero; row < rows; row++ {
		for index, char := range code {
			if index > zero {
				output.WriteString(gap)
			}
			switch row {
			case zero:
				output.WriteString(top)
			case rows - 1:
				output.WriteString(bottom)
			default:
				const edge = "┃"
				output.WriteString(edge + space + string(char) + space + edge)
			}
		}
		output.WriteString(newline)
	}
	return output.String()
}
