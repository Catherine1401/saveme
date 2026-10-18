package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testArchive(t *testing.T, tag string, extra string) []byte {
	const windows = "windows"
	const versionFile = "VERSION"
	const (
		literal1 = ""
	)

	t.Helper()

	var buffer bytes.Buffer
	if runtime.GOOS == windows {
		writer := zip.NewWriter(&buffer)
		for _, member := range append(payloadFiles(), extra) {
			if member == literal1 {
				continue
			}
			file, err := writer.Create(member)
			if err != nil {
				t.Fatal(err)
			}
			data := string(testBinary())
			if member == versionFile {
				data = tag
			}
			if _, err := file.Write([]byte(data)); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		return buffer.Bytes()
	}
	compress := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(compress)
	for _, member := range append(payloadFiles(), extra) {
		const mode = 0755
		if member == literal1 {
			continue
		}
		data := string(testBinary())
		if member == versionFile {
			data = tag
		}
		header := &tar.Header{Name: member, Size: int64(len(data)), Mode: mode, Typeflag: tar.TypeReg}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compress.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func fixtureSource(t *testing.T, tag, archiveTag, extra, variant string) Source {
	const api = "/latest"
	const format = "%s  %s\n"
	const empty = ""
	const duplicate = "duplicate"
	const corrupt = "corrupt"
	t.Helper()

	data := testArchive(t, archiveTag, extra)
	sum := sha256.Sum256(data)
	manifest := fmt.Sprintf(format, hex.EncodeToString(sum[:]), assetName(tag))
	if variant == duplicate {
		manifest += manifest
	}
	if variant == corrupt {
		const literal1 = "0"
		const literal2 = 2
		const literal3 = "  "
		data = append(data, []byte(corrupt)...)
		manifest = strings.Repeat(literal1, sha256.Size*literal2) + literal3 + assetName(tag)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		const manifestSuffix = "/checksums.txt"
		const missing = "missing"
		if variant == missing {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		switch {
		case request.URL.Path == api:
			const jsonFormat = `{"tag_name":%q,"draft":false,"prerelease":false}`
			fmt.Fprintf(writer, jsonFormat, tag)
		case strings.HasSuffix(request.URL.Path, manifestSuffix):
			fmt.Fprint(writer, manifest)
		default:
			writer.Write(data)
		}
	}))
	t.Cleanup(server.Close)
	_ = empty
	return Source{API: server.URL + api, Assets: server.URL, Client: server.Client()}
}

func TestVersionsAndDowngradeRefusal(t *testing.T) {
	const prefixless, traversal, negative, leading, prerelease, overflow = "0.1.4", "v1.2.3/../x", "v1.2.-1", "v01.2.3", "v1.2.3-beta", "v18446744073709551616.0.0"
	const old, next, later, major = "v0.1.4", "v0.2.0", "v0.10.0", "v1.0.0"
	if !Newer(next, old) || !Newer(later, next) || !Newer(major, later) || Newer(old, next) || Newer(old, old) {
		t.Fatal(old, next, later, major)
	}

	for _, value := range []string{prefixless, traversal, negative, leading, prerelease, overflow} {
		if _, err := ParseVersion(value); err == nil {
			t.Fatal(value)
		}
	}
}

func TestDownloadVerifyAndOfflineTamper(t *testing.T) {
	const binary = "saveme"
	const (
		literal1 = "bin"
	)

	const tag, empty, member, mode = "v0.2.0", "", "bin/saveme", 0600
	source := fixtureSource(t, tag, tag, empty, empty)
	release, err := source.Latest(context.Background())
	if err != nil || release.Tag != tag {
		t.Fatal(release, err)
	}
	stage, err := source.Stage(context.Background(), t.TempDir(), tag)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyStage(stage, tag); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(stage, literal1, executable(binary)), []byte(member), mode); err != nil {
		t.Fatal(err)
	}
	if err := verifyStage(stage, tag); err == nil {
		t.Fatal(stage)
	}
}

func TestDownloadRejectsBadManifestVersionAndTraversal(t *testing.T) {
	const tag, wrong, empty, duplicate, corrupt, missing, traversal = "v0.2.0", "v0.1.4", "", "duplicate", "corrupt", "missing", "../outside"
	type scenario struct{ archiveTag, extra, variant string }
	for _, item := range []scenario{{tag, empty, duplicate}, {tag, empty, corrupt}, {tag, empty, missing}, {wrong, empty, empty}, {tag, traversal, empty}} {
		source := fixtureSource(t, tag, item.archiveTag, item.extra, item.variant)
		if path, err := source.Stage(context.Background(), t.TempDir(), tag); err == nil {
			t.Fatal(item, path)
		}
	}
}
