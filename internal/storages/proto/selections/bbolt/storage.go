// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package bbolt is the bbolt (doc-model) implementation of proto-selection
// storage. There is at most one selection per source.
package bbolt

import (
	"context"

	"github.com/altessa-s/go-atlas/core/types/ptr"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	storage "github.com/dmit-4884/natscope/internal/storages/proto/selections"
)

// Storage is the doc-model bbolt proto-selection store.
type Storage struct {
	store *bbstore.Store[selectionDoc, *selectionDoc]
}

// New opens the proto_selections bucket (creating it if needed) and returns the
// store.
func New(ctx context.Context, db *bbstore.DB) (*Storage, error) {
	store, err := bbstore.Open[selectionDoc, *selectionDoc](ctx, db, bbstore.Spec{
		Bucket:   "proto_selections",
		NotFound: errs.ErrProtoSelectionNotFound,
	})
	if err != nil {
		return nil, err
	}
	return &Storage{store: store}, nil
}

// Save upserts the selection for a source, reusing the existing row's id and
// created_at when one already exists.
func (s *Storage) Save(ctx context.Context, in *entities.ProtoSelection) error {
	return s.store.WithTransaction(ctx, func(ctx context.Context) error {
		d := converter.Convert(in, &selectionDoc{}, bbstore.Opts()...)
		existing, err := s.docBySource(ctx, in.SourceID)
		if err != nil {
			return err
		}
		if existing != nil {
			d.ID = existing.ID
			d.CreatedAt = existing.CreatedAt
		}
		return s.store.Upsert(ctx, d)
	})
}

func (s *Storage) Get(ctx context.Context, id string) (*entities.ProtoSelection, error) {
	d, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return converter.Convert(d, &entities.ProtoSelection{}, bbstore.Opts()...), nil
}

func (s *Storage) GetBySource(ctx context.Context, sourceID string) (*entities.ProtoSelection, error) {
	d, err := s.docBySource(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, errs.ErrProtoSelectionNotFound
	}
	return converter.Convert(d, &entities.ProtoSelection{}, bbstore.Opts()...), nil
}

func (s *Storage) List(
	ctx context.Context,
	in *entities.ProtoSelectionsList,
) (*entities.List[entities.ProtoSelections], error) {
	var filters []bbstore.Filter
	if in.SourceID != nil {
		filters = append(filters, bbstore.Filter{Path: "$.sourceId", Val: *in.SourceID})
	}
	docs, next, total, err := s.store.ListFiltered(ctx, filters, in.Cursor, in.GetLimit(), in.IncludeTotalCount)
	if err != nil {
		return nil, err
	}
	out := &entities.List[entities.ProtoSelections]{
		Items: bbstore.ToEntities[entities.ProtoSelection](docs),
		Total: total,
	}
	if next != "" {
		out.NextCursor = ptr.Wrap(next)
	}
	return out, nil
}

func (s *Storage) GetAll(ctx context.Context) (entities.ProtoSelections, error) {
	docs, err := s.store.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return bbstore.ToEntities[entities.ProtoSelection](docs), nil
}

func (s *Storage) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *Storage) DeleteBySource(ctx context.Context, sourceID string) (int64, error) {
	return s.store.DeleteBy(ctx, "$.sourceId", sourceID)
}

// docBySource returns the selection for a source, or nil.
func (s *Storage) docBySource(ctx context.Context, sourceID string) (*selectionDoc, error) {
	docs, err := s.store.ListBy(ctx, "$.sourceId", sourceID)
	if err != nil {
		return nil, err
	}
	if len(docs) > 0 {
		return docs[0], nil
	}
	return nil, nil //nolint:nilnil // (nil, nil) is the documented "not found" result
}

var _ storage.Storage = (*Storage)(nil)
