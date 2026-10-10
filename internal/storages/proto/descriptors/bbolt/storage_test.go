// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package bbolt_test

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	descbbolt "github.com/dmit-4884/natscope/internal/storages/proto/descriptors/bbolt"
)

func newDescStore(t *testing.T) *descbbolt.Storage {
	t.Helper()
	db, err := bbstore.NewDB(filepath.Join(t.TempDir(), "test.bolt"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s, err := descbbolt.New(t.Context(), db)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	return s
}

func desc(sourceID, revision, fingerprint string) *entities.ProtoDescriptor {
	return entities.ProtoDescriptorNew(func(d *entities.ProtoDescriptor) {
		d.SourceID = sourceID
		d.Revision = revision
		d.DescriptorSet = []byte(sourceID + revision)
		d.Fingerprint = fingerprint
		d.MessageTypes = []string{"a.B"}
		d.CompiledAt = 42
	})
}

func TestDescriptors_SaveGetRoundTrip(t *testing.T) {
	s := newDescStore(t)

	if err := s.Save(t.Context(), desc("s1", "abc", "fp1")); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := s.GetBySourceRevision(t.Context(), "s1", "abc")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !bytes.Equal(got.DescriptorSet, []byte("s1abc")) || got.Fingerprint != "fp1" || got.CompiledAt != 42 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if len(got.MessageTypes) != 1 || got.MessageTypes[0] != "a.B" {
		t.Fatalf("message types mismatch: %+v", got.MessageTypes)
	}
}

func TestDescriptors_SaveReplacesSameRevision(t *testing.T) {
	s := newDescStore(t)

	if err := s.Save(t.Context(), desc("s1", "local", "old")); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.Save(t.Context(), desc("s1", "local", "new")); err != nil {
		t.Fatalf("save: %v", err)
	}
	list, err := s.ListBySource(t.Context(), "s1")
	if err != nil || len(list) != 1 || list[0].Fingerprint != "new" {
		t.Fatalf("want one replaced schema, got %v %+v", err, list)
	}
	if _, err := s.GetByFingerprint(t.Context(), "s1", "old"); !errors.Is(err, errs.ErrProtoDescriptorNotFound) {
		t.Fatalf("old fingerprint should be gone, got %v", err)
	}
}

func TestDescriptors_GetByFingerprintIsScopedToSource(t *testing.T) {
	s := newDescStore(t)

	if err := s.Save(t.Context(), desc("s1", "a", "same")); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.Save(t.Context(), desc("s2", "b", "same")); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := s.GetByFingerprint(t.Context(), "s2", "same")
	if err != nil || got.SourceID != "s2" || got.Revision != "b" {
		t.Fatalf("want s2/b, got %v %+v", err, got)
	}
	if _, err := s.GetByFingerprint(t.Context(), "s3", "same"); !errors.Is(err, errs.ErrProtoDescriptorNotFound) {
		t.Fatalf("other source must not match, got %v", err)
	}
	if _, err := s.GetByFingerprint(t.Context(), "s1", ""); !errors.Is(err, errs.ErrProtoDescriptorNotFound) {
		t.Fatalf("empty fingerprint must not match, got %v", err)
	}
}

func TestDescriptors_ListAndDeleteBySource(t *testing.T) {
	s := newDescStore(t)

	for _, d := range []*entities.ProtoDescriptor{desc("s1", "a", "1"), desc("s1", "b", "2"), desc("s2", "a", "3")} {
		if err := s.Save(t.Context(), d); err != nil {
			t.Fatalf("save: %v", err)
		}
	}
	list, err := s.ListBySource(t.Context(), "s1")
	if err != nil || len(list) != 2 {
		t.Fatalf("list s1: %v %d", err, len(list))
	}
	n, err := s.DeleteBySource(t.Context(), "s1")
	if err != nil || n != 2 {
		t.Fatalf("delete: n=%d err=%v", n, err)
	}
	if _, err := s.GetBySourceRevision(t.Context(), "s1", "a"); !errors.Is(err, errs.ErrProtoDescriptorNotFound) {
		t.Fatalf("s1 should be gone, got %v", err)
	}
	if _, err := s.GetBySourceRevision(t.Context(), "s2", "a"); err != nil {
		t.Fatalf("s2 should survive: %v", err)
	}
}
