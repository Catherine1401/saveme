package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

func payloadFiles() []string {
	const bin, cli, daemon, helper, version = "bin/", "saveme", "saved", "saveme-update", "VERSION"
	return []string{bin + executable(cli), bin + executable(daemon), bin + executable(helper), version}
}

func unpack(archive, target, tag string) error {
	const windows, mode, versionName = "windows", 0700, "VERSION"
	if err := os.MkdirAll(target, mode); err != nil {
		return err
	}
	var err error
	if runtime.GOOS == windows {
		err = unzip(archive, target)
	} else {
		err = untar(archive, target)
	}
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(target, versionName))
	if err != nil || strings.TrimSpace(string(data)) != tag {
		return os.ErrInvalid
	}
	for _, member := range payloadFiles() {
		info, err := os.Lstat(filepath.Join(target, filepath.FromSlash(member)))
		if err != nil || !info.Mode().IsRegular() {
			return os.ErrInvalid
		}
		if member != versionName {
			if err := checkBinary(filepath.Join(target, filepath.FromSlash(member))); err != nil {
				return err
			}
		}
	}
	return nil
}

func wanted(name string) bool {
	for _, member := range payloadFiles() {
		if name == member {
			return true
		}
	}
	return false
}

func safeMember(name string) bool {
	const slash, backslash, traversal, colon = "/", `\`, "..", ":"
	return path.Clean(name) == strings.TrimSuffix(name, slash) && !strings.HasPrefix(name, slash) && !strings.Contains(name, backslash) && !strings.Contains(name, traversal) && !strings.Contains(name, colon)
}

func saveMember(reader io.Reader, folder, name string, size int64) error {
	const fileLimit, mode, directoryMode, zero = 64 * 1024 * 1024, 0755, 0700, 0
	if size < zero || size > fileLimit {
		return os.ErrInvalid
	}
	target := filepath.Join(folder, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(target), directoryMode); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, err = io.CopyN(file, reader, size)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

func untar(archive, target string) error {
	const maxMembers, zero = 256, 0
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer compressed.Close()
	reader := tar.NewReader(compressed)

	var total int64
	for count := zero; count < maxMembers; count++ {
		const totalLimit = 256 * 1024 * 1024
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		total += header.Size
		if header.Size < zero || total > totalLimit {
			return os.ErrInvalid
		}
		if !safeMember(header.Name) || (header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir) {
			return os.ErrInvalid
		}
		if wanted(header.Name) {
			if header.Typeflag != tar.TypeReg {
				return os.ErrInvalid
			}
			if err := saveMember(reader, target, header.Name, header.Size); err != nil {
				return err
			}
		}
	}
	return os.ErrInvalid
}

func unzip(archive, target string) error {
	const maxMembers = 256
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer reader.Close()
	if len(reader.File) > maxMembers {
		return os.ErrInvalid
	}

	var total uint64
	for _, member := range reader.File {
		const totalLimit = 256 * 1024 * 1024
		const literal1 = 0
		if member.UncompressedSize64 > totalLimit {
			return os.ErrInvalid
		}
		total += member.UncompressedSize64
		if total > totalLimit {
			return os.ErrInvalid
		}
		if !safeMember(member.Name) || member.Mode()&os.ModeSymlink != literal1 {
			return os.ErrInvalid
		}
		if !wanted(member.Name) {
			continue
		}
		if !member.Mode().IsRegular() {
			return os.ErrInvalid
		}
		file, err := member.Open()
		if err != nil {
			return err
		}
		err = saveMember(file, target, member.Name, int64(member.UncompressedSize64))
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
