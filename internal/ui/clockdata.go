package ui

import (
	_ "embed"
	"encoding/json"
)

//go:embed clockdata.json
var clockDataFile []byte

// Bộ ký tự khung theo tên kiểu nét.
type lineSets struct {
	Rounded lineSet `json:"rounded"`
	Double  lineSet `json:"double"`
	Light   lineSet `json:"light"`
	Heavy   lineSet `json:"heavy"`
}

// Dữ liệu vẽ đồng hồ: font điểm ảnh 3×5, bit braille, đoạn sáng bảy đoạn, bộ ký tự khung.
type clockTables struct {
	Pixel    [][]string `json:"pixel"`
	Braille  [][]rune   `json:"braille"`
	Segments []string   `json:"segments"`
	Lines    lineSets   `json:"lines"`
}

// Bảng dữ liệu nhúng, đọc một lần khi nạp package; file hỏng thì dừng ngay vì là lỗi build.
var clockData = loadClockData(clockDataFile)

func loadClockData(raw []byte) clockTables {
	var tables clockTables
	if err := json.Unmarshal(raw, &tables); err != nil {
		panic(err)
	}
	return tables
}
