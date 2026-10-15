package ui

import (
	"strings"
	"time"
)

// Loại ô trên lưới đồng hồ: nét số, nét tắt, viền neon, sao lấp lánh, mưa, ô trống.
type cellKind int

const (
	strokeCell cellKind = iota
	dimCell
	neonCell
	sparkleCell
	rainCell
	blankCell
)

// Một ô của lưới đồng hồ.
type clockCell struct {
	char rune
	kind cellKind
}

// Dựng lưới ô từ các dòng đồng hồ; khi bật neon, sparkle hoặc rain thì nới thêm 1 ô mỗi phía.
func (m Model) clockGrid(lines []string, dim string) ([][]clockCell, int) {
	const neon, sparkle, rain, blank, none, sides = "neon", "sparkle", "rain", ' ', 0, 2
	pad := m.decorPad()
	width := none
	for _, line := range lines {
		width = max(width, len([]rune(line)))
	}
	grid := make([][]clockCell, len(lines)+pad*sides)
	for row := range grid {
		grid[row] = make([]clockCell, width+pad*sides)
		for column := range grid[row] {
			grid[row][column] = clockCell{char: blank, kind: blankCell}
		}
	}
	for row, line := range lines {
		for column, char := range []rune(line) {
			kind := strokeCell
			switch {
			case char == blank:
				kind = blankCell
			case strings.ContainsRune(dim, char):
				kind = dimCell
			}
			grid[row+pad][column+pad] = clockCell{char: char, kind: kind}
		}
	}
	now := m.frameTime()
	if m.effectOn(neon) {
		addNeon(grid)
	}
	if m.moving(rain) {
		addRain(grid, now)
	}
	if m.moving(sparkle) {
		addSparkles(grid, now)
	}
	return grid, pad
}

// Số ô nới thêm mỗi phía khi bật hiệu ứng trang trí quanh đồng hồ.
func (m Model) decorPad() int {
	const neon, sparkle, rain, none, border = "neon", "sparkle", "rain", 0, 1
	if m.effectOn(neon) || m.moving(sparkle) || m.moving(rain) {
		return border
	}
	return none
}

// Viền neon: ô trống kề cạnh nét số vẽ ký tự mờ.
func addNeon(grid [][]clockCell) {
	const glow, back, forward, still, rowAxis, columnAxis, edge, axes = '░', -1, 1, 0, 0, 1, 0, 2
	steps := [][axes]int{{back, still}, {forward, still}, {still, back}, {still, forward}}
	for row := range grid {
		for column := range grid[row] {
			if grid[row][column].kind != blankCell {
				continue
			}
			for _, step := range steps {
				y, x := row+step[rowAxis], column+step[columnAxis]
				if y >= edge && y < len(grid) && x >= edge && x < len(grid[y]) && grid[y][x].kind == strokeCell {
					grid[row][column] = clockCell{char: glow, kind: neonCell}
					break
				}
			}
		}
	}
}

// Mưa braille: mỗi cột có một giọt trôi xuống mỗi 200 ms, chỉ hiện ở ô trống.
func addRain(grid [][]clockCell, now time.Time) {
	const step, every, gapRows, first, none, dropChars = 200, 3, 4, 0, 0, "⠂⠄⡀"
	drops := []rune(dropChars)
	height := len(grid)
	for column := range grid[first] {
		hash := mixHash(int64(column), column)
		if hash%every != none {
			continue
		}
		row := int((uint64(now.UnixMilli()/step) + hash) % uint64(height+gapRows))
		if row < height && grid[row][column].kind == blankCell {
			grid[row][column] = clockCell{char: drops[int(hash%uint64(len(drops)))], kind: rainCell}
		}
	}
}

// Sao lấp lánh: nửa đầu mỗi giây có tối đa 2 sao ở viền, vị trí theo giây.
func addSparkles(grid [][]clockCell, now time.Time) {
	const half, stars, second, edge, last, none, rowAxis, columnAxis, axes, starChars = 500, 2, 1000, 0, 1, 0, 0, 1, 2, "✦✧"
	shapes := []rune(starChars)
	if now.UnixMilli()%second >= half {
		return
	}
	var border [][axes]int
	for row := range grid {
		for column := range grid[row] {
			if row == edge || row == len(grid)-last || column == edge || column == len(grid[row])-last {
				border = append(border, [axes]int{row, column})
			}
		}
	}
	for index := none; index < stars && len(border) > none; index++ {
		hash := mixHash(now.Unix(), index)
		spot := border[hash%uint64(len(border))]
		if cell := &grid[spot[rowAxis]][spot[columnAxis]]; cell.kind == blankCell {
			*cell = clockCell{char: shapes[int(hash%uint64(len(shapes)))], kind: sparkleCell}
		}
	}
}
