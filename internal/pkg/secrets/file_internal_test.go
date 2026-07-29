// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileSync(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x")

	if err := writeFileSync(path, []byte("data"), filePerm); err != nil {
		t.Fatalf("writeFileSync: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != filePerm {
		t.Fatalf("mode = %v, want %v", info.Mode().Perm(), filePerm)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "data" {
		t.Fatalf("content = %q, err = %v", data, err)
	}
}

func TestWriteFileSyncTruncatesExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x")
	if err := writeFileSync(path, []byte("long-previous-content"), filePerm); err != nil {
		t.Fatalf("first write: %v", err)
	}

	if err := writeFileSync(path, []byte("short"), filePerm); err != nil {
		t.Fatalf("second write: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "short" {
		t.Fatalf("content = %q, want %q", data, "short")
	}
}
