// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt_test

import (
	"path/filepath"
	"testing"

	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	mappingsbbolt "github.com/dmit-4884/natscope/internal/storages/mappings/bbolt"
)

func newStorage(t *testing.T) *mappingsbbolt.Storage {
	t.Helper()
	db, err := bbstore.NewDB(filepath.Join(t.TempDir(), "test.bolt"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s, err := mappingsbbolt.New(t.Context(), db)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	return s
}

func TestMappings_SaveGetRoundTrip(t *testing.T) {
	s := newStorage(t)
	ctx := t.Context()

	in := entities.SubjectMappingNew(func(m *entities.SubjectMapping) {
		m.Pattern = "orders.>"
		m.MessageType = "api.v1.OrderEvent"
		m.SourceID = "src-1"
		m.PinnedTag = ptr.Wrap("v1.2.3")
	})
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := s.Get(ctx, in.Id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Id != in.Id || got.Pattern != in.Pattern || got.MessageType != in.MessageType || got.SourceID != in.SourceID {
		t.Fatalf("scalar mismatch: %+v vs %+v", got, in)
	}
	if got.Etag != in.Etag {
		t.Fatalf("etag mismatch: %q vs %q", got.Etag, in.Etag)
	}
	if got.PinnedTag == nil || *got.PinnedTag != "v1.2.3" {
		t.Fatalf("pinned tag mismatch: %v", got.PinnedTag)
	}
	if got.PinnedFingerprint != nil {
		t.Fatalf("expected nil pinned fingerprint, got %v", *got.PinnedFingerprint)
	}
	// time.Time <-> ms round-trip is lossless at ms precision.
	if !got.CreatedAt.Equal(in.CreatedAt.Truncate(0)) && got.CreatedAt.UnixMilli() != in.CreatedAt.UnixMilli() {
		t.Fatalf("created_at mismatch: %v vs %v", got.CreatedAt, in.CreatedAt)
	}
	if got.DeletedAt != nil {
		t.Fatalf("expected nil deleted_at, got %v", *got.DeletedAt)
	}
}

func TestMappings_DuplicatePatternSource(t *testing.T) {
	s := newStorage(t)
	ctx := t.Context()

	first := entities.SubjectMappingNew(func(m *entities.SubjectMapping) {
		m.Pattern = "a.*"
		m.MessageType = "T1"
		m.SourceID = "src"
	})
	if err := s.Save(ctx, first); err != nil {
		t.Fatalf("save first: %v", err)
	}
	second := entities.SubjectMappingNew(func(m *entities.SubjectMapping) {
		m.Pattern = "a.*"
		m.MessageType = "T2"
		m.SourceID = "src"
	})
	if err := s.Save(ctx, second); err == nil {
		t.Fatal("expected duplicate pattern/source error, got nil")
	}
}

func TestMappings_ListPagination(t *testing.T) {
	s := newStorage(t)
	ctx := t.Context()

	const n = 5
	for i := 0; i < n; i++ {
		m := entities.SubjectMappingNew(func(m *entities.SubjectMapping) {
			m.Pattern = "p" + string(rune('a'+i)) + ".>"
			m.MessageType = "T"
			m.SourceID = "src"
		})
		if err := s.Save(ctx, m); err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}

	limit := int64(2)
	page, err := s.List(ctx, &entities.SubjectMappingsList{
		ListBase: entities.ListBase{Limit: &limit, IncludeTotalCount: true},
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("want 2 items, got %d", len(page.Items))
	}
	if page.Total == nil || *page.Total != n {
		t.Fatalf("want total %d, got %v", n, page.Total)
	}
	if page.NextCursor == nil {
		t.Fatal("expected next cursor")
	}

	// Walk all pages; expect every id exactly once.
	seen := map[string]bool{}
	cursor := ""
	for {
		p, err := s.List(ctx, &entities.SubjectMappingsList{
			ListBase: entities.ListBase{Cursor: cursor, Limit: &limit},
		})
		if err != nil {
			t.Fatalf("list page: %v", err)
		}
		for _, m := range p.Items {
			if seen[m.Id] {
				t.Fatalf("duplicate id across pages: %s", m.Id)
			}
			seen[m.Id] = true
		}
		if p.NextCursor == nil {
			break
		}
		cursor = *p.NextCursor
	}
	if len(seen) != n {
		t.Fatalf("want %d distinct ids, got %d", n, len(seen))
	}
}

func TestMappings_BulkSaveMergesByPatternSource(t *testing.T) {
	s := newStorage(t)
	ctx := t.Context()
	first := entities.SubjectMappingNew(func(m *entities.SubjectMapping) {
		m.Pattern = "a.>"
		m.SourceID = "src1"
		m.MessageType = "TypeA"
	})
	if err := s.Save(ctx, first); err != nil {
		t.Fatalf("save: %v", err)
	}
	// A fresh-id mapping for the SAME (pattern, source) must update in place.
	dup := entities.SubjectMappingNew(func(m *entities.SubjectMapping) {
		m.Pattern = "a.>"
		m.SourceID = "src1"
		m.MessageType = "TypeA-v2"
	})
	add := entities.SubjectMappingNew(func(m *entities.SubjectMapping) {
		m.Pattern = "b.>"
		m.SourceID = "src1"
		m.MessageType = "TypeB"
	})
	if _, err := s.BulkSave(ctx, entities.SubjectMappings{dup, add}); err != nil {
		t.Fatalf("bulk: %v", err)
	}
	all, err := s.ListAll(ctx)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("want 2 mappings, got %d", len(all))
	}
	got, err := s.Get(ctx, first.Id)
	if err != nil {
		t.Fatalf("original row gone: %v", err)
	}
	if got.MessageType != "TypeA-v2" {
		t.Fatalf("merge did not update type: %+v", got)
	}
}

// TestMappings_BulkSaveDeletesAbsent locks in BulkSave's full-replace contract:
// an absent (pattern, sourceId) pair must be deleted and counts must reflect it.
func TestMappings_BulkSaveDeletesAbsent(t *testing.T) {
	s := newStorage(t)
	ctx := t.Context()

	keep := entities.SubjectMappingNew(func(m *entities.SubjectMapping) {
		m.Pattern = "keep.>"
		m.SourceID = "src1"
		m.MessageType = "Keep"
	})
	drop := entities.SubjectMappingNew(func(m *entities.SubjectMapping) {
		m.Pattern = "drop.>"
		m.SourceID = "src1"
		m.MessageType = "Drop"
	})
	if err := s.Save(ctx, keep); err != nil {
		t.Fatalf("save keep: %v", err)
	}
	if err := s.Save(ctx, drop); err != nil {
		t.Fatalf("save drop: %v", err)
	}

	replaceKeep := entities.SubjectMappingNew(func(m *entities.SubjectMapping) {
		m.Pattern = "keep.>"
		m.SourceID = "src1"
		m.MessageType = "Keep-v2"
	})
	newOne := entities.SubjectMappingNew(func(m *entities.SubjectMapping) {
		m.Pattern = "new.>"
		m.SourceID = "src1"
		m.MessageType = "New"
	})
	result, err := s.BulkSave(ctx, entities.SubjectMappings{replaceKeep, newOne})
	if err != nil {
		t.Fatalf("bulk: %v", err)
	}
	if result.Created != 1 || result.Updated != 1 || result.Deleted != 1 {
		t.Fatalf("want created=1 updated=1 deleted=1, got %+v", result)
	}

	all, err := s.ListAll(ctx)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("want 2 mappings after replace, got %d", len(all))
	}
	for _, m := range all {
		if m.Pattern == "drop.>" {
			t.Fatalf("drop.> must have been deleted by the replace, found: %+v", m)
		}
	}
}
