// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	selbbolt "github.com/dmit-4884/natscope/internal/storages/proto/selections/bbolt"
)

func newSelStore(t *testing.T) *selbbolt.Storage {
	t.Helper()
	db, err := bbstore.NewDB(filepath.Join(t.TempDir(), "test.bolt"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s, err := selbbolt.New(t.Context(), db)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	return s
}

func sel(sourceID, tag string) *entities.ProtoSelection {
	return entities.ProtoSelectionNew(func(s *entities.ProtoSelection) {
		s.SourceID = sourceID
		s.Tag = tag
	})
}

func TestSelections_SaveGetRoundTrip(t *testing.T) {
	s := newSelStore(t)
	ctx := t.Context()

	in := sel("s1", "v1")
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := s.Get(ctx, in.Id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.SourceID != "s1" || got.Tag != "v1" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}

	bySource, err := s.GetBySource(ctx, "s1")
	if err != nil {
		t.Fatalf("get by source: %v", err)
	}
	if bySource.Id != in.Id {
		t.Fatalf("get by source id = %q, want %q", bySource.Id, in.Id)
	}
}

func TestSelections_SaveUpsertsPerSource(t *testing.T) {
	s := newSelStore(t)
	ctx := t.Context()

	if err := s.Save(ctx, sel("s1", "v1")); err != nil {
		t.Fatalf("save 1: %v", err)
	}
	// Second save for the same source replaces the tag, not adds a row.
	if err := s.Save(ctx, sel("s1", "v2")); err != nil {
		t.Fatalf("save 2: %v", err)
	}

	got, err := s.GetBySource(ctx, "s1")
	if err != nil {
		t.Fatalf("get by source: %v", err)
	}
	if got.Tag != "v2" {
		t.Fatalf("tag = %q, want v2", got.Tag)
	}

	all, err := s.GetAll(ctx)
	if err != nil {
		t.Fatalf("getall: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 selection after upsert, got %d", len(all))
	}
}

func TestSelections_ListFilterBySource(t *testing.T) {
	s := newSelStore(t)
	ctx := t.Context()
	_ = s.Save(ctx, sel("s1", "v1"))
	_ = s.Save(ctx, sel("s2", "v1"))

	list, err := s.List(ctx, &entities.ProtoSelectionsList{
		SourceID: ptr.Wrap("s1"),
		ListBase: entities.ListBase{Limit: ptr.Wrap(int64(10)), IncludeTotalCount: true},
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if *list.Total != 1 || len(list.Items) != 1 || list.Items[0].SourceID != "s1" {
		t.Fatalf("filtered list = %+v (total %v)", list.Items, list.Total)
	}
}

func TestSelections_DeleteAndDeleteBySource(t *testing.T) {
	s := newSelStore(t)
	ctx := t.Context()

	in := sel("s1", "v1")
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.Delete(ctx, in.Id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Get(ctx, in.Id); !errors.Is(err, errs.ErrProtoSelectionNotFound) {
		t.Fatalf("get after delete: want NotFound, got %v", err)
	}

	_ = s.Save(ctx, sel("s2", "v1"))
	_ = s.Save(ctx, sel("s3", "v1"))
	n, err := s.DeleteBySource(ctx, "s2")
	if err != nil {
		t.Fatalf("delete by source: %v", err)
	}
	if n != 1 {
		t.Fatalf("deleted %d, want 1", n)
	}
	if _, err := s.GetBySource(ctx, "s2"); !errors.Is(err, errs.ErrProtoSelectionNotFound) {
		t.Fatalf("get by source after delete: want NotFound, got %v", err)
	}
}
