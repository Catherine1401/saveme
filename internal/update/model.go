package update

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Request struct {
	Operation string `json:"operation"`
	Automatic bool   `json:"automatic,omitempty"`
}

type Status struct {
	Installed string `json:"installed"`
	Available string `json:"available,omitempty"`
	Phase     string `json:"phase"`
	Deferred  string `json:"deferred,omitempty"`
	LastError string `json:"last_error,omitempty"`
	Rejected  string `json:"rejected,omitempty"`
	Fatal     bool   `json:"fatal,omitempty"`
}

type Response struct {
	Status Status `json:"status"`
	Error  string `json:"error,omitempty"`
}

type Layout struct {
	Root, Endpoint, Launcher, Daemon, Config, Receipt string
}

func (p Layout) current() string {
	const name = "current"
	return filepath.Join(p.Root, name)
}

func (p Layout) Payload(version string) string {
	const folder, binary = "versions", "saveme"
	return filepath.Join(p.Root, folder, version, executable(binary))
}

func (p Layout) ReadVersion() (string, error) {
	data, err := os.ReadFile(p.current())
	if err != nil {
		const empty = ""
		return empty, err
	}
	version := strings.TrimSpace(string(data))
	_, err = ParseVersion(version)
	return version, err
}

type Version struct{ Major, Minor, Patch uint64 }

func ParseVersion(text string) (Version, error) {
	const prefix = "v"
	const separator = "."
	const count = 3
	const first = 0
	const second = 1
	const third = 2
	var version Version
	parts := strings.Split(strings.TrimPrefix(text, prefix), separator)
	if !strings.HasPrefix(text, prefix) || len(parts) != count {
		return version, os.ErrInvalid
	}
	values := [count]uint64{}
	for index, part := range parts {
		const zero = 0
		const low = '0'
		const base = 10
		const bits = 64
		const one = 1
		if len(part) == zero || (len(part) > one && part[zero] == byte(low)) {
			return version, os.ErrInvalid
		}
		for _, char := range part {
			const high = '9'
			if char < low || char > high {
				return version, os.ErrInvalid
			}
		}
		value, err := strconv.ParseUint(part, base, bits)
		if err != nil {
			return version, err
		}
		values[index] = value
	}
	return Version{values[first], values[second], values[third]}, nil
}

func Newer(candidate, installed string) bool {
	left, err := ParseVersion(candidate)
	if err != nil {
		return false
	}
	right, err := ParseVersion(installed)
	if err != nil {
		return false
	}
	if left.Major != right.Major {
		return left.Major > right.Major
	}
	if left.Minor != right.Minor {
		return left.Minor > right.Minor
	}
	return left.Patch > right.Patch
}

func writeJSON(path string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return atomicWrite(path, data)
}

func atomicWrite(path string, data []byte) error {
	const pattern, mode = ".update-*", 0600
	file, err := os.CreateTemp(filepath.Dir(path), pattern)
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err = file.Chmod(mode); err == nil {
		_, err = file.Write(data)
	}
	if err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return replace(file.Name(), path)
}

func errorText(err error) string {
	const format = "%v"
	if err == nil {
		const empty = ""
		return empty
	}
	return fmt.Sprintf(format, err)
}

type Refusal string

const OtherSessions Refusal = "other_sessions"

func (r Refusal) Error() string { return string(r) }
