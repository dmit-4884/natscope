// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package bbolt is the bbolt (doc-model) implementation of message-template
// storage. Each template is one JSON document; headers and wildcards are nested.
package bbolt

import (
	"context"

	"github.com/altessa-s/go-atlas/core/types/ptr"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	storage "github.com/dmit-4884/natscope/internal/storages/templates"
)

// Storage is the doc-model bbolt message-template store.
type Storage struct {
	store *bbstore.Store[templateDoc, *templateDoc]
}

// New opens the templates bucket (creating it if needed) and returns the store.
func New(ctx context.Context, db *bbstore.DB) (*Storage, error) {
	store, err := bbstore.Open[templateDoc, *templateDoc](ctx, db, bbstore.Spec{
		Bucket:   "message_templates",
		NotFound: errs.ErrMessageTemplateNotFound,
		SortBy:   bbstore.SortUpdated,
	})
	if err != nil {
		return nil, err
	}
	return &Storage{store: store}, nil
}

func (s *Storage) Save(ctx context.Context, in *entities.MessageTemplate) error {
	return s.store.Save(ctx, converter.Convert(in, &templateDoc{}, bbstore.Opts()...))
}

func (s *Storage) Update(ctx context.Context, in *entities.MessageTemplate) error {
	return s.store.Update(ctx, converter.Convert(in, &templateDoc{}, bbstore.Opts()...))
}

func (s *Storage) Get(ctx context.Context, id string) (*entities.MessageTemplate, error) {
	d, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return converter.Convert(d, &entities.MessageTemplate{}, bbstore.Opts()...), nil
}

func (s *Storage) List(
	ctx context.Context,
	in *entities.MessageTemplatesList,
) (*entities.List[entities.MessageTemplates], error) {
	docs, next, total, err := s.store.List(ctx, in.Cursor, in.GetLimit(), in.IncludeTotalCount)
	if err != nil {
		return nil, err
	}
	out := &entities.List[entities.MessageTemplates]{
		Items: bbstore.ToEntities[entities.MessageTemplate](docs),
		Total: total,
	}
	if next != "" {
		out.NextCursor = ptr.Wrap(next)
	}
	return out, nil
}

func (s *Storage) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *Storage) DeleteAll(ctx context.Context) (int64, error) {
	return s.store.DeleteAll(ctx)
}

var _ storage.Storage = (*Storage)(nil)
