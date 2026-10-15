package store

// Windows không cho flush handle thư mục; NTFS tự ghi journal metadata của rename.
func syncDir(string) error {
	return nil
}
