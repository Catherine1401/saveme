package ui

// Màu theo vai trò cho một nền terminal.
type shades struct {
	accent, confirm, shimmer, muted, fault, warning, text string
}

// Theme màu: tên dùng trong cờ --theme và file lưu, nhãn hiện trên màn chọn.
type palette struct {
	name, label string
	dark, light shades
}

// Danh sách theme theo thứ tự hiện trên màn chọn; phần tử đầu là mặc định.
func palettes() []palette {
	return []palette{catppuccinPalette(), mintPalette(), draculaPalette(), gruvboxPalette(), solarizedPalette(), tokyoNightPalette(), rosePinePalette(), everforestPalette(), kanagawaPalette(), nightfoxPalette(), flexokiPalette(), selenizedPalette(), modusPalette(), ayuPalette(), nordPalette(), aNSIPalette()}
}

// Tìm theme theo tên.
func findPalette(name string) (palette, bool) {
	for _, item := range palettes() {
		if item.name == name {
			return item, true
		}
	}
	return palette{}, false
}

// Nguồn: catppuccin/palette (palette.json).
func catppuccinPalette() palette {
	const name, label = "catppuccin", "Catppuccin Mocha / Latte"
	const darkAccent, darkConfirm, darkShimmer, darkMuted, darkFault, darkWarning, darkText = "#cba6f7", "#a6e3a1", "#b4befe", "#6c7086", "#f38ba8", "#f9e2af", "#cdd6f4"
	const lightAccent, lightConfirm, lightShimmer, lightMuted, lightFault, lightWarning, lightText = "#8839ef", "#40a02b", "#7287fd", "#8c8fa1", "#d20f39", "#fe640b", "#4c4f69"
	return palette{name: name, label: label,
		dark:  shades{accent: darkAccent, confirm: darkConfirm, shimmer: darkShimmer, muted: darkMuted, fault: darkFault, warning: darkWarning, text: darkText},
		light: shades{accent: lightAccent, confirm: lightConfirm, shimmer: lightShimmer, muted: lightMuted, fault: lightFault, warning: lightWarning, text: lightText}}
}

// Bảng màu mint/tím đầu tiên của saveme.
func mintPalette() palette {
	const name, label = "mint", "Mint / violet (classic saveme)"
	const darkAccent, darkConfirm, darkShimmer, darkMuted, darkFault, darkWarning, darkText = "#78E2BF", "#BEA5F6", "#C8F8E8", "#89909F", "#F38BA8", "#E9C46A", "#DEE3EB"
	const lightAccent, lightConfirm, lightShimmer, lightMuted, lightFault, lightWarning, lightText = "#176B50", "#7042AB", "#247C61", "#626978", "#A72A47", "#855D0E", "#212121"
	return palette{name: name, label: label,
		dark:  shades{accent: darkAccent, confirm: darkConfirm, shimmer: darkShimmer, muted: darkMuted, fault: darkFault, warning: darkWarning, text: darkText},
		light: shades{accent: lightAccent, confirm: lightConfirm, shimmer: lightShimmer, muted: lightMuted, fault: lightFault, warning: lightWarning, text: lightText}}
}

// Nguồn: dracula/dracula-theme (bảng Alucard cho nền sáng).
func draculaPalette() palette {
	const name, label = "dracula", "Dracula / Alucard"
	const darkAccent, darkConfirm, darkShimmer, darkMuted, darkFault, darkWarning, darkText = "#bd93f9", "#50fa7b", "#ff79c6", "#6272a4", "#ff5555", "#ffb86c", "#f8f8f2"
	const lightAccent, lightConfirm, lightShimmer, lightMuted, lightFault, lightWarning, lightText = "#644ac9", "#14710a", "#a3144d", "#6c664b", "#cb3a2a", "#a34d14", "#1f1f1f"
	return palette{name: name, label: label,
		dark:  shades{accent: darkAccent, confirm: darkConfirm, shimmer: darkShimmer, muted: darkMuted, fault: darkFault, warning: darkWarning, text: darkText},
		light: shades{accent: lightAccent, confirm: lightConfirm, shimmer: lightShimmer, muted: lightMuted, fault: lightFault, warning: lightWarning, text: lightText}}
}

