// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"testing"

	"github.com/altessa-s/go-atlas/core/types/ptr"

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

func desc(sourceID, tag string, set []byte) *entities.ProtoDescriptor {
	return entities.ProtoDescriptorNew(func(d *entities.ProtoDescriptor) {
		d.SourceID = sourceID
		d.Tag = tag
		d.DescriptorSet = set
		d.MessageTypes = []string{"pkg.A"}
	})
}

func TestDescriptors_UpsertBySourceTag(t *testing.T) {
	s := newDescStore(t)
	ctx := t.Context()

	if err := s.Save(ctx, desc("s1", "v1", []byte("first"))); err != nil {
		t.Fatalf("save 1: %v", err)
	}
	// Second save for the same (source, tag) replaces, not duplicates.
	if err := s.Save(ctx, desc("s1", "v1", []byte("second"))); err != nil {
		t.Fatalf("save 2: %v", err)
	}

	got, err := s.GetBySourceTag(ctx, "s1", "v1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got.DescriptorSet) != "second" {
		t.Fatalf("descriptor set = %q, want second", got.DescriptorSet)
	}

	all, err := s.GetAll(ctx)
	if err != nil {
		t.Fatalf("getall: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 descriptor after upsert, got %d", len(all))
	}
}

func TestDescriptors_FindByFingerprint(t *testing.T) {
	s := newDescStore(t)
	ctx := t.Context()
	set := []byte("payload-bytes")
	if err := s.Save(ctx, desc("s1", "v1", set)); err != nil {
		t.Fatalf("save: %v", err)
	}

	sum := sha256.Sum256(set)
	fp := hex.EncodeToString(sum[:])
	got, err := s.FindByFingerprint(ctx, fp)
	if err != nil {
		t.Fatalf("find by fingerprint: %v", err)
	}
	if got.SourceID != "s1" || got.Tag != "v1" {
		t.Fatalf("wrong descriptor: %+v", got)
	}

	if _, err := s.FindByFingerprint(ctx, ""); !errors.Is(err, errs.ErrProtoDescriptorNotFound) {
		t.Fatalf("empty fingerprint: want NotFound, got %v", err)
	}
	if _, err := s.FindByFingerprint(ctx, "deadbeef"); !errors.Is(err, errs.ErrProtoDescriptorNotFound) {
		t.Fatalf("unknown fingerprint: want NotFound, got %v", err)
	}
}

func TestDescriptors_ListFilterAndDeleteBySource(t *testing.T) {
	s := newDescStore(t)
	ctx := t.Context()
	_ = s.Save(ctx, desc("s1", "v1", []byte("a")))
	_ = s.Save(ctx, desc("s1", "v2", []byte("b")))
	_ = s.Save(ctx, desc("s2", "v1", []byte("c")))

	list, err := s.List(ctx, &entities.ProtoDescriptorsList{
		SourceID: ptr.Wrap("s1"),
		ListBase: entities.ListBase{Limit: ptr.Wrap(int64(10)), IncludeTotalCount: true},
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if *list.Total != 2 {
		t.Fatalf("total for s1 = %d, want 2", *list.Total)
	}

	n, err := s.DeleteBySource(ctx, "s1")
	if err != nil {
		t.Fatalf("delete by source: %v", err)
	}
	if n != 2 {
		t.Fatalf("deleted %d, want 2", n)
	}
	remaining, _ := s.GetAll(ctx)
	if len(remaining) != 1 || remaining[0].SourceID != "s2" {
		t.Fatalf("remaining = %+v", remaining)
	}
}

func TestDescriptors_GetByIDNotFound(t *testing.T) {
	s := newDescStore(t)
	if _, err := s.GetById(t.Context(), "nope"); !errors.Is(err, errs.ErrProtoDescriptorNotFound) {
		t.Fatalf("want NotFound, got %v", err)
	}
}
