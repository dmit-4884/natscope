// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package bbolt is the bbolt (doc-model) implementation of publish-history
// storage. One document per publish; listing filters by connection/stream.
package bbolt

import (
	"context"

	"github.com/altessa-s/go-atlas/core/types/ptr"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	storage "github.com/dmit-4884/natscope/internal/storages/history"
)

// Storage is the doc-model bbolt publish-history store.
type Storage struct {
	store *bbstore.Store[historyDoc, *historyDoc]
}

// New opens the publish_history bucket (creating it if needed) and returns the
// store.
func New(ctx context.Context, db *bbstore.DB) (*Storage, error) {
	store, err := bbstore.Open[historyDoc, *historyDoc](ctx, db, bbstore.Spec{
		Bucket: "publish_history",
		Indexes: []bbstore.Index{
			{Path: "connectionUrl"},
			{Path: "stream"},
		},
	})
	if err != nil {
		return nil, err
	}
	return &Storage{store: store}, nil
}

func (s *Storage) Save(ctx context.Context, in *entities.PublishHistory) error {
	return s.store.Save(ctx, converter.Convert(in, &historyDoc{}, bbstore.Opts()...))
}

func (s *Storage) List(
	ctx context.Context,
	in *entities.PublishHistoryList,
) (*entities.List[entities.PublishHistories], error) {
	var filters []bbstore.Filter
	if in.ConnectionURL != nil {
		filters = append(filters, bbstore.Filter{Path: "$.connectionUrl", Val: *in.ConnectionURL})
	}
	if in.Stream != nil {
		filters = append(filters, bbstore.Filter{Path: "$.stream", Val: *in.Stream})
	}
	docs, next, total, err := s.store.ListFiltered(ctx, filters, in.Cursor, in.GetLimit(), in.IncludeTotalCount)
	if err != nil {
		return nil, err
	}
	out := &entities.List[entities.PublishHistories]{
		Items: bbstore.ToEntities[entities.PublishHistory](docs),
		Total: total,
	}
	if next != "" {
		out.NextCursor = ptr.Wrap(next)
	}
	return out, nil
}

var _ storage.Storage = (*Storage)(nil)
