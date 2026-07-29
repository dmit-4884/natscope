// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbstore_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/pkg/bbstore"
)

var errWidgetNotFound = errors.New("widget not found")

// widget is a sample persistence model: a base plus a nested sub-struct, an
// optional scalar and a slice, all stored inside the JSON doc.
type widget struct {
	bbstore.Base

	Name  string      `json:"name"`
	Color *string     `json:"color,omitempty"`
	Tags  []string    `json:"tags,omitempty"`
	Spec  *widgetSpec `json:"spec,omitempty"`
}

type widgetSpec struct {
	Size int    `json:"size"`
	Note string `json:"note"`
}

func newDB(t *testing.T) *bbstore.DB {
	t.Helper()
	db, err := bbstore.NewDB(filepath.Join(t.TempDir(), "test.bolt"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func newStore(t *testing.T) *bbstore.Store[widget, *widget] {
	t.Helper()
	s, err := bbstore.Open[widget, *widget](t.Context(), newDB(t), bbstore.Spec{
		Bucket:   "widgets",
		NotFound: errWidgetNotFound,
		Indexes:  []bbstore.Index{{Path: "name"}},
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return s
}

func newUniqueStore(t *testing.T) *bbstore.Store[widget, *widget] {
	t.Helper()
	s, err := bbstore.Open[widget, *widget](t.Context(), newDB(t), bbstore.Spec{
		Bucket:   "widgets",
		NotFound: errWidgetNotFound,
		Indexes:  []bbstore.Index{{Path: "name", Unique: true, ActiveOnly: true}},
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return s
}

func newStoreSortedByUpdated(t *testing.T) *bbstore.Store[widget, *widget] {
	t.Helper()
	s, err := bbstore.Open[widget, *widget](t.Context(), newDB(t), bbstore.Spec{
		Bucket:   "widgets",
		NotFound: errWidgetNotFound,
		SortBy:   bbstore.SortUpdated,
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return s
}

func mk(id, name string, createdAt int64) *widget {
	return &widget{
		Base: bbstore.Base{ID: id, Etag: "e-" + id, CreatedAt: createdAt, UpdatedAt: createdAt},
		Name: name,
	}
}

func TestSaveGetRoundTrip(t *testing.T) {
	s := newStore(t)
	ctx := t.Context()

	in := mk("w1", "alpha", 100)
	in.Color = ptr.Wrap("red")
	in.Tags = []string{"a", "b"}
	in.Spec = &widgetSpec{Size: 7, Note: "hi"}
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := s.Get(ctx, "w1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "alpha" || got.Etag != "e-w1" || got.CreatedAt != 100 {
		t.Fatalf("base mismatch: %+v", got)
	}
	if got.Color == nil || *got.Color != "red" {
		t.Fatalf("color mismatch: %+v", got.Color)
	}
	if len(got.Tags) != 2 || got.Tags[1] != "b" {
		t.Fatalf("tags mismatch: %+v", got.Tags)
	}
	if got.Spec == nil || got.Spec.Size != 7 || got.Spec.Note != "hi" {
		t.Fatalf("spec mismatch: %+v", got.Spec)
	}
}

func TestGetMissing(t *testing.T) {
	s := newStore(t)
	if _, err := s.Get(t.Context(), "nope"); !errors.Is(err, errWidgetNotFound) {
		t.Fatalf("want NotFound, got %v", err)
	}
}

func TestUpdate(t *testing.T) {
	s := newStore(t)
	ctx := t.Context()
	in := mk("w1", "alpha", 100)
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	in.Name = "beta"
	in.UpdatedAt = 200
	if err := s.Update(ctx, in); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := s.Get(ctx, "w1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "beta" || got.UpdatedAt != 200 {
		t.Fatalf("update not applied: %+v", got)
	}
	if err := s.Update(ctx, mk("ghost", "x", 1)); !errors.Is(err, errWidgetNotFound) {
		t.Fatalf("update missing: want NotFound, got %v", err)
	}
}

func TestSoftDelete(t *testing.T) {
	s := newUniqueStore(t)
	ctx := t.Context()
	in := mk("w1", "alpha", 100)
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.SoftDelete(ctx, "w1", 500); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	if _, err := s.Get(ctx, "w1"); !errors.Is(err, errWidgetNotFound) {
		t.Fatalf("active get of deleted: want NotFound, got %v", err)
	}
	got, err := s.Get(ctx, "w1", true)
	if err != nil {
		t.Fatalf("get includeDeleted: %v", err)
	}
	if got.DeletedAt == nil || *got.DeletedAt != 500 || got.UpdatedAt != 500 {
		t.Fatalf("doc not synced on soft delete: %+v", got)
	}
	// the name frees up once soft-deleted (active-only uniqueness)
	n, err := s.CountActiveBy(ctx, "$.name", "alpha", "")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("want 0 active named alpha, got %d", n)
	}
}

func TestExists(t *testing.T) {
	s := newStore(t)
	ctx := t.Context()
	if err := s.Save(ctx, mk("w1", "a", 1)); err != nil {
		t.Fatalf("save: %v", err)
	}
	ok, err := s.Exists(ctx, "w1")
	if err != nil || !ok {
		t.Fatalf("exists: %v %v", ok, err)
	}
	ok, err = s.Exists(ctx, "missing")
	if err != nil || ok {
		t.Fatalf("exists missing: %v %v", ok, err)
	}
}

func TestDelete(t *testing.T) {
	s := newStore(t)
	ctx := t.Context()
	if err := s.Save(ctx, mk("w1", "a", 1)); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.Delete(ctx, "w1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Get(ctx, "w1", true); !errors.Is(err, errWidgetNotFound) {
		t.Fatalf("get after delete: want NotFound, got %v", err)
	}
	if err := s.Delete(ctx, "w1"); !errors.Is(err, errWidgetNotFound) {
		t.Fatalf("delete missing: want NotFound, got %v", err)
	}
}

func TestListPaginationDesc(t *testing.T) {
	s := newStore(t)
	ctx := t.Context()
	for i, n := range []string{"a", "b", "c", "d", "e"} {
		if err := s.Save(ctx, mk(n, n, int64(i+1))); err != nil {
			t.Fatalf("save %s: %v", n, err)
		}
	}

	page1, next, total, err := s.List(ctx, "", 2, true)
	if err != nil {
		t.Fatalf("list page1: %v", err)
	}
	if len(page1) != 2 || next == "" || total == nil || *total != 5 {
		t.Fatalf("page1: items=%d next=%q total=%v", len(page1), next, total)
	}
	// newest first: created_at 5 (e) then 4 (d)
	if page1[0].ID != "e" || page1[1].ID != "d" {
		t.Fatalf("page1 order: %s,%s", page1[0].ID, page1[1].ID)
	}

	page2, _, _, err := s.List(ctx, next, 2, false)
	if err != nil {
		t.Fatalf("list page2: %v", err)
	}
	if len(page2) != 2 || page2[0].ID != "c" || page2[1].ID != "b" {
		t.Fatalf("page2: %v", page2)
	}
}

func TestGetByAndCountActiveBy(t *testing.T) {
	s := newStore(t)
	ctx := t.Context()
	if err := s.Save(ctx, mk("w1", "alpha", 1)); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := s.GetBy(ctx, "$.name", "alpha")
	if err != nil {
		t.Fatalf("getBy: %v", err)
	}
	if got.ID != "w1" {
		t.Fatalf("getBy wrong id: %s", got.ID)
	}
	if _, err := s.GetBy(ctx, "$.name", "absent"); !errors.Is(err, errWidgetNotFound) {
		t.Fatalf("getBy missing: want NotFound, got %v", err)
	}

	n, err := s.CountActiveBy(ctx, "$.name", "alpha", "")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("count alpha: want 1, got %d", n)
	}
	// excludeID skips the row itself (rename-to-same-name is allowed)
	n, err = s.CountActiveBy(ctx, "$.name", "alpha", "w1")
	if err != nil {
		t.Fatalf("count exclude: %v", err)
	}
	if n != 0 {
		t.Fatalf("count exclude self: want 0, got %d", n)
	}
}

func TestSortByUpdatedAt(t *testing.T) {
	s := newStoreSortedByUpdated(t)
	ctx := t.Context()
	// created_at ascending (a<b<c) but updated_at descending (a>b>c)
	a := mk("a", "a", 1)
	a.UpdatedAt = 30
	b := mk("b", "b", 2)
	b.UpdatedAt = 20
	c := mk("c", "c", 3)
	c.UpdatedAt = 10
	for _, w := range []*widget{a, b, c} {
		if err := s.Save(ctx, w); err != nil {
			t.Fatalf("save %s: %v", w.ID, err)
		}
	}
	page, _, _, err := s.List(ctx, "", 10, false)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if page[0].ID != "a" || page[1].ID != "b" || page[2].ID != "c" {
		t.Fatalf("updated_at order: %s,%s,%s", page[0].ID, page[1].ID, page[2].ID)
	}
}

func TestUpsert(t *testing.T) {
	s := newStore(t)
	ctx := t.Context()
	w := mk("w1", "alpha", 1)
	if err := s.Upsert(ctx, w); err != nil {
		t.Fatalf("upsert insert: %v", err)
	}
	w.Name = "beta"
	w.UpdatedAt = 2
	if err := s.Upsert(ctx, w); err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	got, err := s.Get(ctx, "w1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "beta" {
		t.Fatalf("upsert did not update: %+v", got)
	}
}

func TestDeleteAll(t *testing.T) {
	s := newStore(t)
	ctx := t.Context()
	for _, n := range []string{"a", "b", "c"} {
		if err := s.Save(ctx, mk(n, n, 1)); err != nil {
			t.Fatalf("save %s: %v", n, err)
		}
	}
	n, err := s.DeleteAll(ctx)
	if err != nil {
		t.Fatalf("delete all: %v", err)
	}
	if n != 3 {
		t.Fatalf("deleted count: want 3, got %d", n)
	}
	items, _, _, _ := s.List(ctx, "", 10, false)
	if len(items) != 0 {
		t.Fatalf("not empty after delete all: %d", len(items))
	}
}

func TestListByListAllDeleteBy(t *testing.T) {
	s := newStore(t)
	ctx := t.Context()
	mkc := func(id, color string) *widget {
		w := mk(id, id, 1)
		w.Color = ptr.Wrap(color)
		return w
	}
	for _, w := range []*widget{mkc("a", "red"), mkc("b", "red"), mkc("c", "blue")} {
		if err := s.Save(ctx, w); err != nil {
			t.Fatalf("save %s: %v", w.ID, err)
		}
	}
	reds, err := s.ListBy(ctx, "$.color", "red")
	if err != nil {
		t.Fatalf("list by: %v", err)
	}
	if len(reds) != 2 {
		t.Fatalf("listBy red: want 2, got %d", len(reds))
	}
	all, err := s.ListAll(ctx)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("listAll: want 3, got %d", len(all))
	}
	del, err := s.DeleteBy(ctx, "$.color", "red")
	if err != nil {
		t.Fatalf("delete by: %v", err)
	}
	if del != 2 {
		t.Fatalf("deleteBy: want 2, got %d", del)
	}
	all, _ = s.ListAll(ctx)
	if len(all) != 1 {
		t.Fatalf("after deleteBy: want 1, got %d", len(all))
	}
}

func TestListFiltered(t *testing.T) {
	s := newStore(t)
	ctx := t.Context()
	mkc := func(id, color string, created int64) *widget {
		w := mk(id, id, created)
		w.Color = ptr.Wrap(color)
		return w
	}
	for _, w := range []*widget{mkc("a", "red", 1), mkc("b", "red", 2), mkc("c", "blue", 3)} {
		if err := s.Save(ctx, w); err != nil {
			t.Fatalf("save %s: %v", w.ID, err)
		}
	}
	page, next, total, err := s.ListFiltered(ctx, []bbstore.Filter{{Path: "$.color", Val: "red"}}, "", 10, true)
	if err != nil {
		t.Fatalf("list filtered: %v", err)
	}
	if len(page) != 2 || total == nil || *total != 2 || next != "" {
		t.Fatalf("filtered: items=%d total=%v next=%q", len(page), total, next)
	}
	if page[0].ID != "b" || page[1].ID != "a" {
		t.Fatalf("filtered order: %s,%s", page[0].ID, page[1].ID)
	}
}

func TestUniqueViolation(t *testing.T) {
	s := newUniqueStore(t)
	ctx := t.Context()
	if err := s.Save(ctx, mk("w1", "alpha", 1)); err != nil {
		t.Fatalf("save first: %v", err)
	}
	// a second active doc with the same name is rejected
	err := s.Save(ctx, mk("w2", "alpha", 2))
	var uv *bbstore.ErrUniqueViolation
	if !errors.As(err, &uv) {
		t.Fatalf("save dup name: want ErrUniqueViolation, got %v", err)
	}
	if uv.Path != "name" {
		t.Fatalf("violation path: want name, got %q", uv.Path)
	}
	// a different name is fine
	if err := s.Save(ctx, mk("w2", "beta", 2)); err != nil {
		t.Fatalf("save distinct name: %v", err)
	}
	// renaming w2 onto an existing active name is rejected
	w2 := mk("w2", "alpha", 2)
	if err := s.Update(ctx, w2); !errors.As(err, &uv) {
		t.Fatalf("update onto taken name: want ErrUniqueViolation, got %v", err)
	}
	// updating a doc keeping its own name is allowed (exclude self)
	w1 := mk("w1", "alpha", 3)
	if err := s.Update(ctx, w1); err != nil {
		t.Fatalf("update keeping own name: %v", err)
	}
}

func TestWithTransactionRollback(t *testing.T) {
	s := newStore(t)
	ctx := t.Context()
	sentinel := errors.New("boom")
	err := s.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.Save(txCtx, mk("w1", "alpha", 1)); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("want sentinel, got %v", err)
	}
	// the failed transaction rolled back: nothing was written
	if _, err := s.Get(ctx, "w1"); !errors.Is(err, errWidgetNotFound) {
		t.Fatalf("rollback failed, doc present: %v", err)
	}
}

func TestWithTransactionCommit(t *testing.T) {
	s := newStore(t)
	ctx := t.Context()
	err := s.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.Save(txCtx, mk("w1", "alpha", 1)); err != nil {
			return err
		}
		return s.Save(txCtx, mk("w2", "beta", 2))
	})
	if err != nil {
		t.Fatalf("tx: %v", err)
	}
	for _, id := range []string{"w1", "w2"} {
		if _, err := s.Get(ctx, id); err != nil {
			t.Fatalf("get %s after commit: %v", id, err)
		}
	}
}
