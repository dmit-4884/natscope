// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package secretsplit

import (
	"testing"

	"github.com/altessa-s/go-atlas/core/types/ptr"
)

type auth struct {
	Username *string // not a secret
	Password *string `behavior:"input_only" secret:"auth.password"`
	Token    *string `behavior:"input_only" secret:"auth.token"`
}

type doc struct {
	Name string
	Auth *auth
	Key  *string `behavior:"input_only" secret:"tls.key"`
}

func TestSplitLiftsAndBlanks(t *testing.T) {
	d := &doc{
		Name: "conn",
		Auth: &auth{Username: ptr.Wrap("u"), Password: ptr.Wrap("p"), Token: ptr.Wrap("t")},
		Key:  ptr.Wrap("k"),
	}

	secs, err := Split(d)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	want := map[string]string{"auth.password": "p", "auth.token": "t", "tls.key": "k"}
	if len(secs) != len(want) {
		t.Fatalf("secs = %v, want %v", secs, want)
	}
	for k, v := range want {
		if secs[k] != v {
			t.Errorf("secs[%q] = %q, want %q", k, secs[k], v)
		}
	}

	// Secrets blanked, non-secrets intact.
	if d.Auth.Password != nil || d.Auth.Token != nil || d.Key != nil {
		t.Errorf("secret fields not blanked: %+v key=%v", *d.Auth, d.Key)
	}
	if ptr.Unwrap(d.Auth.Username, "") != "u" || d.Name != "conn" {
		t.Errorf("non-secret fields altered: name=%q user=%v", d.Name, d.Auth.Username)
	}
}

func TestRoundTrip(t *testing.T) {
	d := &doc{
		Name: "conn",
		Auth: &auth{Username: ptr.Wrap("u"), Password: ptr.Wrap("p"), Token: ptr.Wrap("t")},
		Key:  ptr.Wrap("k"),
	}

	secs, err := Split(d)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if err := Merge(d, secs); err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if ptr.Unwrap(d.Auth.Password, "") != "p" ||
		ptr.Unwrap(d.Auth.Token, "") != "t" ||
		ptr.Unwrap(d.Key, "") != "k" {
		t.Errorf("secrets not restored: %+v key=%v", *d.Auth, d.Key)
	}
}

func TestSplitNilSubStruct(t *testing.T) {
	d := &doc{Name: "conn", Key: ptr.Wrap("k")} // Auth nil

	secs, err := Split(d)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if len(secs) != 1 || secs["tls.key"] != "k" {
		t.Fatalf("secs = %v, want only tls.key", secs)
	}

	// Merge back into a fresh doc with the same shape must not panic on nil Auth.
	got := &doc{Name: "conn", Key: ptr.Wrap("")}
	if err := Merge(got, secs); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if ptr.Unwrap(got.Key, "") != "k" {
		t.Errorf("key = %v, want k", got.Key)
	}
}

func TestSplitEmptyValueOmitted(t *testing.T) {
	d := &doc{
		Auth: &auth{Password: ptr.Wrap(""), Token: nil}, // empty and nil
		Key:  nil,
	}
	secs, err := Split(d)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if len(secs) != 0 {
		t.Fatalf("secs = %v, want empty (nil/empty secrets omitted)", secs)
	}
}

func TestMergeEmptyLeavesNil(t *testing.T) {
	d := &doc{Auth: &auth{Password: ptr.Wrap("stale")}}
	if err := Merge(d, map[string]string{}); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if d.Auth.Password != nil {
		t.Errorf("password = %v, want nil for a missing vault entry", d.Auth.Password)
	}
}

func TestMissingSecretTagErrors(t *testing.T) {
	type bad struct {
		Secret *string `behavior:"input_only"` // no `secret` tag
	}
	if _, err := Split(&bad{Secret: ptr.Wrap("x")}); err == nil {
		t.Fatal("expected error for input_only field without a secret tag")
	}
}

func TestUnsupportedKindErrors(t *testing.T) {
	type bad struct {
		Secret int `behavior:"input_only" secret:"x"`
	}
	if _, err := Split(&bad{Secret: 7}); err == nil {
		t.Fatal("expected error for a non-string secret field")
	}
}
