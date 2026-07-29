// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt_test

import (
	"path/filepath"
	"testing"

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
