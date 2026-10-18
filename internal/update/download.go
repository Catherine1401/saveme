package update

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Source struct {
	API, Assets string
	Client      *http.Client
}

type Release struct {
	Tag        string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

func GitHub() Source {
	const timeout = 2 * time.Minute
	const api = "https://api.github.com/repos/Catherine1401/saveme/releases/latest"
	const assets = "https://github.com/Catherine1401/saveme/releases/download"
	client := &http.Client{Timeout: timeout, CheckRedirect: func(request *http.Request, prior []*http.Request) error {
		const maximumRedirects = 10
		const scheme, host, suffix = "https", "github.com", ".githubusercontent.com"
		if len(prior) >= maximumRedirects || request.URL.Scheme != scheme || (request.URL.Hostname() != host && !strings.HasSuffix(request.URL.Hostname(), suffix)) {
			return os.ErrPermission
		}
		return nil
	}}
	return Source{api, assets, client}
}

func (s Source) fetch(ctx context.Context, url string, target io.Writer, limit int64) error {
	const method, agentName, agentValue, ok, one = "GET", "User-Agent", "saveme-updater", http.StatusOK, 1
	request, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set(agentName, agentValue)
	response, err := s.Client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != ok {
		const format = "release HTTP status %d"
		return fmt.Errorf(format, response.StatusCode)
	}
	count, err := io.Copy(target, io.LimitReader(response.Body, limit+one))
	if err != nil {
		return err
	}
	if count > limit {
		return os.ErrInvalid
	}
	return nil
}

func (s Source) Latest(ctx context.Context) (Release, error) {
	const limit = 1024 * 1024
	var body strings.Builder
	var release Release
	if err := s.fetch(ctx, s.API, &body, limit); err != nil {
		return release, err
	}
	if err := json.Unmarshal([]byte(body.String()), &release); err != nil {
		return release, err
	}
	if _, err := ParseVersion(release.Tag); err != nil {
		return release, err
	}
	if release.Draft || release.Prerelease {
		return release, os.ErrInvalid
	}
	return release, nil
}

func assetName(tag string) string {
	const windows = "windows"
	const tar = "tar.gz"
	const prefix = "v"
	const format = "saveme_%s_%s_%s.%s"
	suffix := tar
	if runtime.GOOS == windows {
		const zip = "zip"
		suffix = zip
	}
	return fmt.Sprintf(format, strings.TrimPrefix(tag, prefix), runtime.GOOS, runtime.GOARCH, suffix)
}

func (s Source) Stage(ctx context.Context, root, tag string) (string, error) {
	const payload = "payload"
	const empty, cache, pattern, slash, manifest, archive = "", "cache", ".download-*", "/", "checksums.txt", "bundle"
	const mode, manifestLimit, archiveLimit = 0700, 1024 * 1024, 64 * 1024 * 1024
	if _, err := ParseVersion(tag); err != nil {
		return empty, err
	}
	folder := filepath.Join(root, cache)
	if err := os.MkdirAll(folder, mode); err != nil {
		return empty, err
	}
	temporary, err := os.MkdirTemp(folder, pattern)
	if err != nil {
		return empty, err
	}
	defer os.RemoveAll(temporary)
	var checksums strings.Builder
	base := s.Assets + slash + tag + slash
	if err := s.fetch(ctx, base+manifest, &checksums, manifestLimit); err != nil {
		return empty, err
	}
	expected, err := selectChecksum(checksums.String(), assetName(tag))
	if err != nil {
		return empty, err
	}
	path := filepath.Join(temporary, archive)
	file, err := os.Create(path)
	if err != nil {
		return empty, err
	}
	hash := sha256.New()
	err = s.fetch(ctx, base+assetName(tag), io.MultiWriter(file, hash), archiveLimit)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return empty, err
	}
	if hex.EncodeToString(hash.Sum(nil)) != expected {
		return empty, os.ErrInvalid
	}

	unpacked := filepath.Join(temporary, payload)
	if err := unpack(path, unpacked, tag); err != nil {
		return empty, err
	}
	if err := sealStage(unpacked, tag); err != nil {
		return empty, err
	}
	target := filepath.Join(folder, tag)
	if err := os.RemoveAll(target); err != nil {
		return empty, err
	}
	if err := os.Rename(unpacked, target); err != nil {
		return empty, err
	}
	return target, nil
}

func selectChecksum(text, name string) (string, error) {
	const empty = ""
	const zero = 0
	found := empty
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		const fieldsCount = 2
		const first = 0
		const second = 1
		const expectedBytes = sha256.Size
		fields := strings.Fields(scanner.Text())
		if len(fields) != fieldsCount || fields[second] != name {
			continue
		}
		decoded, err := hex.DecodeString(fields[first])
		if err != nil || len(decoded) != expectedBytes || found != empty {
			return empty, os.ErrInvalid
		}
		found = strings.ToLower(fields[first])
	}
	if err := scanner.Err(); err != nil {
		return empty, err
	}
	if len(found) == zero {
		return empty, os.ErrInvalid
	}
	return found, nil
}
