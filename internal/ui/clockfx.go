package ui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Hiệu ứng đồng hồ theo thứ tự hiện trên màn chọn; bật được nhiều hiệu ứng cùng lúc, mặc định chỉ blink.
func clockEffects() []option {
	const blink, slide, gradient, shimmer, flip, fill, breathe = "blink", "slide", "gradient", "shimmer", "flip", "fill", "breathe"
	const rainbow, glitch, scanline, neon, sparkle, tick, rain = "rainbow", "glitch", "scanline", "neon", "sparkle", "tick", "rain"
	const blinkLabel, slideLabel, gradientLabel, shimmerLabel, flipLabel, fillLabel, breatheLabel = "Blinking colons", "Sliding digits", "Color gradient", "Shimmer sweep", "Split-flap digits", "Progress fill", "Breathing glow"
	const rainbowLabel, glitchLabel, scanlineLabel, neonLabel, sparkleLabel, tickLabel, rainLabel = "Rainbow cycle", "Glitch", "Scanline", "Neon outline", "Sparkles", "Ticking seconds", "Digital rain"
	return []option{
		{name: blink, label: blinkLabel}, {name: slide, label: slideLabel}, {name: flip, label: flipLabel}, {name: tick, label: tickLabel},
		{name: fill, label: fillLabel}, {name: gradient, label: gradientLabel}, {name: rainbow, label: rainbowLabel}, {name: shimmer, label: shimmerLabel},
		{name: breathe, label: breatheLabel}, {name: scanline, label: scanlineLabel}, {name: glitch, label: glitchLabel}, {name: neon, label: neonLabel},
		{name: sparkle, label: sparkleLabel}, {name: rain, label: rainLabel},
	}
}

// Hiệu ứng có đang bật không.
func (m Model) effectOn(effect string) bool {
	return slices.Contains(m.appearance.effects, effect)
}

// Một lựa chọn trong màn chọn: tên lưu và nhãn hiển thị.
type option struct{ name, label string }

// Kiểu đồng hồ đang dùng; chưa đặt thì dùng mặc định.
func (m Model) clockStyle() clockStyle {
	const first, none = 0, ""
	if m.appearance.clock.name == none {
		return clockStyles()[first]
	}
	return m.appearance.clock
}

// Thời điểm vẽ hiện tại: lấy từ khung hoạt ảnh, chưa có thì dùng giờ máy.
func (m Model) frameTime() time.Time {
	if m.appearance.now.IsZero() {
		return time.Now()
	}
	return m.appearance.now
}

// Hiệu ứng chuyển động chỉ chạy khi không bật --no-motion.
func (m Model) moving(effect string) bool {
	return !m.appearance.still && m.effectOn(effect)
}

// Ghép glyph thành các dòng đồng hồ; blink ẩn dấu ':' nửa sau mỗi giây, slide trượt chữ số vừa đổi.
func (m Model) clockLines(text string) []string {
	const colon, colonIndex, blink, slide, flip, glitch, half, duration, space = ':', 10, "blink", "slide", "flip", "glitch", 500, 300 * time.Millisecond, " "
	const newline, zero = "\n", 0
	style := m.clockStyle()
	if style.plain {
		return []string{text}
	}
	if style.classic {
		return strings.Split(strings.TrimRight(drawClock(text), newline), newline)
	}
	now := m.frameTime()
	hideColon := m.moving(blink) && now.Nanosecond()/int(time.Millisecond) >= half
	elapsed := now.Sub(m.changedAt)
	height := len(style.glyphs[colonIndex])
	lines := make([]string, height)
	for position, char := range []rune(text) {
		glyph := m.glyphFor(style, char)
		if char == colon && hideColon {
			glyph = blankGlyph(glyph)
		}
		previous := []rune(m.previousClock)
		changed := elapsed >= zero && elapsed < duration && text == m.clockText && position < len(previous) && previous[position] != char
		switch {
		case changed && m.moving(flip):
			glyph = flipGlyph(m.glyphFor(style, previous[position]), glyph)
		case changed && m.moving(slide):
			glyph = slideGlyph(m.glyphFor(style, previous[position]), glyph, float64(elapsed)/float64(duration))
		}
		for row := range lines {
			if position > zero {
				lines[row] += strings.Repeat(space, style.gap)
			}
			lines[row] += glyph[row]
		}
	}
	if m.moving(glitch) {
		lines = glitchLines(lines, now)
	}
	return lines
}

// Glyph của một ký tự trong kiểu đồng hồ.
func (m Model) glyphFor(style clockStyle, char rune) []string {
	const colonIndex = 10
	if digit, err := strconv.Atoi(string(char)); err == nil {
		return style.glyphs[digit]
	}
	return style.glyphs[colonIndex]
}

// Glyph trống cùng kích thước.
func blankGlyph(glyph []string) []string {
	const space = " "
	blank := make([]string, len(glyph))
	for row, line := range glyph {
		blank[row] = strings.Repeat(space, lipgloss.Width(line))
	}
	return blank
}

// Số cũ trượt lên, số mới đi vào từ dưới theo tiến độ 0–1.
func slideGlyph(previous, next []string, progress float64) []string {
	const zero = 0
	height := len(next)
	shift := min(max(int(progress*float64(height)), zero), height)
	moved := make([]string, height)
	for row := range moved {
		if source := row + shift; source < height {
			moved[row] = previous[source]
		} else {
			moved[row] = next[source-height]
		}
	}
	return moved
}

// Màu trộn giữa màu nhấn và màu shimmer của theme; theme ANSI thì giữ màu nhấn.
func (m Model) blendStyle(share float64) lipgloss.Style {
	const none, first = "", 0
	theme := m.appearance.palette
	if theme.name == none {
		theme = palettes()[first]
	}
	dark, okDark := mix(theme.dark.accent, theme.dark.shimmer, share)
	light, okLight := mix(theme.light.accent, theme.light.shimmer, share)
	if !okDark || !okLight {
		return m.style(toneAccent)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Dark: dark, Light: light})
}

// Trộn hai màu hex theo tỉ lệ; false khi màu không phải hex.
func mix(from, to string, share float64) (string, bool) {
	const length, format, hash, empty, base, size, first, channels, red, green, blue = 7, "#%02x%02x%02x", '#', "", 16, 0, 1, 3, 0, 1, 2
	const width, head = 2, 0
	if len(from) != length || len(to) != length || from[head] != hash || to[head] != hash {
		return empty, false
	}
	var a, b [channels]int64
	for channel := range a {
		start := first + channel*width
		a[channel], _ = strconv.ParseInt(from[start:start+width], base, size)
		b[channel], _ = strconv.ParseInt(to[start:start+width], base, size)
	}
	blend := func(channel int) int64 { return a[channel] + int64(float64(b[channel]-a[channel])*share) }
	return fmt.Sprintf(format, blend(red), blend(green), blend(blue)), true
}
