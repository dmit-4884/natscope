// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	histbbolt "github.com/dmit-4884/natscope/internal/storages/history/bbolt"
)

func newHistStore(t *testing.T) *histbbolt.Storage {
	t.Helper()
	db, err := bbstore.NewDB(filepath.Join(t.TempDir(), "test.bolt"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s, err := histbbolt.New(t.Context(), db)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	return s
}

func entry(url, stream, subject string) *entities.PublishHistory {
	return entities.PublishHistoryNew(func(h *entities.PublishHistory) {
		h.ConnectionURL = url
		h.Stream = stream
		h.Subject = subject
		h.EncodingType = entities.EncodingTypeJSON
		h.Success = true
	})
}

func TestHistory_SaveAndListFilter(t *testing.T) {
	s := newHistStore(t)
	ctx := t.Context()

	_ = s.Save(ctx, entry("nats://a:4222", "ORDERS", "orders.1"))
	_ = s.Save(ctx, entry("nats://a:4222", "EVENTS", "events.1"))
	_ = s.Save(ctx, entry("nats://b:4222", "ORDERS", "orders.2"))

	// Filter by connection URL.
	byURL, err := s.List(ctx, &entities.PublishHistoryList{
		ConnectionURL: ptr.Wrap("nats://a:4222"),
		ListBase:      entities.ListBase{Limit: ptr.Wrap(int64(10)), IncludeTotalCount: true},
	})
	if err != nil {
		t.Fatalf("list by url: %v", err)
	}
	if *byURL.Total != 2 {
		t.Fatalf("by url total = %d, want 2", *byURL.Total)
	}

	// Filter by connection URL + stream.
	both, err := s.List(ctx, &entities.PublishHistoryList{
		ConnectionURL: ptr.Wrap("nats://a:4222"),
		Stream:        ptr.Wrap("ORDERS"),
		ListBase:      entities.ListBase{Limit: ptr.Wrap(int64(10))},
	})
	if err != nil {
		t.Fatalf("list by url+stream: %v", err)
	}
	if len(both.Items) != 1 || both.Items[0].Subject != "orders.1" {
		t.Fatalf("by url+stream = %+v", both.Items)
	}
}

// TestHistory_ListFilterByConnectionID is a regression test: filtering by
// connection_id must find the entry regardless of how many URLs the saved
// connection has, and must not conflate entries from a different connection
// that happens to share a URL.
func TestHistory_ListFilterByConnectionID(t *testing.T) {
	s := newHistStore(t)
	ctx := t.Context()

	multi := entry("nats://a:4222,nats://b:4222", "MULTI", "multi.1")
	multi.ConnectionID = ptr.Wrap("conn-multi")
	_ = s.Save(ctx, multi)

	shared1 := entry("nats://shared:4222", "SHARED", "shared.1")
	shared1.ConnectionID = ptr.Wrap("conn-shared-1")
	_ = s.Save(ctx, shared1)

	shared2 := entry("nats://shared:4222", "SHARED", "shared.2")
	shared2.ConnectionID = ptr.Wrap("conn-shared-2")
	_ = s.Save(ctx, shared2)

	byID, err := s.List(ctx, &entities.PublishHistoryList{
		ConnectionID: ptr.Wrap("conn-multi"),
		ListBase:     entities.ListBase{Limit: ptr.Wrap(int64(10))},
	})
	if err != nil {
		t.Fatalf("list by connection_id: %v", err)
	}
	if len(byID.Items) != 1 || byID.Items[0].Subject != "multi.1" {
		t.Fatalf("by connection_id = %+v, want [multi.1]", byID.Items)
	}

	sharedFiltered, err := s.List(ctx, &entities.PublishHistoryList{
		ConnectionID: ptr.Wrap("conn-shared-1"),
		ListBase:     entities.ListBase{Limit: ptr.Wrap(int64(10))},
	})
	if err != nil {
		t.Fatalf("list by connection_id (shared url): %v", err)
	}
	if len(sharedFiltered.Items) != 1 || sharedFiltered.Items[0].Subject != "shared.1" {
		t.Fatalf("by connection_id (shared url) = %+v, want [shared.1]", sharedFiltered.Items)
	}
}

func TestHistory_PaginationNewestFirst(t *testing.T) {
	s := newHistStore(t)
	ctx := t.Context()

	// Entries may share a CreatedAt ms; (sort_key, id) keyset still totally orders them.
	for i := 0; i < 5; i++ {
		_ = s.Save(ctx, entry("nats://a:4222", "S", "s.x"))
	}

	page, err := s.List(ctx, &entities.PublishHistoryList{
		ListBase: entities.ListBase{Limit: ptr.Wrap(int64(2)), IncludeTotalCount: true},
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 2 || page.Total == nil || *page.Total != 5 {
		t.Fatalf("page = %d items total=%v", len(page.Items), page.Total)
	}
	if page.NextCursor == nil {
		t.Fatal("expected next cursor")
	}

	// Walk all pages via cursor and ensure we see every record exactly once.
	seen := map[string]bool{}
	cursor := ""
	for {
		p, err := s.List(ctx, &entities.PublishHistoryList{
			ListBase: entities.ListBase{Cursor: cursor, Limit: ptr.Wrap(int64(2))},
		})
		if err != nil {
			t.Fatalf("page walk: %v", err)
		}
		for _, it := range p.Items {
			if seen[it.Id] {
				t.Fatalf("duplicate id across pages: %s", it.Id)
			}
			seen[it.Id] = true
		}
		if p.NextCursor == nil {
			break
		}
		cursor = *p.NextCursor
	}
	if len(seen) != 5 {
		t.Fatalf("saw %d records across pages, want 5", len(seen))
	}
}

func TestHistory_PruneKeepsNewest(t *testing.T) {
	s := newHistStore(t)
	ctx := t.Context()

	base := time.Now().Truncate(time.Second)
	save := func(i int) {
		e := entry("nats://a:4222", "S", "s.x")
		e.CreatedAt = base.Add(time.Duration(i) * time.Second)
		if err := s.Save(ctx, e); err != nil {
			t.Fatalf("save: %v", err)
		}
	}
	for i := range 12 {
		save(i)
	}

	removed, err := s.Prune(ctx, 5, 2)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if removed != 7 {
		t.Fatalf("removed = %d, want 7", removed)
	}

	page, err := s.List(ctx, &entities.PublishHistoryList{ListBase: entities.ListBase{IncludeTotalCount: true}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if page.Total == nil || *page.Total != 5 {
		t.Fatalf("total = %v, want 5", page.Total)
	}
	oldest := page.Items[len(page.Items)-1].CreatedAt
	if oldest.Before(base.Add(7 * time.Second)) {
		t.Fatalf("oldest kept entry %v is older than the newest five", oldest)
	}

	save(12)
	save(13)
	if removed, err = s.Prune(ctx, 5, 2); err != nil || removed != 0 {
		t.Fatalf("prune within slack: removed=%d err=%v, want 0", removed, err)
	}
	save(14)
	if removed, err = s.Prune(ctx, 5, 2); err != nil || removed != 3 {
		t.Fatalf("prune over slack: removed=%d err=%v, want 3", removed, err)
	}
}
