package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Tuỳ chọn giao diện của người dùng; trường rỗng nghĩa là dùng mặc định.
type Preferences struct {
	Theme   string   `json:"theme,omitempty"`
	Clock   string   `json:"clock,omitempty"`
	Effect  string   `json:"effect,omitempty"`
	Effects []string `json:"effects"`
	Warning *bool    `json:"warning,omitempty"`
}

// Thư mục cấu hình saveme của người dùng.
func preferencesDir() (string, error) {
	const app, empty = "saveme", ""
	dir, err := os.UserConfigDir()
	if err != nil {
		return empty, err
	}
	return filepath.Join(dir, app), nil
}

// Đường dẫn file tuỳ chọn.
func PreferencesPath() (string, error) {
	const name, empty = "preferences.json", ""
	dir, err := preferencesDir()
	if err != nil {
		return empty, err
	}
	return filepath.Join(dir, name), nil
}

// Đọc tuỳ chọn; chưa có file mới thì lấy theme từ file theme.json cũ; file hỏng thì dùng mặc định.
func LoadPreferences() Preferences {
	const legacy = "theme.json"
	var saved Preferences
	path, err := PreferencesPath()
	if err != nil {
		return saved
	}
	data, err := os.ReadFile(path)
	if err == nil {
		if json.Unmarshal(data, &saved) != nil {
			return Preferences{}
		}
		return saved
	}
	if data, err = os.ReadFile(filepath.Join(filepath.Dir(path), legacy)); err == nil && json.Unmarshal(data, &saved) != nil {
		return Preferences{}
	}
	return saved
}

// Ghi tuỳ chọn qua file tạm rồi rename để không để lại file dở dang.
func SavePreferences(preferences Preferences) error {
	const dirMode, fileMode, pattern = 0700, 0600, ".preferences-*"
	path, err := PreferencesPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		return err
	}
	data, err := json.Marshal(preferences)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), pattern)
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(fileMode); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
