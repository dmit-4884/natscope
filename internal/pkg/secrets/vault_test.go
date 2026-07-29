// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package secrets_test

import (
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/dmit-4884/natscope/internal/pkg/secrets"
)

func TestKeyringRoundTrip(t *testing.T) {
	keyring.MockInit()
	v := secrets.NewKeyring("natscope-test")
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

func TestKeyringGetMissingReturnsEmpty(t *testing.T) {
	keyring.MockInit()
	v := secrets.NewKeyring("natscope-test")
	got, err := v.Get(t.Context(), "connections", "nope")
	if err != nil {
		t.Fatalf("get missing: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty, got %v", got)
	}
}

func TestKeyringPutEmptyDeletes(t *testing.T) {
	keyring.MockInit()
	v := secrets.NewKeyring("natscope-test")
	ctx := t.Context()
	if err := v.Put(ctx, "connections", "c1", map[string]string{"auth.password": "pw"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// An all-empty set clears any stored secrets for the entity.
	if err := v.Put(ctx, "connections", "c1", map[string]string{"auth.password": ""}); err != nil {
		t.Fatalf("put empty: %v", err)
	}
	got, _ := v.Get(ctx, "connections", "c1")
	if len(got) != 0 {
		t.Fatalf("want cleared, got %v", got)
	}
}

func TestKeyringDeleteIdempotent(t *testing.T) {
	keyring.MockInit()
	v := secrets.NewKeyring("natscope-test")
	if err := v.Delete(t.Context(), "connections", "ghost"); err != nil {
		t.Fatalf("delete missing should be nil: %v", err)
	}
}

func TestMemoryRoundTrip(t *testing.T) {
	v := secrets.NewMemory()
	ctx := t.Context()
	if err := v.Put(ctx, "ns", "id", map[string]string{"k": "val"}); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, _ := v.Get(ctx, "ns", "id")
	if got["k"] != "val" {
		t.Fatalf("memory round trip: %v", got)
	}
	if err := v.Delete(ctx, "ns", "id"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, _ = v.Get(ctx, "ns", "id")
	if len(got) != 0 {
		t.Fatalf("after delete: %v", got)
	}
}
