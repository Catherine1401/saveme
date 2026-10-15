package ui

import "strings"

// Ký tự khung theo hướng nối của ô: bit 1 lên, 2 xuống, 4 trái, 8 phải.
type lineSet [lineShapes]string

// Số kiểu nối của một ô: 4 bit hướng cho 16 tổ hợp.
const lineShapes = 16

// Kích thước lưới vẽ chữ số bảy đoạn.
const gridRows, gridColumns = 5, 4

// Hướng nối của từng ô cho các đoạn a–g: bit 1 lên, 2 xuống, 4 trái, 8 phải.
func segmentGrid(on string) [gridRows][gridColumns]int {
	const up, down, left, right, top, middle, bottom, first, last = 1, 2, 4, 8, 0, 2, 4, 0, 3
	const segA, segB, segC, segD, segE, segF, segG = 'a', 'b', 'c', 'd', 'e', 'f', 'g'
	var grid [gridRows][gridColumns]int
	horizontal := func(row int) {
		for column := first; column <= last; column++ {
			if column > first {
				grid[row][column] |= left
			}
			if column < last {
				grid[row][column] |= right
			}
		}
	}
	vertical := func(column, from, to int) {
		for row := from; row <= to; row++ {
			if row > from {
				grid[row][column] |= up
			}
			if row < to {
				grid[row][column] |= down
			}
		}
	}
	for _, segment := range on {
		switch segment {
		case segA:
			horizontal(top)
		case segG:
			horizontal(middle)
		case segD:
			horizontal(bottom)
		case segF:
			vertical(first, top, middle)
		case segB:
			vertical(last, top, middle)
		case segE:
			vertical(first, middle, bottom)
		case segC:
			vertical(last, middle, bottom)
		}
	}
	return grid
}

// Dấu ':' cho kiểu nét khung.
func lineColon(dot string) []string {
	const blank = " "
	return []string{blank, dot, blank, dot, blank}
}

// Vẽ chữ số bảy đoạn bằng nét khung nối liền ở góc.
func lineGlyph(set lineSet) func(int) []string {
	const colon, dot = 10, "•"
	return func(index int) []string {
		if index == colon {
			return lineColon(dot)
		}
		grid := segmentGrid(litSegments(index))
		lines := make([]string, gridRows)
		for row := range grid {
			var line strings.Builder
			for _, mask := range grid[row] {
				line.WriteString(set[mask])
			}
			lines[row] = line.String()
		}
		return lines
	}
}

// Kiểu LCD: đoạn sáng nét đậm, đoạn tắt của khung số 8 vẽ nét mảnh để tô màu mờ.
func lcdGlyph(index int) []string {
	const colon, dot, all, space, none = 10, "▪", "abcdefg", " ", 0
	heavyLines, lightLines := clockData.Lines.Heavy, clockData.Lines.Light
	if index == colon {
		return lineColon(dot)
	}
	lit, frame := segmentGrid(litSegments(index)), segmentGrid(all)
	lines := make([]string, gridRows)
	for row := range frame {
		var line strings.Builder
		for column, mask := range frame[row] {
			switch {
			case lit[row][column] != none:
				line.WriteString(heavyLines[lit[row][column]])
			case mask != none:
				line.WriteString(lightLines[mask])
			default:
				line.WriteString(space)
			}
		}
		lines[row] = line.String()
	}
	return lines
}