// Nguồn: morhetz/gruvbox (bản sáng dùng bộ “faded”).
func gruvboxPalette() palette {
	const name, label = "gruvbox", "Gruvbox dark / light"
	const darkAccent, darkConfirm, darkShimmer, darkMuted, darkFault, darkWarning, darkText = "#fe8019", "#8ec07c", "#fabd2f", "#928374", "#fb4934", "#fabd2f", "#ebdbb2"
	const lightAccent, lightConfirm, lightShimmer, lightMuted, lightFault, lightWarning, lightText = "#af3a03", "#427b58", "#d65d0e", "#7c6f64", "#9d0006", "#b57614", "#3c3836"
	return palette{name: name, label: label,
		dark:  shades{accent: darkAccent, confirm: darkConfirm, shimmer: darkShimmer, muted: darkMuted, fault: darkFault, warning: darkWarning, text: darkText},
		light: shades{accent: lightAccent, confirm: lightConfirm, shimmer: lightShimmer, muted: lightMuted, fault: lightFault, warning: lightWarning, text: lightText}}
}

// Nguồn: ethanschoonover.com/solarized.
func solarizedPalette() palette {
	const name, label = "solarized", "Solarized dark / light"
	const darkAccent, darkConfirm, darkShimmer, darkMuted, darkFault, darkWarning, darkText = "#268bd2", "#859900", "#2aa198", "#586e75", "#dc322f", "#b58900", "#839496"
	const lightAccent, lightConfirm, lightShimmer, lightMuted, lightFault, lightWarning, lightText = "#268bd2", "#859900", "#2aa198", "#93a1a1", "#dc322f", "#b58900", "#657b83"
	return palette{name: name, label: label,
		dark:  shades{accent: darkAccent, confirm: darkConfirm, shimmer: darkShimmer, muted: darkMuted, fault: darkFault, warning: darkWarning, text: darkText},
		light: shades{accent: lightAccent, confirm: lightConfirm, shimmer: lightShimmer, muted: lightMuted, fault: lightFault, warning: lightWarning, text: lightText}}
}

// Nguồn: folke/tokyonight.nvim (extras/kitty).
func tokyoNightPalette() palette {
	const name, label = "tokyonight", "Tokyo Night / Day"
	const darkAccent, darkConfirm, darkShimmer, darkMuted, darkFault, darkWarning, darkText = "#7aa2f7", "#9ece6a", "#8db0ff", "#414868", "#f7768e", "#e0af68", "#c0caf5"
	const lightAccent, lightConfirm, lightShimmer, lightMuted, lightFault, lightWarning, lightText = "#2e7de9", "#587539", "#358aff", "#8990b3", "#f52a65", "#8c6c3e", "#3760bf"
	return palette{name: name, label: label,
		dark:  shades{accent: darkAccent, confirm: darkConfirm, shimmer: darkShimmer, muted: darkMuted, fault: darkFault, warning: darkWarning, text: darkText},
		light: shades{accent: lightAccent, confirm: lightConfirm, shimmer: lightShimmer, muted: lightMuted, fault: lightFault, warning: lightWarning, text: lightText}}
}

// Nguồn: rose-pine/rose-pine-palette (palette.json).
func rosePinePalette() palette {
	const name, label = "rosepine", "Rosé Pine / Dawn"
	const darkAccent, darkConfirm, darkShimmer, darkMuted, darkFault, darkWarning, darkText = "#c4a7e7", "#9ccfd8", "#ebbcba", "#6e6a86", "#eb6f92", "#f6c177", "#e0def4"
	const lightAccent, lightConfirm, lightShimmer, lightMuted, lightFault, lightWarning, lightText = "#907aa9", "#286983", "#d7827e", "#9893a5", "#b4637a", "#ea9d34", "#464261"
	return palette{name: name, label: label,
		dark:  shades{accent: darkAccent, confirm: darkConfirm, shimmer: darkShimmer, muted: darkMuted, fault: darkFault, warning: darkWarning, text: darkText},
		light: shades{accent: lightAccent, confirm: lightConfirm, shimmer: lightShimmer, muted: lightMuted, fault: lightFault, warning: lightWarning, text: lightText}}
}

// Nguồn: sainnhe/everforest (palette.md, độ tương phản medium).
func everforestPalette() palette {
	const name, label = "everforest", "Everforest dark / light"
	const darkAccent, darkConfirm, darkShimmer, darkMuted, darkFault, darkWarning, darkText = "#a7c080", "#7fbbb3", "#83c092", "#859289", "#e67e80", "#dbbc7f", "#d3c6aa"
	const lightAccent, lightConfirm, lightShimmer, lightMuted, lightFault, lightWarning, lightText = "#8da101", "#3a94c5", "#35a77c", "#829181", "#f85552", "#dfa000", "#5c6a72"
	return palette{name: name, label: label,
		dark:  shades{accent: darkAccent, confirm: darkConfirm, shimmer: darkShimmer, muted: darkMuted, fault: darkFault, warning: darkWarning, text: darkText},
		light: shades{accent: lightAccent, confirm: lightConfirm, shimmer: lightShimmer, muted: lightMuted, fault: lightFault, warning: lightWarning, text: lightText}}
}

