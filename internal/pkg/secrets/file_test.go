// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package secrets_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dmit-4884/natscope/internal/pkg/secrets"
)

func TestFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	v, err := secrets.NewFile(dir)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	ctx := t.Context()

	if err := v.Put(ctx, "connections", "c1", map[string]string{
		"auth.password": "pw", "auth.token": "tok",
	}); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, err := v.Get(ctx, "connections", "c1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got["auth.password"] != "pw" || got["auth.token"] != "tok" {
		t.Fatalf("round trip: %v", got)
	}
}

func TestFilePersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	ctx := t.Context()

	v1, err := secrets.NewFile(dir)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := v1.Put(ctx, "sources", "s1", map[string]string{"auth.token": "ghp_x"}); err != nil {
		t.Fatalf("put: %v", err)
	}

	v2, err := secrets.NewFile(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	got, err := v2.Get(ctx, "sources", "s1")
	if err != nil {
		t.Fatalf("get after reopen: %v", err)
	}
	if got["auth.token"] != "ghp_x" {
		t.Fatalf("want persisted secret, got %v", got)
	}
}

func TestFileVaultIsEncryptedOnDisk(t *testing.T) {
	dir := t.TempDir()
	ctx := t.Context()
	v, err := secrets.NewFile(dir)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := v.Put(ctx, "connections", "c1", map[string]string{"auth.password": "hunter2"}); err != nil {
		t.Fatalf("put: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "secrets.vault"))
	if err != nil {
		t.Fatalf("read vault file: %v", err)
	}
	if strings.Contains(string(raw), "hunter2") {
		t.Fatal("secret stored in plaintext")
	}
}

func TestFilePutEmptyDeletes(t *testing.T) {
	dir := t.TempDir()
	ctx := t.Context()
	v, err := secrets.NewFile(dir)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := v.Put(ctx, "connections", "c1", map[string]string{"auth.password": "pw"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := v.Put(ctx, "connections", "c1", map[string]string{"auth.password": ""}); err != nil {
		t.Fatalf("put empty: %v", err)
	}
	got, _ := v.Get(ctx, "connections", "c1")
	if len(got) != 0 {
		t.Fatalf("want cleared, got %v", got)
	}
}

func TestFileDeleteIdempotent(t *testing.T) {
	dir := t.TempDir()
	v, err := secrets.NewFile(dir)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := v.Delete(t.Context(), "connections", "ghost"); err != nil {
		t.Fatalf("delete missing should be nil: %v", err)
	}
}

func TestFileWrongKeyFailsClearly(t *testing.T) {
	t.Setenv("SECRETS__FILE_KEY", "") // force the co-located key path
	dir := t.TempDir()
	ctx := t.Context()
	v, err := secrets.NewFile(dir)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := v.Put(ctx, "connections", "c1", map[string]string{"auth.password": "pw"}); err != nil {
		t.Fatalf("put: %v", err)
	}

	// Replace the key: the vault must refuse to open with a clear error.
	keyPath := filepath.Join(dir, "vault.key")
	if err := os.WriteFile(keyPath, []byte(strings.Repeat("ab", 32)+"\n"), 0o600); err != nil {
		t.Fatalf("swap key: %v", err)
	}
	if _, err := secrets.NewFile(dir); !errors.Is(err, secrets.ErrVaultKeyMismatch) {
		t.Fatalf("want ErrVaultKeyMismatch, got %v", err)
	}
}

// TestFilePersistLeavesNoTempFile pins that a completed write publishes exactly
// the vault file: a stray .tmp means the atomic replace did not run to the end.
func TestFilePersistLeavesNoTempFile(t *testing.T) {
	dir := t.TempDir()
	v, err := secrets.NewFile(dir)
	if err != nil {
		t.Fatalf("NewFile: %v", err)
	}
	if err := v.Put(t.Context(), "conn", "id-1", map[string]string{"password": "pw"}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "secrets.vault.tmp")); !os.IsNotExist(err) {
		t.Fatalf("temp vault file must not survive a successful persist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "secrets.vault")); err != nil {
		t.Fatalf("vault file must exist after persist: %v", err)
	}
}

// TestFilePersistSurvivesReopen proves the bytes actually reached the file, not
// just the in-memory map.
func TestFilePersistSurvivesReopen(t *testing.T) {
	dir := t.TempDir()
	v, err := secrets.NewFile(dir)
	if err != nil {
		t.Fatalf("NewFile: %v", err)
	}
	if err := v.Put(t.Context(), "conn", "id-1", map[string]string{"password": "pw"}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	reopened, err := secrets.NewFile(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	got, err := reopened.Get(t.Context(), "conn", "id-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got["password"] != "pw" {
		t.Fatalf("secret did not survive reopen: %#v", got)
	}
}
