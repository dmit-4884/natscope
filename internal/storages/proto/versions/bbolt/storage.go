// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package bbolt is the bbolt (doc-model) implementation of proto-version
// storage. Files and configs are nested JSON; versions are cached data, so
// deletes are hard.
package bbolt

import (
	"context"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/core/types/ptr"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	storage "github.com/dmit-4884/natscope/internal/storages/proto/versions"
)

// Storage is the doc-model bbolt proto-version store.
type Storage struct {
	store *bbstore.Store[versionDoc, *versionDoc]
}

// New opens the proto_versions bucket (creating it if needed) and returns the
// store.
func New(ctx context.Context, db *bbstore.DB) (*Storage, error) {
	store, err := bbstore.Open[versionDoc, *versionDoc](ctx, db, bbstore.Spec{
		Bucket:   "proto_versions",
		NotFound: errs.ErrProtoVersionNotFound,
	})
	if err != nil {
		return nil, err
	}
	return &Storage{store: store}, nil
}

// Save inserts a version, rejecting a duplicate (source, tag).
func (s *Storage) Save(ctx context.Context, in *entities.ProtoVersion) error {
	return s.store.WithTransaction(ctx, func(ctx context.Context) error {
		existing, err := s.docBySourceTag(ctx, in.SourceID, in.Tag)
		if err != nil {
			return err
		}
		if existing != nil {
			return errs.ErrProtoVersionAlreadyExists
		}
		return s.store.Save(ctx, converter.Convert(in, &versionDoc{}, bbstore.Opts()...))
	})
}

func (s *Storage) Get(ctx context.Context, id string) (*entities.ProtoVersion, error) {
	d, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return converter.Convert(d, &entities.ProtoVersion{}, bbstore.Opts()...), nil
}

func (s *Storage) GetBySourceAndTag(ctx context.Context, sourceID, tag string) (*entities.ProtoVersion, error) {
	d, err := s.docBySourceTag(ctx, sourceID, tag)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, errs.ErrProtoVersionNotFound
	}
	return converter.Convert(d, &entities.ProtoVersion{}, bbstore.Opts()...), nil
}

func (s *Storage) List(
	ctx context.Context,
	in *entities.ProtoVersionsList,
) (*entities.List[entities.ProtoVersions], error) {
	docs, next, total, err := s.store.ListFiltered(ctx,
		[]bbstore.Filter{{Path: "$.sourceId", Val: in.SourceID}},
		in.Cursor, in.GetLimit(), in.IncludeTotalCount)
	if err != nil {
		return nil, err
	}
	out := &entities.List[entities.ProtoVersions]{
		Items: bbstore.ToEntities[entities.ProtoVersion](docs),
		Total: total,
	}
	if next != "" {
		out.NextCursor = ptr.Wrap(next)
	}
	return out, nil
}

func (s *Storage) GetTagsBySource(ctx context.Context, sourceID string) ([]string, error) {
	docs, err := s.store.ListBy(ctx, "$.sourceId", sourceID)
	if err != nil {
		return nil, err
	}
	return slices.To(docs, func(d *versionDoc) string { return d.Tag }), nil
}

func (s *Storage) Exists(ctx context.Context, id string) (bool, error) {
	return s.store.Exists(ctx, id)
}

func (s *Storage) ExistsBySourceAndTag(ctx context.Context, sourceID, tag string) (bool, error) {
	d, err := s.docBySourceTag(ctx, sourceID, tag)
	if err != nil {
		return false, err
	}
	return d != nil, nil
}

func (s *Storage) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *Storage) DeleteBySource(ctx context.Context, sourceID string) (int64, error) {
	return s.store.DeleteBy(ctx, "$.sourceId", sourceID)
}

// docBySourceTag finds the version for (sourceId, tag), or nil.
func (s *Storage) docBySourceTag(ctx context.Context, sourceID, tag string) (*versionDoc, error) {
	docs, err := s.store.ListBy(ctx, "$.sourceId", sourceID)
	if err != nil {
		return nil, err
	}
	for _, d := range docs {
		if d.Tag == tag {
			return d, nil
		}
	}
	return nil, nil //nolint:nilnil // (nil, nil) is the documented "not found" result
}

var _ storage.Storage = (*Storage)(nil)
