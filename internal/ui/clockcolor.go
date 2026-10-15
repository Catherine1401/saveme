package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Tô màu đồng hồ theo từng ô; progress < 0 nghĩa là không biết tiến độ.
func (m Model) paintClock(lines []string, remaining int64, progress float64) string {
	const newline, first = "\n", 0
	style := m.clockStyle()
	grid, pad := m.clockGrid(lines, style.dim)
	var out strings.Builder
	for row, cells := range grid {
		for column, cell := range cells {
			text := string(cell.char)
			if m.appearance.plain || cell.kind == blankCell {
				out.WriteString(text)
				continue
			}
			out.WriteString(m.cellStyle(cell, row-pad, column-pad, len(lines), lipgloss.Width(lines[first]), remaining, progress).Render(text))
		}
		out.WriteString(newline)
	}
	return strings.TrimSuffix(out.String(), newline)
}

// Chọn style cho một ô theo thứ tự ưu tiên của các hiệu ứng.
func (m Model) cellStyle(cell clockCell, row, column, height, width int, remaining int64, progress float64) lipgloss.Style {
	const finalMinute, half, second, zero, one, halves = 60, 500, 1000, 0, 1, 2
	const shimmer, scanline, tick, fill, rainbow, gradient, breathe = "shimmer", "scanline", "tick", "fill", "rainbow", "gradient", "breathe"
	now := m.frameTime()
	millis := now.UnixMilli()
	switch cell.kind {
	case dimCell, neonCell, rainCell:
		return m.style(toneMuted)
	case sparkleCell:
		return m.style(toneShimmer).Bold(true)
	}
	if m.appearance.warning && remaining <= finalMinute {
		faint := !m.appearance.still && millis%second >= half
		return m.style(toneWarning).Bold(true).Faint(faint)
	}
	if strings.ContainsRune(glitchNoise(), cell.char) {
		return m.style(toneConfirm).Bold(true)
	}
	base := m.style(toneAccent).Bold(true)
	switch {
	case m.moving(scanline) && row == int(millis%scanPeriod())*height/int(scanPeriod()):
		base = m.style(toneShimmer).Bold(true)
	case m.moving(shimmer) && m.inSweep(column, width):
		base = m.style(toneShimmer).Bold(true)
	case m.moving(tick) && column >= m.secondsStart(width) && millis%second < tickWindow():
		base = m.style(toneShimmer).Bold(true)
	case m.effectOn(fill) && progress >= zero && row < int(float64(height)*(one-progress)):
		base = m.style(toneMuted).Bold(true)
	case m.effectOn(rainbow):
		base = m.rainbowStyle(column, width, now).Bold(true)
	case m.effectOn(gradient):
		base = m.blendStyle(float64(column) / float64(max(width-one, one))).Bold(true)
	}
	if m.moving(breathe) && millis%breathPeriod() >= breathPeriod()/halves {
		base = base.Faint(true)
	}
	return base
}

func glitchNoise() string             { const noise = "▓▒"; return noise }
func scanPeriod() int64               { const period = 2000; return period }
func breathPeriod() int64             { const period = 3000; return period }
func tickWindow() int64               { const window = 150; return window }
func sweepPeriod() time.Duration      { const period = 4 * time.Second; return period }
func (m Model) sweepBand() (int, int) { const band, double = 3, 2; return band, double }

// Cột đang nằm trong vệt sáng shimmer chạy ngang.
func (m Model) inSweep(column, width int) bool {
	band, double := m.sweepBand()
	period := sweepPeriod().Milliseconds()
	sweep := int(m.frameTime().UnixMilli()%period) * (width + band*double) / int(period)
	return column >= sweep-band*double && column < sweep-band
}

// Cột bắt đầu của hai chữ số giây (hai glyph cuối).
func (m Model) secondsStart(width int) int {
	const digits, first = 2, 0
	style := m.clockStyle()
	if style.glyphs == nil {
		return width
	}
	digit := lipgloss.Width(style.glyphs[first][first])
	return width - digits*digit - style.gap
}

// Màu cầu vồng chạy vòng qua các vai trò màu của theme, lệch pha theo cột.
func (m Model) rainbowStyle(column, width int, now time.Time) lipgloss.Style {
	const period, zero, one = 4000, 0, 1
	tones := []tone{toneAccent, toneConfirm, toneShimmer, toneWarning}
	phase := float64(zero)
	if !m.appearance.still {
		phase = float64(now.UnixMilli()%period) / period
	}
	share := float64(column)/float64(max(width, one)) + phase
	index := int(share*float64(len(tones))) % len(tones)
	return m.style(tones[index])
}