// Nguồn: rebelot/kanagawa.nvim (colors.lua).
func kanagawaPalette() palette {
	const name, label = "kanagawa", "Kanagawa Wave / Lotus"
	const darkAccent, darkConfirm, darkShimmer, darkMuted, darkFault, darkWarning, darkText = "#7e9cd8", "#98bb6c", "#7fb4ca", "#727169", "#e82424", "#ff9e3b", "#dcd7ba"
	const lightAccent, lightConfirm, lightShimmer, lightMuted, lightFault, lightWarning, lightText = "#4d699b", "#6f894e", "#5d57a3", "#8a8980", "#c84053", "#cc6d00", "#545464"
	return palette{name: name, label: label,
		dark:  shades{accent: darkAccent, confirm: darkConfirm, shimmer: darkShimmer, muted: darkMuted, fault: darkFault, warning: darkWarning, text: darkText},
		light: shades{accent: lightAccent, confirm: lightConfirm, shimmer: lightShimmer, muted: lightMuted, fault: lightFault, warning: lightWarning, text: lightText}}
}

// Nguồn: EdenEast/nightfox.nvim (extra/kitty).
func nightfoxPalette() palette {
	const name, label = "nightfox", "Nightfox / Dayfox"
	const darkAccent, darkConfirm, darkShimmer, darkMuted, darkFault, darkWarning, darkText = "#719cd6", "#81b29a", "#86abdc", "#738091", "#c94f6d", "#dbc074", "#cdcecf"
	const lightAccent, lightConfirm, lightShimmer, lightMuted, lightFault, lightWarning, lightText = "#2848a9", "#396847", "#4863b6", "#837a72", "#a5222f", "#ac5402", "#3d2b5a"
	return palette{name: name, label: label,
		dark:  shades{accent: darkAccent, confirm: darkConfirm, shimmer: darkShimmer, muted: darkMuted, fault: darkFault, warning: darkWarning, text: darkText},
		light: shades{accent: lightAccent, confirm: lightConfirm, shimmer: lightShimmer, muted: lightMuted, fault: lightFault, warning: lightWarning, text: lightText}}
}

// Nguồn: kepano/flexoki. Màu muted và text chưa xác minh từ nguồn.
func flexokiPalette() palette {
	const name, label = "flexoki", "Flexoki dark / light"
	const darkAccent, darkConfirm, darkShimmer, darkMuted, darkFault, darkWarning, darkText = "#3aa99f", "#879a39", "#4385be", "#878580", "#d14d41", "#d0a215", "#cecdc3"
	const lightAccent, lightConfirm, lightShimmer, lightMuted, lightFault, lightWarning, lightText = "#24837b", "#66800b", "#205ea6", "#6f6e69", "#af3029", "#ad8301", "#100f0f"
	return palette{name: name, label: label,
		dark:  shades{accent: darkAccent, confirm: darkConfirm, shimmer: darkShimmer, muted: darkMuted, fault: darkFault, warning: darkWarning, text: darkText},
		light: shades{accent: lightAccent, confirm: lightConfirm, shimmer: lightShimmer, muted: lightMuted, fault: lightFault, warning: lightWarning, text: lightText}}
}

// Nguồn: jan-warchol/selenized (the-values.md). Shimmer nền sáng là violet thay thế.
func selenizedPalette() palette {
	const name, label = "selenized", "Selenized dark / light"
	const darkAccent, darkConfirm, darkShimmer, darkMuted, darkFault, darkWarning, darkText = "#4695f7", "#75b938", "#58a3ff", "#72898f", "#fa5750", "#dbb32d", "#adbcbc"
	const lightAccent, lightConfirm, lightShimmer, lightMuted, lightFault, lightWarning, lightText = "#0072d4", "#489100", "#8762c6", "#909995", "#d2212d", "#ad8900", "#53676d"
	return palette{name: name, label: label,
		dark:  shades{accent: darkAccent, confirm: darkConfirm, shimmer: darkShimmer, muted: darkMuted, fault: darkFault, warning: darkWarning, text: darkText},
		light: shades{accent: lightAccent, confirm: lightConfirm, shimmer: lightShimmer, muted: lightMuted, fault: lightFault, warning: lightWarning, text: lightText}}
}

