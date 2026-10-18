package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type VerifiedFile struct {
	Name string `json:"name"`
	Hash string `json:"sha256"`
}

type Manifest struct {
	Version string         `json:"version"`
	Files   []VerifiedFile `json:"files"`
}

func sealStage(folder, version string) error {
	const name = "verified.json"
	manifest := Manifest{Version: version}
	for _, member := range payloadFiles() {
		hash, err := stageHash(folder, member)
		if err != nil {
			return err
		}
		manifest.Files = append(manifest.Files, VerifiedFile{Name: member, Hash: hash})
	}
	return writeJSON(filepath.Join(folder, name), manifest)
}

func stageHash(folder, member string) (string, error) {
	const empty = ""
	path := filepath.Join(folder, filepath.FromSlash(member))
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return empty, os.ErrInvalid
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return empty, err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func verifyStage(folder, version string) error {
	const name, versionFile = "verified.json", "VERSION"
	data, err := os.ReadFile(filepath.Join(folder, name))
	if err != nil {
		return err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	if manifest.Version != version || len(manifest.Files) != len(payloadFiles()) {
		return os.ErrInvalid
	}
	for index, member := range payloadFiles() {
		file := manifest.Files[index]
		hash, err := stageHash(folder, member)
		if err != nil || file.Name != member || file.Hash != hash {
			return os.ErrInvalid
		}
	}
	data, err = os.ReadFile(filepath.Join(folder, versionFile))
	if err != nil || strings.TrimSpace(string(data)) != version {
		return os.ErrInvalid
	}
	return nil
}
