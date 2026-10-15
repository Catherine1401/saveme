package ui

import (
	"strings"
	"time"
)

// Băm thuần từ hạt giống và chỉ số, thay cho nguồn ngẫu nhiên để test xác định được.
func mixHash(seed int64, index int) uint64 {
	const prime, step, shift = 2654435761, 40503, 13
	value := uint64(seed)*prime + uint64(index)*step
	return value ^ value>>shift
}

// Lật kiểu split-flap: nửa trên số mới, nửa dưới số cũ, dòng giữa là bản lề.
func flipGlyph(previous, next []string) []string {
	const hinge, blank, half = '▁', ' ', 2
	height := len(next)
	middle := height / half
	flipped := make([]string, height)
	for row := range flipped {
		switch {
		case row < middle:
			flipped[row] = next[row]
		case row > middle:
			flipped[row] = previous[row]
		default:
			flipped[row] = strings.Map(func(char rune) rune {
				if char == blank {
					return char
				}
				return hinge
			}, next[row])
		}
	}
	return flipped
}

// Nhiễu glitch: mỗi 3 giây có 150 ms thay 1–2 ô có nét bằng ký tự nhiễu.
func glitchLines(lines []string, now time.Time) []string {
	const period, window, cells, blank, second, spread, first = 3000, 150, 2, ' ', 1000, 7, 0
	noise := []rune(glitchNoise())
	if now.UnixMilli()%period >= window || len(lines) == first {
		return lines
	}
	grid := make([][]rune, len(lines))
	for row, line := range lines {
		grid[row] = []rune(line)
	}
	width := len(grid[first])
	for index := first; index < cells; index++ {
		hash := mixHash(now.Unix()/(period/second), index)
		row, column := int(hash%uint64(len(grid))), int(hash/spread%uint64(width))
		if column < len(grid[row]) && grid[row][column] != blank {
			grid[row][column] = noise[int(hash%uint64(len(noise)))]
		}
	}
	glitched := make([]string, len(grid))
	for row := range grid {
		glitched[row] = string(grid[row])
	}
	return glitched
}