// Nguồn: protesilaos/modus-themes. Tương phản cao nhất trong danh sách.
func modusPalette() palette {
	const name, label = "modus", "Modus Vivendi / Operandi"
	const darkAccent, darkConfirm, darkShimmer, darkMuted, darkFault, darkWarning, darkText = "#79a8ff", "#44bc44", "#00bcff", "#989898", "#ff5f59", "#fec43f", "#ffffff"
	const lightAccent, lightConfirm, lightShimmer, lightMuted, lightFault, lightWarning, lightText = "#3548cf", "#006800", "#0031a9", "#595959", "#a60000", "#884900", "#000000"
	return palette{name: name, label: label,
		dark:  shades{accent: darkAccent, confirm: darkConfirm, shimmer: darkShimmer, muted: darkMuted, fault: darkFault, warning: darkWarning, text: darkText},
		light: shades{accent: lightAccent, confirm: lightConfirm, shimmer: lightShimmer, muted: lightMuted, fault: lightFault, warning: lightWarning, text: lightText}}
}

// Nguồn: ayu-theme/ayu-colors. Giá trị là màu gốc trước bước build; text nền sáng chưa xác minh.
func ayuPalette() palette {
	const name, label = "ayu", "Ayu Dark / Light"
	const darkAccent, darkConfirm, darkShimmer, darkMuted, darkFault, darkWarning, darkText = "#ffb454", "#aad94c", "#e6c08a", "#5a6673", "#f07178", "#ff8f40", "#bfbdb6"
	const lightAccent, lightConfirm, lightShimmer, lightMuted, lightFault, lightWarning, lightText = "#fa8532", "#86b300", "#e59645", "#adaeb1", "#f07171", "#eba400", "#5c6166"
	return palette{name: name, label: label,
		dark:  shades{accent: darkAccent, confirm: darkConfirm, shimmer: darkShimmer, muted: darkMuted, fault: darkFault, warning: darkWarning, text: darkText},
		light: shades{accent: lightAccent, confirm: lightConfirm, shimmer: lightShimmer, muted: lightMuted, fault: lightFault, warning: lightWarning, text: lightText}}
}

// Nguồn: nordtheme/nord. Nord không có bản sáng chính thức; bản sáng ở đây ghép từ chính các màu Nord.
func nordPalette() palette {
	const name, label = "nord", "Nord"
	const darkAccent, darkConfirm, darkShimmer, darkMuted, darkFault, darkWarning, darkText = "#88c0d0", "#a3be8c", "#8fbcbb", "#4c566a", "#bf616a", "#ebcb8b", "#d8dee9"
	const lightAccent, lightConfirm, lightShimmer, lightMuted, lightFault, lightWarning, lightText = "#5e81ac", "#a3be8c", "#81a1c1", "#4c566a", "#bf616a", "#d08770", "#2e3440"
	return palette{name: name, label: label,
		dark:  shades{accent: darkAccent, confirm: darkConfirm, shimmer: darkShimmer, muted: darkMuted, fault: darkFault, warning: darkWarning, text: darkText},
		light: shades{accent: lightAccent, confirm: lightConfirm, shimmer: lightShimmer, muted: lightMuted, fault: lightFault, warning: lightWarning, text: lightText}}
}

// Màu ANSI 16 của terminal; màu thật theo theme terminal của người dùng.
func aNSIPalette() palette {
	const name, label = "ansi", "ANSI 16 colors (terminal theme)"
	const darkAccent, darkConfirm, darkShimmer, darkMuted, darkFault, darkWarning, darkText = "6", "5", "14", "8", "1", "3", ""
	const lightAccent, lightConfirm, lightShimmer, lightMuted, lightFault, lightWarning, lightText = "6", "5", "14", "8", "1", "3", ""
	return palette{name: name, label: label,
		dark:  shades{accent: darkAccent, confirm: darkConfirm, shimmer: darkShimmer, muted: darkMuted, fault: darkFault, warning: darkWarning, text: darkText},
		light: shades{accent: lightAccent, confirm: lightConfirm, shimmer: lightShimmer, muted: lightMuted, fault: lightFault, warning: lightWarning, text: lightText}}
}
