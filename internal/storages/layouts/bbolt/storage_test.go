// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore/bbstoretest"

	layoutsbbolt "github.com/dmit-4884/natscope/internal/storages/layouts/bbolt"
)

func newStorage(t *testing.T) *layoutsbbolt.Storage {
	t.Helper()
	s, err := layoutsbbolt.New(t.Context(), bbstoretest.NewMemoryDB(t))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	return s
}

func TestLayouts_GetMissing(t *testing.T) {
	s := newStorage(t)
	if _, err := s.Get(t.Context(), "conn-1"); !errors.Is(err, errs.ErrSidebarLayoutNotFound) {
		t.Fatalf("get missing: want ErrSidebarLayoutNotFound, got %v", err)
	}
}

func TestLayouts_UpdateRoundTrip(t *testing.T) {
	s := newStorage(t)
	ctx := t.Context()
	stamp := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

	_, err := s.Update(ctx, "conn-1", func(l *entities.SidebarLayout) {
		l.Streams = entities.SectionLayout{Pinned: []string{"ORDERS"}, Order: []string{"EVENTS", "AUDIT"}}
		l.KV = entities.SectionLayout{Pinned: []string{"config"}}
		l.UpdatedAt = stamp
	})
	if err != nil {
		t.Fatalf("first update: %v", err)
	}

	_, err = s.Update(ctx, "conn-1", func(l *entities.SidebarLayout) {
		if !slices.Equal(l.Streams.Pinned, []string{"ORDERS"}) {
			t.Errorf("mutate saw streams.pinned %v, want [ORDERS]", l.Streams.Pinned)
		}
		l.Objects = entities.SectionLayout{Order: []string{"assets"}}
	})
	if err != nil {
		t.Fatalf("second update: %v", err)
	}

	got, err := s.Get(ctx, "conn-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ConnectionID != "conn-1" {
		t.Errorf("connection id = %q, want conn-1", got.ConnectionID)
	}
	if !slices.Equal(got.Streams.Order, []string{"EVENTS", "AUDIT"}) {
		t.Errorf("streams.order = %v, want [EVENTS AUDIT]", got.Streams.Order)
	}
	if !slices.Equal(got.KV.Pinned, []string{"config"}) {
		t.Errorf("kv.pinned = %v, want [config]", got.KV.Pinned)
	}
	if !slices.Equal(got.Objects.Order, []string{"assets"}) {
		t.Errorf("objects.order = %v, want [assets]", got.Objects.Order)
	}
	if !got.UpdatedAt.Equal(stamp) {
		t.Errorf("updated at = %v, want %v", got.UpdatedAt, stamp)
	}
}

func TestLayouts_ConnectionsAreIsolated(t *testing.T) {
	s := newStorage(t)
	ctx := t.Context()

	if _, err := s.Update(ctx, "conn-1", func(l *entities.SidebarLayout) {
		l.Streams.Pinned = []string{"ORDERS"}
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if _, err := s.Get(ctx, "conn-2"); !errors.Is(err, errs.ErrSidebarLayoutNotFound) {
		t.Fatalf("other connection: want ErrSidebarLayoutNotFound, got %v", err)
	}
}

func TestLayouts_Delete(t *testing.T) {
	s := newStorage(t)
	ctx := t.Context()

	if _, err := s.Update(ctx, "conn-1", func(l *entities.SidebarLayout) {
		l.Streams.Pinned = []string{"ORDERS"}
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := s.Delete(ctx, "conn-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Get(ctx, "conn-1"); !errors.Is(err, errs.ErrSidebarLayoutNotFound) {
		t.Fatalf("get after delete: want ErrSidebarLayoutNotFound, got %v", err)
	}
	if err := s.Delete(ctx, "conn-1"); !errors.Is(err, errs.ErrSidebarLayoutNotFound) {
		t.Fatalf("second delete: want ErrSidebarLayoutNotFound, got %v", err)
	}
}
