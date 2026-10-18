package main

import (
	"os"
	"path/filepath"
	"testing"

	"saveme/internal/update"
)

func TestWorkerChangeWaitsForTransactionCleanup(t *testing.T) {
	const current, transaction, before, after, mode = "current", "transaction", "v0.2.0", "v0.2.1", 0700
	paths := update.Layout{Root: t.TempDir()}
	if err := os.WriteFile(filepath.Join(paths.Root, current), []byte(after), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(paths.Root, transaction), mode); err != nil {
		t.Fatal(err)
	}
	if workerChanged(paths, before) {
		const pendingError = "worker changed before transaction cleanup"
		t.Fatal(pendingError)
	}
	if err := os.Remove(filepath.Join(paths.Root, transaction)); err != nil {
		t.Fatal(err)
	}
	if !workerChanged(paths, before) || workerChanged(paths, after) {
		const versionError = "worker change did not follow committed version"
		t.Fatal(versionError)
	}
}
