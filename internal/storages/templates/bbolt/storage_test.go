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

	tmplbbolt "github.com/dmit-4884/natscope/internal/storages/templates/bbolt"
)

func newStorage(t *testing.T) *tmplbbolt.Storage {
	t.Helper()
	db, err := bbstore.NewDB(filepath.Join(t.TempDir(), "test.bolt"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s, err := tmplbbolt.New(t.Context(), db)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	return s
}

func sample(name string) *entities.MessageTemplate {
	return entities.MessageTemplateNew(func(tpl *entities.MessageTemplate) {
		tpl.Name = name
		tpl.Subject = "orders.*"
		tpl.MessageType = "acme.Order"
		tpl.Data = `{"id":1}`
		tpl.Headers = map[string]string{"k": "v"}
		tpl.Wildcards = []string{"new"}
	})
}

func TestTemplates_RoundTrip(t *testing.T) {
	s := newStorage(t)
	ctx := t.Context()
	in := sample("t1")
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := s.Get(ctx, in.Id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "t1" || got.Subject != "orders.*" || got.MessageType != "acme.Order" || got.Data != `{"id":1}` {
		t.Fatalf("scalar mismatch: %+v", got)
	}
	if got.Headers["k"] != "v" || len(got.Wildcards) != 1 || got.Wildcards[0] != "new" {
		t.Fatalf("collection mismatch: headers=%v wildcards=%v", got.Headers, got.Wildcards)
	}
}

func TestTemplates_Update(t *testing.T) {
	s := newStorage(t)
	ctx := t.Context()
	in := sample("t1")
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	in.Name = "renamed"
	in.Headers = map[string]string{} // clear
	if err := s.Update(ctx, in); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := s.Get(ctx, in.Id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "renamed" || len(got.Headers) != 0 {
		t.Fatalf("update not applied: %+v", got)
	}
	if err := s.Update(ctx, sample("ghost")); !errors.Is(err, errs.ErrMessageTemplateNotFound) {
		t.Fatalf("update missing: want NotFound, got %v", err)
	}
}

func TestTemplates_DeleteAndDeleteAll(t *testing.T) {
	s := newStorage(t)
	ctx := t.Context()
	a, b := sample("a"), sample("b")
	if err := s.Save(ctx, a); err != nil {
		t.Fatalf("save a: %v", err)
	}
	if err := s.Save(ctx, b); err != nil {
		t.Fatalf("save b: %v", err)
	}
	if err := s.Delete(ctx, a.Id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Get(ctx, a.Id); !errors.Is(err, errs.ErrMessageTemplateNotFound) {
		t.Fatalf("get deleted: want NotFound, got %v", err)
	}
	n, err := s.DeleteAll(ctx)
	if err != nil {
		t.Fatalf("delete all: %v", err)
	}
	if n != 1 {
		t.Fatalf("delete all count: want 1, got %d", n)
	}
}

func TestTemplates_ListPagination(t *testing.T) {
	s := newStorage(t)
	ctx := t.Context()
	for _, n := range []string{"a", "b", "c"} {
		if err := s.Save(ctx, sample(n)); err != nil {
			t.Fatalf("save %s: %v", n, err)
		}
	}
	page, err := s.List(ctx, &entities.MessageTemplatesList{
		ListBase: entities.ListBase{Limit: ptr.Wrap(int64(2)), IncludeTotalCount: true},
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 2 || page.Total == nil || *page.Total != 3 || page.NextCursor == nil {
		t.Fatalf("page1 = %d items total=%v cursor=%v", len(page.Items), page.Total, page.NextCursor)
	}
}
