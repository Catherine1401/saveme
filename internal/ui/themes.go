package ui

// Tên mọi theme hợp lệ, theo thứ tự hiển thị.
func ThemeNames() []string {
	return names(palettes(), func(item palette) string { return item.name })
}

// Tên mọi kiểu đồng hồ hợp lệ.
func ClockNames() []string {
	return names(clockStyles(), func(style clockStyle) string { return style.name })
}

// Tên mọi hiệu ứng đồng hồ hợp lệ.
func EffectNames() []string {
	return names(clockEffects(), func(effect option) string { return effect.name })
}

// Lấy tên của từng mục theo đúng thứ tự.
func names[T any](items []T, name func(T) string) []string {
	var result []string
	for _, item := range items {
		result = append(result, name(item))
	}
	return result
}
