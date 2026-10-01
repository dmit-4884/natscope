// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	fsbbolt "github.com/dmit-4884/natscope/internal/storages/proto/filesets/bbolt"
)

func newStore(t *testing.T) *fsbbolt.Storage {
	t.Helper()
	db, err := bbstore.NewDB(filepath.Join(t.TempDir(), "test.bolt"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s, err := fsbbolt.New(t.Context(), db)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	return s
}

func fileSet(sourceID, revision, content string) *entities.ProtoFileSet {
	return entities.ProtoFileSetNew(func(v *entities.ProtoFileSet) {
		v.SourceID = sourceID
		v.Revision = revision
		v.Files = []entities.ProtoFileEntry{{Path: "a/x.proto", Content: content, Size: int64(len(content))}}
		v.Configs = []entities.ProtoFileEntry{{Path: "buf.yaml", Content: "version: v2", Size: 11}}
	})
}

func TestFileSets_SaveGetRoundTrip(t *testing.T) {
	s := newStore(t)

	if err := s.Save(t.Context(), fileSet("s1", "abc", `syntax = "proto3";`)); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := s.GetBySourceRevision(t.Context(), "s1", "abc")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.SourceID != "s1" || got.Revision != "abc" {
		t.Fatalf("scalar mismatch: %+v", got)
	}
	if len(got.Files) != 1 || got.Files[0].Path != "a/x.proto" || got.Files[0].Size != 18 {
		t.Fatalf("files mismatch: %+v", got.Files)
	}
	if len(got.Configs) != 1 || got.Configs[0].Path != "buf.yaml" {
		t.Fatalf("configs mismatch: %+v", got.Configs)
	}
}

func TestFileSets_SaveReplacesSameRevision(t *testing.T) {
	s := newStore(t)

	if err := s.Save(t.Context(), fileSet("s1", "upload", "one")); err != nil {
		t.Fatalf("save first: %v", err)
	}
	if err := s.Save(t.Context(), fileSet("s1", "upload", "two")); err != nil {
		t.Fatalf("save second: %v", err)
	}
	got, err := s.GetBySourceRevision(t.Context(), "s1", "upload")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Files[0].Content != "two" {
		t.Fatalf("want replaced content, got %q", got.Files[0].Content)
	}
}

func TestFileSets_RevisionsAndSourcesAreSeparate(t *testing.T) {
	s := newStore(t)

	for _, fs := range []*entities.ProtoFileSet{fileSet("s1", "a", "s1a"), fileSet("s1", "b", "s1b"), fileSet("s2", "a", "s2a")} {
		if err := s.Save(t.Context(), fs); err != nil {
			t.Fatalf("save: %v", err)
		}
	}
	got, err := s.GetBySourceRevision(t.Context(), "s1", "b")
	if err != nil || got.Files[0].Content != "s1b" {
		t.Fatalf("s1/b: %v %+v", err, got)
	}
	got, err = s.GetBySourceRevision(t.Context(), "s2", "a")
	if err != nil || got.Files[0].Content != "s2a" {
		t.Fatalf("s2/a: %v %+v", err, got)
	}
}

func TestFileSets_NotFoundAndDeleteBySource(t *testing.T) {
	s := newStore(t)

	if _, err := s.GetBySourceRevision(t.Context(), "s1", "missing"); !errors.Is(err, errs.ErrProtoFileSetNotFound) {
		t.Fatalf("want ErrProtoFileSetNotFound, got %v", err)
	}
	for _, rev := range []string{"a", "b"} {
		if err := s.Save(t.Context(), fileSet("s1", rev, rev)); err != nil {
			t.Fatalf("save: %v", err)
		}
	}
	if err := s.Save(t.Context(), fileSet("s2", "a", "keep")); err != nil {
		t.Fatalf("save: %v", err)
	}
	n, err := s.DeleteBySource(t.Context(), "s1")
	if err != nil || n != 2 {
		t.Fatalf("delete: n=%d err=%v", n, err)
	}
	if _, err := s.GetBySourceRevision(t.Context(), "s1", "a"); !errors.Is(err, errs.ErrProtoFileSetNotFound) {
		t.Fatalf("s1 should be gone, got %v", err)
	}
	if _, err := s.GetBySourceRevision(t.Context(), "s2", "a"); err != nil {
		t.Fatalf("s2 should survive: %v", err)
	}
}
