package ui

import "strings"

// Kiểu đồng hồ: glyph cho 0–9 và ':', khoảng cách giữa glyph, các ký tự vẽ màu mờ (đoạn tắt).
type clockStyle struct {
	name, label string
	glyphs      [][]string
	gap         int
	dim         string
	classic     bool
	plain       bool
}

// Danh sách kiểu đồng hồ theo thứ tự hiện trên màn chọn; phần tử đầu là mặc định.
func clockStyles() []clockStyle {
	const half, block, rounded, classic, double, shade, dots, lcd, ascii, text = "half", "block", "rounded", "classic", "double", "shade", "dots", "lcd", "ascii", "text"
	const halfLabel, blockLabel, roundedLabel, classicLabel, doubleLabel = "Half block (large)", "Solid block", "Rounded lines", "Classic segments", "Double lines"
	const shadeLabel, dotsLabel, lcdLabel, asciiLabel, textLabel = "Shaded panel", "Braille dots", "LCD segments", "ASCII", "Plain text"
	const narrow, wide, solid, hash, lit, off, unlit, first, joiner = 1, 2, "██", "##", "██", "  ", "░░", 0, ""
	roundedLines, doubleLines := clockData.Lines.Rounded, clockData.Lines.Double
	return []clockStyle{
		{name: half, label: halfLabel, glyphs: glyphSet(halfGlyph), gap: narrow},
		{name: block, label: blockLabel, glyphs: glyphSet(pixelGlyph(solid, off)), gap: wide},
		{name: rounded, label: roundedLabel, glyphs: glyphSet(lineGlyph(roundedLines)), gap: wide},
		{name: classic, label: classicLabel, classic: true},
		{name: double, label: doubleLabel, glyphs: glyphSet(lineGlyph(doubleLines)), gap: wide},
		{name: shade, label: shadeLabel, glyphs: glyphSet(pixelGlyph(lit, unlit)), gap: wide, dim: unlit},
		{name: dots, label: dotsLabel, glyphs: glyphSet(brailleGlyph), gap: narrow},
		{name: lcd, label: lcdLabel, glyphs: glyphSet(lcdGlyph), gap: wide, dim: strings.Join(clockData.Lines.Light[first:], joiner)},
		{name: ascii, label: asciiLabel, glyphs: glyphSet(pixelGlyph(hash, off)), gap: wide},
		{name: text, label: textLabel, plain: true},
	}
}

// Tìm kiểu đồng hồ theo tên.
func findClock(name string) (clockStyle, bool) {
	for _, style := range clockStyles() {
		if style.name == name {
			return style, true
		}
	}
	return clockStyle{}, false
}

// Dựng glyph cho 10 chữ số và dấu ':' (phần tử cuối).
func glyphSet(build func(index int) []string) [][]string {
	const symbols = 11
	glyphs := make([][]string, symbols)
	for index := range glyphs {
		glyphs[index] = build(index)
	}
	return glyphs
}

// Font điểm ảnh 3×5 cho chữ số; phần tử cuối là dấu ':'.
func pixelFont(index int) []string {
	return clockData.Pixel[index]
}

// Mỗi điểm ảnh thành một cặp ký tự sáng hoặc tắt.
func pixelGlyph(on, off string) func(int) []string {
	const pixel = '#'
	return func(index int) []string {
		var rows []string
		for _, row := range pixelFont(index) {
			var line strings.Builder
			for _, cell := range row {
				if cell == pixel {
					line.WriteString(on)
				} else {
					line.WriteString(off)
				}
			}
			rows = append(rows, line.String())
		}
		return rows
	}
}

// Ghép hai hàng điểm ảnh thành một hàng nửa khối, mỗi điểm rộng hai cột: cao 3 dòng.
func halfGlyph(index int) []string {
	const pixel, full, top, bottom, blank, width, first, pair, next = '#', "█", "▀", "▄", " ", 2, 0, 2, 1
	font := pixelFont(index)
	var rows []string
	for row := first; row < len(font); row += pair {
		var line strings.Builder
		for column, cell := range font[row] {
			lower := row+next < len(font) && rune(font[row+next][column]) == pixel
			upper := cell == pixel
			char := blank
			switch {
			case upper && lower:
				char = full
			case upper:
				char = top
			case lower:
				char = bottom
			}
			line.WriteString(strings.Repeat(char, width))
		}
		rows = append(rows, line.String())
	}
	return rows
}

// Phóng điểm ảnh ×2 rồi gói vào ô braille 2×4 chấm: cao 3 dòng.
func brailleGlyph(index int) []string {
	const pixel, scale, cellWidth, cellHeight, base, first = '#', 2, 2, 4, 0x2800, 0
	bits := clockData.Braille
	font := pixelFont(index)
	height, width := len(font)*scale, len(font[first])*scale
	lit := func(y, x int) bool { return y < height && x < width && rune(font[y/scale][x/scale]) == pixel }
	var rows []string
	for y := first; y < height; y += cellHeight {
		var line strings.Builder
		for x := first; x < width; x += cellWidth {
			cell := rune(base)
			for dy := first; dy < cellHeight; dy++ {
				for dx := first; dx < cellWidth; dx++ {
					if lit(y+dy, x+dx) {
						cell |= bits[dy][dx]
					}
				}
			}
			line.WriteRune(cell)
		}
		rows = append(rows, line.String())
	}
	return rows
}

// Các đoạn a–g đang sáng của từng chữ số; phần tử cuối (dấu ':') không có đoạn nào.
func litSegments(index int) string {
	return clockData.Segments[index]
}
