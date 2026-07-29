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

	verbbolt "github.com/dmit-4884/natscope/internal/storages/proto/versions/bbolt"
)

func newVerStore(t *testing.T) *verbbolt.Storage {
	t.Helper()
	db, err := bbstore.NewDB(filepath.Join(t.TempDir(), "test.bolt"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s, err := verbbolt.New(t.Context(), db)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	return s
}

func ver(sourceID, tag string) *entities.ProtoVersion {
	return entities.ProtoVersionNew(func(v *entities.ProtoVersion) {
		v.SourceID = sourceID
		v.Tag = tag
		v.Files = []entities.ProtoFileEntry{{Path: "a/x.proto", Content: `syntax = "proto3";`, Size: 18}}
		v.Configs = []entities.ProtoFileEntry{{Path: "buf.yaml", Content: "version: v2", Size: 11}}
	})
}

func TestVersions_SaveGetRoundTrip(t *testing.T) {
	s := newVerStore(t)
	ctx := t.Context()

	in := ver("s1", "v1")
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := s.Get(ctx, in.Id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.SourceID != "s1" || got.Tag != "v1" {
		t.Fatalf("scalar mismatch: %+v", got)
	}
	if len(got.Files) != 1 || got.Files[0].Path != "a/x.proto" || got.Files[0].Size != 18 {
		t.Fatalf("files round-trip mismatch: %+v", got.Files)
	}
	if len(got.Configs) != 1 || got.Configs[0].Path != "buf.yaml" {
		t.Fatalf("configs round-trip mismatch: %+v", got.Configs)
	}

	byTag, err := s.GetBySourceAndTag(ctx, "s1", "v1")
	if err != nil {
		t.Fatalf("get by source+tag: %v", err)
	}
	if byTag.Id != in.Id {
		t.Fatalf("get by source+tag id = %q, want %q", byTag.Id, in.Id)
	}
}

func TestVersions_SaveDuplicateRejected(t *testing.T) {
	s := newVerStore(t)
	ctx := t.Context()

	if err := s.Save(ctx, ver("s1", "v1")); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.Save(ctx, ver("s1", "v1")); !errors.Is(err, errs.ErrProtoVersionAlreadyExists) {
		t.Fatalf("duplicate save: want AlreadyExists, got %v", err)
	}
}

func TestVersions_ListTagsAndExists(t *testing.T) {
	s := newVerStore(t)
	ctx := t.Context()
	_ = s.Save(ctx, ver("s1", "v1"))
	_ = s.Save(ctx, ver("s1", "v2"))
	_ = s.Save(ctx, ver("s2", "v1"))

	list, err := s.List(ctx, &entities.ProtoVersionsList{
		SourceID: "s1",
		ListBase: entities.ListBase{Limit: ptr.Wrap(int64(10)), IncludeTotalCount: true},
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if *list.Total != 2 {
		t.Fatalf("total for s1 = %d, want 2", *list.Total)
	}

	tags, err := s.GetTagsBySource(ctx, "s1")
	if err != nil {
		t.Fatalf("get tags: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("tags = %v, want 2", tags)
	}

	ok, err := s.ExistsBySourceAndTag(ctx, "s1", "v2")
	if err != nil || !ok {
		t.Fatalf("exists by source+tag = %v, err %v", ok, err)
	}
	if ok, _ = s.ExistsBySourceAndTag(ctx, "s1", "v9"); ok {
		t.Fatalf("unknown tag should not exist")
	}
}

func TestVersions_DeleteAndDeleteBySource(t *testing.T) {
	s := newVerStore(t)
	ctx := t.Context()

	in := ver("s1", "v1")
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	exists, err := s.Exists(ctx, in.Id)
	if err != nil || !exists {
		t.Fatalf("exists = %v, err %v", exists, err)
	}
	if err := s.Delete(ctx, in.Id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Get(ctx, in.Id); !errors.Is(err, errs.ErrProtoVersionNotFound) {
		t.Fatalf("get after delete: want NotFound, got %v", err)
	}

	_ = s.Save(ctx, ver("s2", "v1"))
	_ = s.Save(ctx, ver("s2", "v2"))
	n, err := s.DeleteBySource(ctx, "s2")
	if err != nil {
		t.Fatalf("delete by source: %v", err)
	}
	if n != 2 {
		t.Fatalf("deleted %d, want 2", n)
	}
}
